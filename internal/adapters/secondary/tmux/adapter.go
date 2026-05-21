package tmux

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dnlopes/overseer/internal/core/domain"
)

var _ domain.TmuxAdapter = (*Adapter)(nil)

// fieldSep is ASCII Unit Separator (0x1f). It cannot appear in tmux epoch
// timestamps and is not produced by any well-formed overseer session name,
// so it is a safe delimiter for our `-F` format output.
const fieldSep = "\x1f"

const sessionFormat = "#{session_name}" + fieldSep + "#{session_created}" + fieldSep + "#{session_activity}"

// Adapter drives the local tmux server by invoking the tmux binary directly.
// The tmuxID exchanged with the domain is the tmux session name.
type Adapter struct {
	tmuxBin string
	logger  *slog.Logger
}

// New constructs an Adapter using the `tmux` binary discovered on PATH.
// Returns an error if tmux is not installed on the system.
func New(logger *slog.Logger) (*Adapter, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux: not found on PATH: %w", err)
	}
	return &Adapter{tmuxBin: path, logger: logger}, nil
}

// CreateSession creates a detached tmux session with the given name and returns it as the tmuxID.
// An empty startDir lets tmux use the caller's working directory; an empty shellCommand
// lets tmux launch the user's default shell.
func (a *Adapter) CreateSession(_ context.Context, name, startDir, shellCommand string) (string, error) {
	args := []string{"new-session", "-d", "-s", name}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if shellCommand != "" {
		args = append(args, shellCommand)
	}

	if _, err := a.run(args...); err != nil {
		return "", fmt.Errorf("tmux: new session %q: %w", name, err)
	}
	a.logger.Debug("tmux session created", "name", name, "start_dir", startDir, "shell_command", shellCommand)
	return name, nil
}

// KillSession kills the tmux session identified by tmuxID (the session name).
func (a *Adapter) KillSession(_ context.Context, tmuxID string) error {
	if _, err := a.run("kill-session", "-t", tmuxID); err != nil {
		return fmt.Errorf("tmux: kill session %q: %w", tmuxID, err)
	}
	a.logger.Debug("tmux session killed", "name", tmuxID)
	return nil
}

// GetSession returns the tmux session identified by tmuxID (the session name).
func (a *Adapter) GetSession(_ context.Context, tmuxID string) (domain.TmuxSession, error) {
	stdout, err := a.run("display-message", "-p", "-t", tmuxID, "-F", sessionFormat)
	if err != nil {
		return domain.TmuxSession{}, fmt.Errorf("tmux: get session %q: %w", tmuxID, err)
	}
	line := strings.TrimRight(stdout, "\n")
	sess, ok := parseSessionLine(line)
	if !ok {
		return domain.TmuxSession{}, fmt.Errorf("tmux: get session %q: malformed output %q", tmuxID, line)
	}
	// tmux display-message silently falls back to the "current" session when the
	// requested -t target does not exist, so we must verify the name ourselves.
	if sess.ID != tmuxID {
		return domain.TmuxSession{}, fmt.Errorf("tmux: get session %q: %w", tmuxID, domain.ErrTmuxSessionNotFound)
	}
	return sess, nil
}

// ListSessions returns every tmux session known to the local tmux server.
// An empty slice (not an error) is returned when no server is running.
func (a *Adapter) ListSessions(_ context.Context) ([]domain.TmuxSession, error) {
	stdout, err := a.run("list-sessions", "-F", sessionFormat)
	if err != nil {
		if errors.Is(err, errTmuxNoServer) {
			return []domain.TmuxSession{}, nil
		}
		return nil, fmt.Errorf("tmux: list sessions: %w", err)
	}
	out := []domain.TmuxSession{}
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		sess, ok := parseSessionLine(line)
		if !ok {
			a.logger.Warn("tmux: list-sessions malformed line", "line", line)
			continue
		}
		out = append(out, sess)
	}
	return out, nil
}

// AttachSession attaches the current terminal to the named tmux session.
// The call blocks until the user detaches; callers running inside an alt-screen TUI
// must release the terminal (e.g. via tea.ExecProcess) before invoking this method.
func (a *Adapter) AttachSession(_ context.Context, tmuxID string) error {
	cmd := exec.Command(a.tmuxBin, "attach-session", "-t", tmuxID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux: attach session %q: %w", tmuxID, err)
	}
	return nil
}

// AttachCommand builds (but does not run) a `tmux attach-session` command for tmuxID.
// Stdin/stdout/stderr are pinned to the process's own os.Stdin/Stdout/Stderr so the
// child tmux client inherits the controlling terminal directly. tea.ExecProcess only
// fills these in when they are nil, so pre-setting them keeps bubbletea from substituting
// its own /dev/tty open — which tmux rejects with "can't use /dev/tty" because it isn't
// the controlling TTY of the spawned process.
func (a *Adapter) AttachCommand(_ context.Context, tmuxID string) (*exec.Cmd, error) {
	cmd := exec.Command(a.tmuxBin, "attach-session", "-t", tmuxID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// CapturePane returns the current visible content of the active pane of the
// named tmux session. The `-e` flag preserves ANSI escape sequences so callers
// can render the output with its original colors. "can't find session" stderr
// is mapped to domain.ErrTmuxSessionNotFound by the run helper.
func (a *Adapter) CapturePane(_ context.Context, tmuxID string) (string, error) {
	stdout, err := a.run("capture-pane", "-p", "-e", "-t", "="+tmuxID+":")
	if err != nil {
		return "", fmt.Errorf("tmux: capture pane %q: %w", tmuxID, err)
	}
	return stdout, nil
}

// ResizeWindow resizes the named tmux session's window to width×height cells.
// tmux sends SIGWINCH to the foreground process so the terminal UI redraws on
// the new canvas; without this, detached agent sessions stay frozen at tmux's
// default size and render small/collapsed layouts inside a wider preview.
//
// resize-window with explicit -x/-y silently switches window-size to "manual",
// which would freeze the pane at this preview size even when a larger client
// later attaches — the user sees their app boxed inside dotted padding. We
// immediately restore "latest" so attaching clients can grow the window to
// fit their terminal. Empirically the pane stays at the size we just set
// until the next attach or resize, which is exactly what we want.
func (a *Adapter) ResizeWindow(_ context.Context, tmuxID string, width, height int) error {
	target := "=" + tmuxID + ":"
	if _, err := a.run("resize-window", "-t", target, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height)); err != nil {
		return fmt.Errorf("tmux: resize window %q to %dx%d: %w", tmuxID, width, height, err)
	}
	if _, err := a.run("set-window-option", "-t", target, "window-size", "latest"); err != nil {
		return fmt.Errorf("tmux: restore window-size on %q: %w", tmuxID, err)
	}
	return nil
}

// GetPanePID returns the operating-system PID of the foreground process in
// the active pane of the named tmux session. Used by the agent-activity
// adapters as the root for the lsof descendant walk.
//
// This method is NOT part of domain.TmuxAdapter — it is a private adapter
// capability consumed by other secondary adapters (claudejsonl,
// opencodesqlite) via duck-typed interfaces. Keeping it off the port avoids
// polluting the domain contract with an introspection method that no use
// case directly needs.
func (a *Adapter) GetPanePID(_ context.Context, tmuxID string) (int, error) {
	stdout, err := a.run("list-panes", "-t", tmuxID, "-F", "#{pane_pid}")
	if err != nil {
		return 0, fmt.Errorf("tmux: get pane pid for %q: %w", tmuxID, err)
	}
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		raw := strings.TrimSpace(line)
		if raw == "" {
			continue
		}
		pid, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return 0, fmt.Errorf("tmux: parse pane pid %q for %q: %w", raw, tmuxID, parseErr)
		}
		return pid, nil
	}
	return 0, fmt.Errorf("tmux: no panes reported for %q: %w", tmuxID, domain.ErrTmuxSessionNotFound)
}

// errTmuxNoServer is returned by run when tmux reports that no server is running.
var errTmuxNoServer = errors.New("tmux: no server running")

// run executes the tmux binary with the given arguments, returning stdout.
// "can't find session" stderr is mapped to domain.ErrTmuxSessionNotFound; "no server
// running" is mapped to errTmuxNoServer so callers can distinguish empty state from failure.
func (a *Adapter) run(args ...string) (string, error) {
	cmd := exec.Command(a.tmuxBin, args...)
	stdout, err := cmd.Output()
	if err == nil {
		return string(stdout), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		switch {
		case strings.Contains(stderr, "can't find session"),
			strings.Contains(stderr, "can't find pane"),
			strings.Contains(stderr, "can't find window"):
			return "", domain.ErrTmuxSessionNotFound
		case strings.Contains(stderr, "no server running"):
			return "", errTmuxNoServer
		default:
			return "", fmt.Errorf("%w: %s", err, stderr)
		}
	}
	return "", err
}

func parseSessionLine(line string) (domain.TmuxSession, bool) {
	parts := strings.Split(line, fieldSep)
	if len(parts) != 3 {
		return domain.TmuxSession{}, false
	}
	return domain.TmuxSession{
		ID:        parts[0],
		CreatedAt: parseTmuxEpoch(parts[1]),
		UpdatedAt: parseTmuxEpoch(parts[2]),
	}, true
}

// parseTmuxEpoch interprets a tmux #{session_created}/#{session_activity} field, which tmux
// emits as a Unix-epoch second count, returning the zero time on parse failure.
func parseTmuxEpoch(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}
