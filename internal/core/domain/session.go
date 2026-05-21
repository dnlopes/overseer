package domain

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionKind discriminates how a session relates to its branch:
//
//   - SessionKindFeature (the default, also the JSON zero value) is a regular
//     feature session: FeatureBranch is a NEW branch forked from BaseBranch
//     and the user is expected to push it / open a PR.
//   - SessionKindCheckout is a check / draft session created by checking out
//     an existing branch: FeatureBranch is a local branch whose upstream is
//     set to origin/<BaseBranch>, so the user can `git pull` to follow the
//     tracked branch and commit local drafts without disturbing it.
//
// Persisted as a string so the data file stays human-readable; the empty
// string deserialises to SessionKindFeature for backward compatibility with
// records created before this discriminator existed. The storage layer
// translates the legacy value "open-branch" to "checkout" when loading.
type SessionKind string

const (
	SessionKindFeature  SessionKind = ""
	SessionKindCheckout SessionKind = "checkout"
)

// Session is the aggregate representing a single AI agent session.
//
// Every session is bound to a Project (its git repository) — ProjectID is
// required and rejected when uuid.Nil. The session's worktree lives at
// WorktreePath; BaseBranch and FeatureBranch are populated together with
// WorktreePath via AssignWorktree before the session is persisted.
//
// The Kind field discriminates between feature sessions (FeatureBranch is a
// brand-new branch forked from BaseBranch) and checkout sessions
// (FeatureBranch is a local branch tracking origin/<BaseBranch>). The
// worktree layout itself is identical in both cases.
type Session struct {
	ID            uuid.UUID
	Name          string
	ProjectID     uuid.UUID
	Kind          SessionKind `json:",omitempty"`
	Order         int
	WorktreePath  string
	BaseBranch    string
	FeatureBranch string
	AgentCommand  string
	EditorCommand string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewSession constructs a feature Session with no worktree assigned. Callers
// must follow up with AssignWorktree before persisting; ProjectID is
// required and uuid.Nil is rejected.
func NewSession(name string, projectID uuid.UUID) (Session, error) {
	return newSession(name, projectID, SessionKindFeature)
}

// NewCheckoutSession constructs a checkout Session with no worktree
// assigned. Checkout sessions exist to inspect or draft on top of an
// existing branch (typically the default branch). Callers must follow up
// with AssignWorktree before persisting; ProjectID is required and uuid.Nil
// is rejected.
func NewCheckoutSession(name string, projectID uuid.UUID) (Session, error) {
	return newSession(name, projectID, SessionKindCheckout)
}

func newSession(name string, projectID uuid.UUID, kind SessionKind) (Session, error) {
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
		Kind:      kind,
		Order:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsCheckout reports whether this is a checkout (tracking) session.
func (s Session) IsCheckout() bool {
	return s.Kind == SessionKindCheckout
}

// HasWorktree reports whether the session has a worktree assigned.
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

// AssignWorktree populates the worktree ensemble (path + base branch +
// feature branch). All three must be non-empty and the path must be
// absolute; partial assignment is rejected to preserve the all-or-none
// invariant. The same call shape is used for both feature and checkout
// sessions — the discriminator lives in Session.Kind, not in this method.
func (s *Session) AssignWorktree(worktreePath, baseBranch, featureBranch string) error {
	worktreePath = strings.TrimSpace(worktreePath)
	baseBranch = strings.TrimSpace(baseBranch)
	featureBranch = strings.TrimSpace(featureBranch)

	if worktreePath == "" || baseBranch == "" || featureBranch == "" {
		return ErrSessionWorktreeFieldsMismatch
	}
	if !filepath.IsAbs(worktreePath) {
		return ErrSessionWorktreePathNotAbsolute
	}

	s.WorktreePath = worktreePath
	s.BaseBranch = baseBranch
	s.FeatureBranch = featureBranch
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
	// CreateTrackingWorktree creates a new git worktree at worktreePath whose
	// local branch (localBranch) tracks origin/<remoteBranch>. Used for
	// checkout sessions so a plain `git pull` follows the upstream branch
	// while still letting the user commit local drafts that never collide
	// with the tracked branch checked out elsewhere.
	CreateTrackingWorktree(ctx context.Context, repoPath, remoteBranch, localBranch, worktreePath string) error
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
}

// Session sentinel errors.
var (
	ErrSessionEmptyName                = errors.New("session name cannot be empty")
	ErrSessionNameTooLong              = errors.New("session name exceeds 100 characters")
	ErrSessionEmptyProjectID           = errors.New("session project id cannot be empty")
	ErrSessionEmptyBaseBranch          = errors.New("session base branch cannot be empty")
	ErrSessionNotFound                 = errors.New("session not found")
	ErrSessionAlreadyExists            = errors.New("session already exists")
	ErrSessionWorktreeFieldsMismatch   = errors.New("session worktree fields must all be set")
	ErrSessionWorktreePathNotAbsolute  = errors.New("session worktree path must be absolute")
	ErrSessionWorktreePathOutsideRoot  = errors.New("session worktree path is outside the managed worktree root")
	ErrSessionEmptyAgentCommand        = errors.New("session agent command cannot be empty")
	ErrSessionNoAgentCommandAvailable  = errors.New("session has no agent command and no default launcher is configured")
	ErrSessionEmptyEditorCommand       = errors.New("session editor command cannot be empty")
	ErrSessionNoEditorCommandAvailable = errors.New("session has no editor command and no default editor is configured")
)
