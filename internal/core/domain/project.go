package domain

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Project is the aggregate representing a registered Git repository on disk.
type Project struct {
	ID        uuid.UUID
	Name      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewProject(path, name string) (Project, error) {
	path = strings.TrimSpace(path)
	name = strings.TrimSpace(name)

	if path == "" {
		return Project{}, ErrProjectEmptyPath
	}
	if !filepath.IsAbs(path) {
		return Project{}, ErrProjectPathNotAbsolute
	}
	if name == "" {
		name = filepath.Base(path)
	}
	if len(name) > 100 {
		return Project{}, ErrProjectNameTooLong
	}

	now := time.Now()
	return Project{
		ID:        uuid.New(),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Project ports.

type ProjectRepository interface {
	Save(ctx context.Context, p Project) error
	Get(ctx context.Context, id uuid.UUID) (Project, error)
	GetByPath(ctx context.Context, path string) (Project, error)
	List(ctx context.Context) ([]Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Project sentinel errors.
var (
	ErrProjectEmptyPath       = errors.New("project path cannot be empty")
	ErrProjectPathNotAbsolute = errors.New("project path must be absolute")
	ErrProjectNameTooLong     = errors.New("project name exceeds 100 characters")
	ErrProjectNotFound        = errors.New("project not found")
	ErrProjectAlreadyExists   = errors.New("project already exists")
	ErrProjectNotGitRepo      = errors.New("project path is not a git repository")
	ErrProjectNoDefaultBranch = errors.New("project has no detectable default branch")
)
