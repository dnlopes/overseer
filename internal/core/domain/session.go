package domain

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Session is the aggregate representing a single AI agent session.
//
// Sessions come in two flavours, disambiguated purely by whether a worktree
// is assigned (HasWorktree):
//
//   - Worktree session (Mode 1): WorktreePath and Branch are populated; the
//     session lives inside an isolated git worktree forked from a base
//     branch.
//   - Project session (Mode 2): WorktreePath and Branch are both empty; the
//     session is attached directly to the project's working directory and
//     the "branch" is whatever the project's HEAD currently points at — read
//     live by the UI rather than persisted.
//
// Every session is bound to a Project — ProjectID is required and rejected
// when uuid.Nil. The two modes share every other field; the storage layer
// persists the same struct shape for both.
type Session struct {
	ID            uuid.UUID
	Name          string
	ProjectID     uuid.UUID
	Order         int
	Branch        string
	WorktreePath  string
	AgentCommand  string
	EditorCommand string
	AgentType     AgentType `json:",omitempty"`
	Label         string    `json:",omitempty"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewSession constructs a Session with no worktree assigned. For a Mode 1
// (worktree) session, callers must follow up with AssignWorktree before
// persisting. For a Mode 2 (project) session, callers persist the Session
// as-is — HasWorktree will remain false. ProjectID is required; uuid.Nil
// is rejected.
func NewSession(name string, projectID uuid.UUID) (Session, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return Session{}, ErrSessionEmptyName
	}
	if len(name) > 100 {
		return Session{}, ErrSessionNameTooLong
	}
	if projectID == uuid.Nil {
		return Session{}, ErrSessionEmptyProjectID
	}

	now := time.Now()
	return Session{
		ID:        uuid.New(),
		Name:      name,
		ProjectID: projectID,
		Order:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// HasWorktree reports whether the session is a Mode 1 (worktree) session.
// Mode 2 (project) sessions return false; their working directory lives at
// the project's path.
func (s Session) HasWorktree() bool {
	return s.WorktreePath != ""
}

// WorktreeIsInsideRoot reports whether the session's WorktreePath lies
// inside the supplied root directory. It is the safety guard that protects
// destructive operations (worktree removal) from acting on paths outside
// the managed worktree root — even if the persisted Session row has been
// hand-edited or corrupted.
//
// The check uses filepath.Rel so siblings of the root that share a textual
// prefix (e.g. "/data/worktrees-evil" vs "/data/worktrees") are rejected.
// A session without a worktree is trivially "inside" — callers should gate
// this check with HasWorktree.
func (s Session) WorktreeIsInsideRoot(root string) bool {
	if s.WorktreePath == "" {
		return true
	}
	root = strings.TrimSpace(root)
	if root == "" || !filepath.IsAbs(root) {
		return false
	}
	rel, err := filepath.Rel(root, s.WorktreePath)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (s *Session) Rename(newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return ErrSessionEmptyName
	}
	if len(newName) > 100 {
		return ErrSessionNameTooLong
	}

	s.Name = newName
	s.UpdatedAt = time.Now()
	return nil
}

// AssignWorktree populates the worktree fields (path + branch). Both must be
// non-empty and the path must be absolute; partial assignment is rejected to
// preserve the all-or-none invariant. Only Mode 1 sessions call this; Mode 2
// sessions skip it entirely.
func (s *Session) AssignWorktree(worktreePath, branch string) error {
	worktreePath = strings.TrimSpace(worktreePath)
	branch = strings.TrimSpace(branch)

	if worktreePath == "" || branch == "" {
		return ErrSessionWorktreeFieldsMismatch
	}
	if !filepath.IsAbs(worktreePath) {
		return ErrSessionWorktreePathNotAbsolute
	}

	s.WorktreePath = worktreePath
	s.Branch = branch
	s.UpdatedAt = time.Now()
	return nil
}

// AssignAgentCommand sets the raw shell command used to launch this
// session's agent program (e.g. "opencode", "claude --foo"). The command
// must be non-empty after trimming; empty values are rejected so the
// invariant "if AgentCommand is set, it is runnable" holds.
func (s *Session) AssignAgentCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ErrSessionEmptyAgentCommand
	}

	s.AgentCommand = cmd
	s.UpdatedAt = time.Now()
	return nil
}

// AssignEditorCommand sets the raw shell command used to open this
// session's worktree in an editor (e.g. "code", "cursor --wait", "nvim").
// The command must be non-empty after trimming; empty values are rejected
// so the invariant "if EditorCommand is set, it is runnable" holds.
func (s *Session) AssignEditorCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ErrSessionEmptyEditorCommand
	}

	s.EditorCommand = cmd
	s.UpdatedAt = time.Now()
	return nil
}

// AssignAgentType sets the session's agent type — the discriminator the
// status-detection registry uses to route a session to the right detector.
// Empty values are rejected so the invariant "if AgentType is set, it is
// routable" holds; the storage migration handles legacy sessions by
// inferring or falling back to AgentTypeUnknown, which is still routable
// (just resolves to no-detector → Unknown status).
func (s *Session) AssignAgentType(t AgentType) error {
	if t == "" {
		return ErrAgentTypeRequired
	}
	s.AgentType = t
	s.UpdatedAt = time.Now()
	return nil
}

// AssignLabel sets the session's status label code (e.g. "WIP", "done").
// The empty string is accepted and clears the label — pressing the l
// shortcut past the last label cycles back through the empty state.
// Codes longer than 50 characters are rejected to guard against
// corrupted or hand-edited storage; valid codes are validated upstream
// at config load (see Label / NewLabel).
func (s *Session) AssignLabel(code string) error {
	code = strings.TrimSpace(code)
	if len(code) > labelCodeMaxLen {
		return ErrSessionLabelTooLong
	}
	s.Label = code
	s.UpdatedAt = time.Now()
	return nil
}

// BranchScope discriminates whether a BranchInfo points at a local branch
// (refs/heads/...) or a remote-tracking branch (refs/remotes/...).
type BranchScope int

const (
	BranchScopeLocal BranchScope = iota
	BranchScopeRemote
)

// BranchInfo describes a single git branch surfaced by GitAdapter.ListBranches.
// The TUI's branch picker renders these directly, so the struct is shaped for
// display: Name is short ("main", "origin/feat/foo"), Scope drives the
// glyph, CommitterDate drives the "age" column.
type BranchInfo struct {
	Name          string
	Scope         BranchScope
	CommitterDate time.Time
}

// Session ports.

type SessionRepository interface {
	Save(ctx context.Context, s Session) error
	Get(ctx context.Context, id uuid.UUID) (Session, error)
	List(ctx context.Context) ([]Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type GitAdapter interface {
	// CreateWorktree creates a new git worktree at worktreePath, forked from
	// baseBranch inside repoPath, on a new branch named featureBranch.
	CreateWorktree(ctx context.Context, repoPath, baseBranch, featureBranch, worktreePath string) error
	// RemoveWorktree removes the worktree at worktreePath from the repository
	// rooted at repoPath. Implementations may force-remove uncommitted changes.
	RemoveWorktree(ctx context.Context, repoPath, worktreePath string) error
	// IsGitRepo reports whether path is the root of a git working tree.
	IsGitRepo(ctx context.Context, path string) error
	// GetDefaultBranch resolves the repository's default branch — typically
	// the branch HEAD on origin points at, with a "main"/"master" fallback
	// for repos without an origin remote. Returns ErrProjectNoDefaultBranch
	// when neither signal is available.
	GetDefaultBranch(ctx context.Context, repoPath string) (string, error)
	// ListBranches enumerates the local and remote-tracking branches in
	// repoPath. Order is implementation-defined; consumers sort as needed.
	// The HEAD symbolic ref is omitted.
	ListBranches(ctx context.Context, repoPath string) ([]BranchInfo, error)
	// CurrentBranch reads the branch HEAD currently points at inside
	// repoPath. Used by Mode 2 sessions to surface the project's live
	// branch without persisting it.
	CurrentBranch(ctx context.Context, repoPath string) (string, error)
}

// Session sentinel errors.
var (
	ErrSessionEmptyName                = errors.New("session name cannot be empty")
	ErrSessionNameTooLong              = errors.New("session name exceeds 100 characters")
	ErrSessionEmptyProjectID           = errors.New("session project id cannot be empty")
	ErrSessionNotFound                 = errors.New("session not found")
	ErrSessionAlreadyExists            = errors.New("session already exists")
	ErrSessionWorktreeFieldsMismatch   = errors.New("session worktree fields must all be set")
	ErrSessionWorktreePathNotAbsolute  = errors.New("session worktree path must be absolute")
	ErrSessionWorktreePathOutsideRoot  = errors.New("session worktree path is outside the managed worktree root")
	ErrSessionEmptyAgentCommand        = errors.New("session agent command cannot be empty")
	ErrSessionNoAgentCommandAvailable  = errors.New("session has no agent command and no default launcher is configured")
	ErrSessionEmptyEditorCommand       = errors.New("session editor command cannot be empty")
	ErrSessionNoEditorCommandAvailable = errors.New("session has no editor command and no default editor is configured")
	ErrSessionLabelTooLong             = errors.New("session label exceeds 50 characters")
)
