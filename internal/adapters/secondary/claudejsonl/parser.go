// Package claudejsonl implements domain.AgentActivityProvider for Claude Code
// agent sessions by observing the JSONL session-store file Claude writes to
// `~/.claude/projects/<encoded-cwd>/<session-uuid>.jsonl`.
//
// The provider follows the "observe-then-cache" strategy mandated by the
// project's HANDOFF: discovery via lsof is best-effort (Claude on macOS does
// NOT keep its JSONL file open between writes), so once a path is observed
// it is cached and used as the source of truth for the session's lifetime.
// No path prediction is performed at any layer — `CLAUDE_CONFIG_DIR`,
// custom launchers, direnv, and similar env tweaks all work transparently
// because the parser learns the path from a live FD, not from a guess.
package claudejsonl

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/secondary/lsof"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// TmuxPaneResolver returns the OS PID of the foreground process in the
// active pane of a tmux session. Implemented by tmux.Adapter.GetPanePID;
// defined here so the parser can be exercised with a fake.
type TmuxPaneResolver interface {
	GetPanePID(ctx context.Context, tmuxID string) (int, error)
}

// AgentResolver discovers the agent session-store file by walking the
// descendant process tree of a root PID. Implemented by lsof.Adapter.
type AgentResolver interface {
	ResolveAgentSession(ctx context.Context, rootPID int) (lsof.Resolution, error)
}

// Parser implements domain.AgentActivityProvider for Claude Code.
type Parser struct {
	tmux    TmuxPaneResolver
	agent   AgentResolver
	logger  *slog.Logger
	nowFunc func() time.Time

	mu    sync.Mutex
	cache map[uuid.UUID]string
}

var _ domain.AgentActivityProvider = (*Parser)(nil)

// New constructs a Parser bound to the given tmux pane resolver and lsof
// agent resolver. Both dependencies are required.
func New(tmuxResolver TmuxPaneResolver, agent AgentResolver, logger *slog.Logger) *Parser {
	return &Parser{
		tmux:    tmuxResolver,
		agent:   agent,
		logger:  logger,
		nowFunc: time.Now,
		cache:   make(map[uuid.UUID]string),
	}
}

// Observe returns the current activity for the referenced Claude session.
//
// Resolution proceeds in three stages:
//
//  1. Resolve the agent's root PID via the tmux pane resolver. A missing
//     tmux session is mapped to domain.ErrAgentNotRunning.
//  2. Attempt to discover the JSONL path via the lsof descendant walk.
//     - On hit: validate the kind is Claude and refresh the per-session
//     cache.
//     - On ErrNoDescendants: surface domain.ErrAgentNotRunning regardless
//     of cache (the agent process is gone; whatever path we have is
//     stale by definition).
//     - On ErrNoAgentStore: fall back to the cached path if any;
//     otherwise return domain.ErrAgentStoreNotResolved so the scheduler
//     retries on the next tick.
//  3. Stat the resolved path. A missing file invalidates the cache and
//     returns domain.ErrAgentStoreNotResolved. Otherwise scan the JSONL
//     and run the two-step resolver (determineStatus → resolveActivity)
//     against the current wall clock.
func (p *Parser) Observe(ctx context.Context, ref domain.AgentSessionRef) (domain.AgentActivity, error) {
	rootPID, err := p.tmux.GetPanePID(ctx, ref.AgentTmuxSessionID)
	if err != nil {
		if errors.Is(err, domain.ErrTmuxSessionNotFound) {
			return domain.AgentActivity{}, domain.ErrAgentNotRunning
		}
		return domain.AgentActivity{}, fmt.Errorf("claudejsonl: resolve tmux pane pid for %q: %w", ref.AgentTmuxSessionID, err)
	}

	path, err := p.resolvePath(ctx, ref, rootPID)
	if err != nil {
		return domain.AgentActivity{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			p.invalidate(ref.SessionID)
			p.logger.Debug("claudejsonl: cached store no longer exists, invalidating",
				"session_id", ref.SessionID, "path", path)
			return domain.AgentActivity{}, domain.ErrAgentStoreNotResolved
		}
		return domain.AgentActivity{}, fmt.Errorf("claudejsonl: stat %q: %w", path, err)
	}

	result, err := scanJSONL(path)
	if err != nil {
		return domain.AgentActivity{}, fmt.Errorf("claudejsonl: scan %q: %w", path, err)
	}
	if result.LastActivity.IsZero() {
		result.LastActivity = info.ModTime()
	}

	kind, tool := resolveActivity(result, p.nowFunc())
	return domain.NewAgentActivity(ref.SessionID, kind, tool)
}

func (p *Parser) resolvePath(ctx context.Context, ref domain.AgentSessionRef, rootPID int) (string, error) {
	res, err := p.agent.ResolveAgentSession(ctx, rootPID)
	switch {
	case err == nil:
		if res.Kind != lsof.AgentKindClaude {
			return "", fmt.Errorf("claudejsonl: unexpected agent kind %q for session %s (parser only handles Claude)", res.Kind, ref.SessionID)
		}
		p.store(ref.SessionID, res.Path)
		return res.Path, nil
	case errors.Is(err, lsof.ErrNoDescendants):
		return "", domain.ErrAgentNotRunning
	case errors.Is(err, lsof.ErrNoAgentStore):
		if cached, ok := p.get(ref.SessionID); ok {
			return cached, nil
		}
		return "", domain.ErrAgentStoreNotResolved
	default:
		return "", fmt.Errorf("claudejsonl: resolve agent session for pid %d: %w", rootPID, err)
	}
}

func (p *Parser) store(id uuid.UUID, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if prev, ok := p.cache[id]; ok && prev != path {
		p.logger.Debug("claudejsonl: cached path changed",
			"session_id", id, "previous", prev, "current", path)
	}
	p.cache[id] = path
}

func (p *Parser) get(id uuid.UUID) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.cache[id]
	return v, ok
}

func (p *Parser) invalidate(id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cache, id)
}
