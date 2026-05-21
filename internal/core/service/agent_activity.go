package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// AgentActivityService observes the runtime activity of agent sessions.
//
// It is a thin orchestration layer over the AgentActivityProvider port.
// Polling, diffing and message emission are caller concerns (the TUI
// scheduler) — this service exposes a single use-case verb, Observe,
// that returns the current snapshot for one agent session.
type AgentActivityService struct {
	provider domain.AgentActivityProvider
	logger   *slog.Logger
}

// NewAgentActivityService constructs an AgentActivityService bound to the
// given provider.
func NewAgentActivityService(provider domain.AgentActivityProvider, logger *slog.Logger) *AgentActivityService {
	return &AgentActivityService{provider: provider, logger: logger}
}

// ObserveAgentActivityRequest is the input to AgentActivityService.Observe.
type ObserveAgentActivityRequest struct {
	Ref domain.AgentSessionRef
}

// ObserveAgentActivityResponse is the output of AgentActivityService.Observe.
type ObserveAgentActivityResponse struct {
	Activity domain.AgentActivity
}

// Observe returns the current activity for the referenced agent session.
//
// The two soft-failure sentinels — ErrAgentNotRunning and
// ErrAgentStoreNotResolved — are returned unwrapped so callers can match
// them with errors.Is and degrade to ActivityUnknown / retry, respectively.
// Any other error is wrapped with use-case context.
func (s *AgentActivityService) Observe(ctx context.Context, req ObserveAgentActivityRequest) (ObserveAgentActivityResponse, error) {
	activity, err := s.provider.Observe(ctx, req.Ref)
	if err != nil {
		if errors.Is(err, domain.ErrAgentNotRunning) || errors.Is(err, domain.ErrAgentStoreNotResolved) {
			return ObserveAgentActivityResponse{}, err
		}
		s.logger.Warn("agent activity observe failed",
			"session_id", req.Ref.SessionID,
			"tmux_session", req.Ref.AgentTmuxSessionID,
			"error", err)
		return ObserveAgentActivityResponse{}, fmt.Errorf("observe agent activity for session %s: %w", req.Ref.SessionID, err)
	}
	return ObserveAgentActivityResponse{Activity: activity}, nil
}
