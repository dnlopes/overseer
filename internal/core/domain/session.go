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
// ProjectID is uuid.Nil when the session is not associated with any project.
//
// Worktree fields (WorktreePath, BaseBranch, FeatureBranch) are an ensemble:
// they are either all set (project-backed sessions, populated via
// AssignWorktree) or all empty (project-less sessions, which shell into the
// user's home directory).
type Session struct {
	ID            uuid.UUID
	Name          string
	ProjectID     uuid.UUID
	Order         int
	WorktreePath  string
	BaseBranch    string
	FeatureBranch string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewSession constructs a Session with no worktree assigned. Callers that
// need a project-backed session must follow up with AssignWorktree.
func NewSession(name string, projectID uuid.UUID) (Session, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return Session{}, ErrSessionEmptyName
	}
	if len(name) > 100 {
		return Session{}, ErrSessionNameTooLong
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

// HasWorktree reports whether the session has a worktree assigned.
func (s Session) HasWorktree() bool {
	return s.WorktreePath != ""
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
// invariant.
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
}

// Session sentinel errors.
var (
	ErrSessionEmptyName               = errors.New("session name cannot be empty")
	ErrSessionNameTooLong             = errors.New("session name exceeds 100 characters")
	ErrSessionNotFound                = errors.New("session not found")
	ErrSessionAlreadyExists           = errors.New("session already exists")
	ErrSessionWorktreeFieldsMismatch  = errors.New("session worktree fields must all be set")
	ErrSessionWorktreePathNotAbsolute = errors.New("session worktree path must be absolute")
)
