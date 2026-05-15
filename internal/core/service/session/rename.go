package session

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
)

type RenameRequest struct {
	ID      uuid.UUID
	NewName string
}

type RenameResponse struct {
	Session domainsession.Session
}

type RenameUseCase struct {
	repo   domainsession.Repository
	logger *slog.Logger
}

func NewRenameUseCase(repo domainsession.Repository, logger *slog.Logger) *RenameUseCase {
	return &RenameUseCase{repo: repo, logger: logger}
}

func (uc *RenameUseCase) Execute(ctx context.Context, req RenameRequest) (RenameResponse, error) {
	s, err := uc.repo.Get(ctx, req.ID)
	if err != nil {
		return RenameResponse{}, err
	}

	existing, err := uc.repo.List(ctx)
	if err != nil {
		return RenameResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	for _, candidate := range existing {
		if candidate.ID == s.ID {
			continue
		}
		if candidate.ProjectName == s.ProjectName && candidate.Name == req.NewName {
			return RenameResponse{}, domainsession.ErrAlreadyExists
		}
	}

	if err := s.Rename(req.NewName); err != nil {
		return RenameResponse{}, err
	}

	if err := uc.repo.Save(ctx, s); err != nil {
		return RenameResponse{}, fmt.Errorf("save session: %w", err)
	}

	return RenameResponse{Session: s}, nil
}
