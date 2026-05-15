package session

import (
	"context"
	"fmt"
	"log/slog"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
)

type CreateRequest struct {
	Name        string
	ProjectName string
}

type CreateResponse struct {
	Session domainsession.Session
}

type CreateUseCase struct {
	repo   domainsession.Repository
	tmux   domainsession.TmuxAdapter
	git    domainsession.GitAdapter
	logger *slog.Logger
}

func NewCreateUseCase(repo domainsession.Repository, tmux domainsession.TmuxAdapter, git domainsession.GitAdapter, logger *slog.Logger) *CreateUseCase {
	return &CreateUseCase{repo: repo, tmux: tmux, git: git, logger: logger}
}

func (uc *CreateUseCase) Execute(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	s, err := domainsession.New(req.Name, req.ProjectName)
	if err != nil {
		return CreateResponse{}, err
	}

	existing, err := uc.repo.List(ctx)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	nextOrder := 1
	for _, candidate := range existing {
		if candidate.ProjectName != s.ProjectName {
			continue
		}
		if candidate.Name == s.Name {
			return CreateResponse{}, domainsession.ErrAlreadyExists
		}
		if candidate.Order >= nextOrder {
			nextOrder = candidate.Order + 1
		}
	}
	s.Order = nextOrder

	if _, err := uc.tmux.CreateSession(ctx, req.Name); err != nil {
		return CreateResponse{}, fmt.Errorf("create tmux session: %w", err)
	}
	if err := uc.git.CreateWorktree(ctx, "main", req.Name); err != nil {
		return CreateResponse{}, fmt.Errorf("create git worktree: %w", err)
	}
	if err := uc.repo.Save(ctx, s); err != nil {
		return CreateResponse{}, fmt.Errorf("save session: %w", err)
	}

	return CreateResponse{Session: s}, nil
}
