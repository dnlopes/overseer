package domain

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

type TmuxSession struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TmuxAdapter interface {
	GetSession(ctx context.Context, tmuxID string) (TmuxSession, error)
	ListSessions(ctx context.Context) ([]TmuxSession, error)
	CreateSession(ctx context.Context, name, startDir, shellCommand string) (tmuxID string, err error)
	KillSession(ctx context.Context, tmuxID string) error
	AttachSession(ctx context.Context, tmuxID string) error
	// AttachCommand returns a runnable *exec.Cmd that attaches the caller's
	// terminal to the named tmux session when executed. It does NOT take over
	// the terminal itself — the caller is responsible for running it (typically
	// via tea.ExecProcess from inside a Bubble Tea alt-screen TUI).
	AttachCommand(ctx context.Context, tmuxID string) (*exec.Cmd, error)
	// CapturePane returns the current visible content of the named tmux
	// session's active pane, with ANSI escape sequences preserved so callers
	// can render colored output. Returns ErrTmuxSessionNotFound if the session
	// does not exist.
	CapturePane(ctx context.Context, tmuxID string) (string, error)
	// ResizeWindow resizes the named tmux session's window to the given
	// dimensions in cells. The running terminal application receives SIGWINCH
	// and redraws on the new canvas. Returns ErrTmuxSessionNotFound if the
	// session does not exist.
	ResizeWindow(ctx context.Context, tmuxID string, width, height int) error
}

var ErrTmuxSessionNotFound = errors.New("tmux session not found")
