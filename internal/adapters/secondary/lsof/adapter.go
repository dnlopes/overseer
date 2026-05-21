// Package lsof discovers which session-store file an agent process is
// currently writing to, by walking the descendant process tree of a tmux
// pane (or any rootPID) and inspecting open file descriptors via `lsof`.
//
// This package is a low-level infrastructure helper, NOT a domain-port
// implementation. Downstream adapters (claudejsonl, opencodesqlite) and
// the composite AgentActivityProvider depend on Adapter to locate the
// per-session file they parse.
package lsof

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// AgentKind classifies the agent backend by the file format it writes.
// Internal to the lsof layer; downstream callers translate it into domain
// types as needed.
type AgentKind string

const (
	AgentKindClaude   AgentKind = "claude"
	AgentKindOpenCode AgentKind = "opencode"
)

// Resolution is the outcome of a successful agent-session discovery.
type Resolution struct {
	// PID of the descendant process that has the session file open.
	PID int
	// Path is the absolute, symlink-resolved path of the session file.
	Path string
	// Kind classifies the file format (Claude JSONL or OpenCode SQLite).
	Kind AgentKind
}

// Sentinel errors returned by ResolveAgentSession. Callers use errors.Is
// to distinguish them from infrastructure failures (which are wrapped).
var (
	// ErrNoDescendants reports that rootPID has zero live descendants.
	// Typical cause: the tmux pane shell has not yet forked an agent process.
	// Callers should treat this as a transient, retry-friendly state.
	ErrNoDescendants = errors.New("lsof: rootPID has no descendant processes")

	// ErrNoAgentStore reports that descendants exist but none currently
	// holds an agent session-store file open (`*.jsonl` or `*.db`).
	// Typical cause: the agent has not yet performed its first write, or
	// (for Claude on macOS) the agent has closed the file between writes.
	// Callers should retry; this is mapped to domain.ErrAgentStoreNotResolved
	// upstream.
	ErrNoAgentStore = errors.New("lsof: no descendant holds an agent session store open")
)

// Adapter wraps the `lsof` and `ps` binaries to discover an agent process
// and the session-store file it has open.
type Adapter struct {
	lsofBin string
	psBin   string
	logger  *slog.Logger
}

// New constructs an Adapter using the `lsof` and `ps` binaries discovered
// on PATH. Returns an error if either binary is missing.
func New(logger *slog.Logger) (*Adapter, error) {
	lsofPath, err := exec.LookPath("lsof")
	if err != nil {
		return nil, fmt.Errorf("lsof: not found on PATH: %w", err)
	}
	psPath, err := exec.LookPath("ps")
	if err != nil {
		return nil, fmt.Errorf("ps: not found on PATH: %w", err)
	}
	return &Adapter{lsofBin: lsofPath, psBin: psPath, logger: logger}, nil
}

// ResolveAgentSession walks the descendant process tree of rootPID and
// returns the first descendant that has an agent session-store file open.
//
// Discovery rules (in order):
//  1. Use `ps -A -o pid=,ppid=` to build the process-tree once per call.
//  2. BFS from rootPID; the root itself is NOT inspected (it is typically
//     a shell, never the agent).
//  3. For each descendant in BFS order, list its open paths via
//     `lsof -p <pid> -F n` and pick the first path whose suffix matches
//     `*.jsonl` (Claude) or `*.db` (OpenCode). SQLite auxiliary files
//     (`-wal`, `-shm`) are deliberately excluded.
//
// Returns ErrNoDescendants if rootPID has no live children, or
// ErrNoAgentStore if children exist but none holds a relevant file open.
func (a *Adapter) ResolveAgentSession(ctx context.Context, rootPID int) (Resolution, error) {
	descendants, err := a.descendants(ctx, rootPID)
	if err != nil {
		return Resolution{}, fmt.Errorf("lsof: enumerate descendants of %d: %w", rootPID, err)
	}
	if len(descendants) == 0 {
		return Resolution{}, ErrNoDescendants
	}

	for _, pid := range descendants {
		if err := ctx.Err(); err != nil {
			return Resolution{}, err
		}
		paths, err := a.openPaths(ctx, pid)
		if err != nil {
			a.logger.Debug("lsof: open paths failed for descendant, continuing",
				"pid", pid, "error", err)
			continue
		}
		for _, p := range paths {
			if kind, ok := classify(p); ok {
				return Resolution{PID: pid, Path: p, Kind: kind}, nil
			}
		}
	}
	return Resolution{}, ErrNoAgentStore
}

// descendants returns the recursive descendants of rootPID in BFS order,
// excluding rootPID itself. Implemented via a single `ps` invocation.
func (a *Adapter) descendants(ctx context.Context, rootPID int) ([]int, error) {
	out, err := exec.CommandContext(ctx, a.psBin, "-A", "-o", "pid=,ppid=").Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("ps: %w", err)
	}

	children := make(map[int][]int)
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}

	visited := map[int]bool{rootPID: true}
	queue := []int{rootPID}
	var result []int
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range children[cur] {
			if visited[child] {
				continue
			}
			visited[child] = true
			result = append(result, child)
			queue = append(queue, child)
		}
	}
	return result, nil
}

// openPaths returns the absolute file paths currently held open by pid, as
// reported by `lsof -p <pid> -F n`. Non-file entries (sockets, pipes,
// kqueue descriptors) are filtered out because lsof prefixes them with
// something other than a leading slash. Process-not-found is treated as
// "no paths" rather than an error so the caller can keep walking siblings.
func (a *Adapter) openPaths(ctx context.Context, pid int) ([]string, error) {
	out, err := exec.CommandContext(ctx, a.lsofBin, "-p", strconv.Itoa(pid), "-F", "n").Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof -p %d: %w", pid, err)
	}

	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "n/") {
			continue
		}
		paths = append(paths, line[1:])
	}
	return paths, nil
}

// classify maps an open file path to an AgentKind based on its suffix.
// `.db-wal` and `.db-shm` are SQLite auxiliary files that exist whenever
// the main `.db` is open in WAL mode; they would otherwise produce false
// OpenCode matches when the main DB is held by a sibling process or has
// rotated.
func classify(path string) (AgentKind, bool) {
	switch {
	case strings.HasSuffix(path, ".jsonl"):
		return AgentKindClaude, true
	case strings.HasSuffix(path, ".db-wal"), strings.HasSuffix(path, ".db-shm"):
		return "", false
	case strings.HasSuffix(path, ".db"):
		return AgentKindOpenCode, true
	default:
		return "", false
	}
}
