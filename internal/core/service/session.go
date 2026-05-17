package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/shared/errs"
)

// SessionService orchestrates Session use cases.
type SessionService struct {
	repo   domain.SessionRepository
	tmux   domain.TmuxAdapter
	git    domain.GitAdapter
	logger *slog.Logger
}

func NewSessionService(repo domain.SessionRepository, tmux domain.TmuxAdapter, git domain.GitAdapter, logger *slog.Logger) *SessionService {
	return &SessionService{repo: repo, tmux: tmux, git: git, logger: logger}
}

// --- Create ---

type CreateSessionRequest struct {
	Name        string
	ProjectName string
}

type CreateSessionResponse struct {
	Session domain.Session
}

func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error) {
	sess, err := domain.NewSession(req.Name, req.ProjectName)
	if err != nil {
		return CreateSessionResponse{}, err
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return CreateSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	nextOrder := 1
	for _, candidate := range existing {
		if candidate.ProjectName != sess.ProjectName {
			continue
		}
		if candidate.Name == sess.Name {
			return CreateSessionResponse{}, domain.ErrSessionAlreadyExists
		}
		if candidate.Order >= nextOrder {
			nextOrder = candidate.Order + 1
		}
	}
	sess.Order = nextOrder

	if _, err := s.tmux.CreateSession(ctx, req.Name); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("create tmux session: %w", err)
	}
	if err := s.git.CreateWorktree(ctx, "main", req.Name); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("create git worktree: %w", err)
	}
	if err := s.repo.Save(ctx, sess); err != nil {
		return CreateSessionResponse{}, fmt.Errorf("save session: %w", err)
	}

	return CreateSessionResponse{Session: sess}, nil
}

// --- Rename ---

type RenameSessionRequest struct {
	ID      uuid.UUID
	NewName string
}

type RenameSessionResponse struct {
	Session domain.Session
}

func (s *SessionService) Rename(ctx context.Context, req RenameSessionRequest) (RenameSessionResponse, error) {
	sess, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return RenameSessionResponse{}, err
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return RenameSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	for _, candidate := range existing {
		if candidate.ID == sess.ID {
			continue
		}
		if candidate.ProjectName == sess.ProjectName && candidate.Name == req.NewName {
			return RenameSessionResponse{}, domain.ErrSessionAlreadyExists
		}
	}

	if err := sess.Rename(req.NewName); err != nil {
		return RenameSessionResponse{}, err
	}

	if err := s.repo.Save(ctx, sess); err != nil {
		return RenameSessionResponse{}, fmt.Errorf("save session: %w", err)
	}

	return RenameSessionResponse{Session: sess}, nil
}

// --- List ---

type ListSessionsRequest struct{}

type SessionGroup struct {
	ProjectName string
	Sessions    []domain.Session
}

type ListSessionsResponse struct {
	Groups []SessionGroup
}

func (s *SessionService) List(ctx context.Context, _ ListSessionsRequest) (ListSessionsResponse, error) {
	sessions, err := s.repo.List(ctx)
	if err != nil {
		return ListSessionsResponse{}, err
	}

	grouped := make(map[string]*SessionGroup)
	for _, sess := range sessions {
		if _, ok := grouped[sess.ProjectName]; !ok {
			grouped[sess.ProjectName] = &SessionGroup{ProjectName: sess.ProjectName}
		}
		grouped[sess.ProjectName].Sessions = append(grouped[sess.ProjectName].Sessions, sess)
	}

	groups := make([]SessionGroup, 0, len(grouped))
	for _, g := range grouped {
		sort.Slice(g.Sessions, func(i, j int) bool { return g.Sessions[i].Order < g.Sessions[j].Order })
		groups = append(groups, *g)
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].ProjectName < groups[j].ProjectName })

	return ListSessionsResponse{Groups: groups}, nil
}

// --- Reorder ---

type ReorderSessionRequest struct {
	ID        uuid.UUID
	Direction int // +1 = down (higher order), -1 = up (lower order)
}

type ReorderSessionResponse struct {
	Sessions []domain.Session
}

func (s *SessionService) Reorder(ctx context.Context, req ReorderSessionRequest) (ReorderSessionResponse, error) {
	target, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return ReorderSessionResponse{}, err
	}

	all, err := s.repo.List(ctx)
	if err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	projectSessions := make([]domain.Session, 0, len(all))
	for _, sess := range all {
		if sess.ProjectName == target.ProjectName {
			projectSessions = append(projectSessions, sess)
		}
	}
	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	if len(projectSessions) <= 1 {
		return ReorderSessionResponse{}, errs.ErrNoOp
	}

	idx := -1
	for i, sess := range projectSessions {
		if sess.ID == target.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ReorderSessionResponse{}, fmt.Errorf("session %s not found in project list", target.ID)
	}

	if (idx == 0 && req.Direction == -1) || (idx == len(projectSessions)-1 && req.Direction == 1) {
		return ReorderSessionResponse{}, errs.ErrNoOp
	}

	neighbor := idx + req.Direction
	projectSessions[idx].Order, projectSessions[neighbor].Order =
		projectSessions[neighbor].Order, projectSessions[idx].Order

	if err := s.repo.Save(ctx, projectSessions[idx]); err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("save session: %w", err)
	}
	if err := s.repo.Save(ctx, projectSessions[neighbor]); err != nil {
		return ReorderSessionResponse{}, fmt.Errorf("save neighbor session: %w", err)
	}

	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	s.logger.InfoContext(ctx, "session reordered",
		slog.String("id", target.ID.String()),
		slog.Int("direction", req.Direction),
	)

	return ReorderSessionResponse{Sessions: projectSessions}, nil
}
