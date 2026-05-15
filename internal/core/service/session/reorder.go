package session

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	"github.com/dnlopes/overseer/internal/shared/errs"
	"github.com/google/uuid"
)

type ReorderRequest struct {
	ID        uuid.UUID
	Direction int // +1 = down (higher order), -1 = up (lower order)
}

type ReorderResponse struct {
	Sessions []domainsession.Session
}

type ReorderUseCase struct {
	repo   domainsession.Repository
	logger *slog.Logger
}

func NewReorderUseCase(repo domainsession.Repository, logger *slog.Logger) *ReorderUseCase {
	return &ReorderUseCase{repo: repo, logger: logger}
}

func (uc *ReorderUseCase) Execute(ctx context.Context, req ReorderRequest) (ReorderResponse, error) {
	target, err := uc.repo.Get(ctx, req.ID)
	if err != nil {
		return ReorderResponse{}, err
	}

	all, err := uc.repo.List(ctx)
	if err != nil {
		return ReorderResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	projectSessions := make([]domainsession.Session, 0, len(all))
	for _, s := range all {
		if s.ProjectName == target.ProjectName {
			projectSessions = append(projectSessions, s)
		}
	}
	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	if len(projectSessions) <= 1 {
		return ReorderResponse{}, errs.ErrNoOp
	}

	idx := -1
	for i, s := range projectSessions {
		if s.ID == target.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ReorderResponse{}, fmt.Errorf("session %s not found in project list", target.ID)
	}

	if (idx == 0 && req.Direction == -1) || (idx == len(projectSessions)-1 && req.Direction == 1) {
		return ReorderResponse{}, errs.ErrNoOp
	}

	neighbor := idx + req.Direction
	projectSessions[idx].Order, projectSessions[neighbor].Order =
		projectSessions[neighbor].Order, projectSessions[idx].Order

	if err := uc.repo.Save(ctx, projectSessions[idx]); err != nil {
		return ReorderResponse{}, fmt.Errorf("save session: %w", err)
	}
	if err := uc.repo.Save(ctx, projectSessions[neighbor]); err != nil {
		return ReorderResponse{}, fmt.Errorf("save neighbor session: %w", err)
	}

	sort.Slice(projectSessions, func(i, j int) bool {
		return projectSessions[i].Order < projectSessions[j].Order
	})

	uc.logger.InfoContext(ctx, "session reordered",
		slog.String("id", target.ID.String()),
		slog.Int("direction", req.Direction),
	)

	return ReorderResponse{Sessions: projectSessions}, nil
}
