package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Session is the aggregate representing a single AI agent session.
type Session struct {
	ID          uuid.UUID
	Name        string
	ProjectName string
	Order       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewSession(name, project string) (Session, error) {
	name = strings.TrimSpace(name)
	project = strings.TrimSpace(project)

	if name == "" {
		return Session{}, ErrSessionEmptyName
	}
	if len(name) > 100 {
		return Session{}, ErrSessionNameTooLong
	}
	if project == "" {
		return Session{}, ErrSessionEmptyProject
	}
	if len(project) > 100 {
		return Session{}, ErrSessionProjectTooLong
	}

	now := time.Now()
	return Session{
		ID:          uuid.New(),
		Name:        name,
		ProjectName: project,
		Order:       0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
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

func (s Session) String() string {
	return "[" + s.ProjectName + "] " + s.Name
}

// Session ports.

type SessionRepository interface {
	Save(ctx context.Context, s Session) error
	Get(ctx context.Context, id uuid.UUID) (Session, error)
	List(ctx context.Context) ([]Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type TmuxAdapter interface {
	CreateSession(ctx context.Context, name string) (tmuxID string, err error)
	KillSession(ctx context.Context, tmuxID string) error
}

type GitAdapter interface {
	CreateWorktree(ctx context.Context, baseBranch, path string) error
	RemoveWorktree(ctx context.Context, path string) error
}

type AgentLauncher interface {
	Launch(ctx context.Context, harness, workdir string) (pid int, err error)
}

// Session sentinel errors.
var (
	ErrSessionEmptyName      = errors.New("session name cannot be empty")
	ErrSessionNameTooLong    = errors.New("session name exceeds 100 characters")
	ErrSessionEmptyProject   = errors.New("project name cannot be empty")
	ErrSessionProjectTooLong = errors.New("project name exceeds 100 characters")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionAlreadyExists  = errors.New("session already exists")
)
