package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// AgentStatusService polls every persisted session and resolves its
// AgentStatus by combining a tmux liveness probe (Dead detection) with a
// per-agent-type detector (Running / Waiting / Idle / Unknown). Per-agent
// detectors are kept ignorant of process liveness on purpose — see plan
// §4.
type AgentStatusService struct {
	sessions          domain.SessionRepository
	tmux              domain.TmuxAdapter
	registry          domain.AgentStatusDetectorRegistry
	logger            *slog.Logger
	fanOutConcurrency int
	detectorTimeout   time.Duration
}

const (
	defaultFanOutConcurrency = 8
	defaultDetectorTimeout   = 2 * time.Second
)

// NewAgentStatusService wires the use-case. Defaults: 8 concurrent
// detectors (bounded to prevent tmux subprocess flooding) and a 2-second
// per-detector timeout (so one hung agent does not stall the whole tick).
func NewAgentStatusService(
	sessions domain.SessionRepository,
	tmux domain.TmuxAdapter,
	registry domain.AgentStatusDetectorRegistry,
	logger *slog.Logger,
) *AgentStatusService {
	return &AgentStatusService{
		sessions:          sessions,
		tmux:              tmux,
		registry:          registry,
		logger:            logger,
		fanOutConcurrency: defaultFanOutConcurrency,
		detectorTimeout:   defaultDetectorTimeout,
	}
}

type PollAllAgentStatusesRequest struct{}

type PollAllAgentStatusesResponse struct {
	Statuses map[uuid.UUID]domain.AgentStatus
}

// PollAll resolves the AgentStatus for every persisted session. The
// returned map is keyed by Session.ID; sessions whose detector errors out
// are still present in the map with a status of AgentStatusUnknown so
// callers can tell "polled, failed" from "not polled at all". Per-session
// work fans out across goroutines bounded by fanOutConcurrency.
func (s *AgentStatusService) PollAll(ctx context.Context, _ PollAllAgentStatusesRequest) (PollAllAgentStatusesResponse, error) {
	sessions, err := s.sessions.List(ctx)
	if err != nil {
		return PollAllAgentStatusesResponse{}, fmt.Errorf("list sessions: %w", err)
	}

	statuses := make(map[uuid.UUID]domain.AgentStatus, len(sessions))
	var mu sync.Mutex
	sem := make(chan struct{}, s.fanOutConcurrency)
	var wg sync.WaitGroup

	for _, sess := range sessions {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			status := s.pollOne(ctx, sess)
			mu.Lock()
			statuses[sess.ID] = status
			mu.Unlock()
		}()
	}
	wg.Wait()

	return PollAllAgentStatusesResponse{Statuses: statuses}, nil
}

func (s *AgentStatusService) pollOne(ctx context.Context, sess domain.Session) domain.AgentStatus {
	now := time.Now()
	agentTmuxID := sess.ID.String() + "-agent"

	if _, err := s.tmux.GetSession(ctx, agentTmuxID); err != nil {
		if errors.Is(err, domain.ErrTmuxSessionNotFound) {
			return domain.AgentStatus{
				Kind:       domain.AgentStatusDead,
				DetectedAt: now,
				Source:     "service/tmux-liveness",
				Reason:     "agent tmux session not found",
			}
		}
		s.logger.WarnContext(ctx, "agent status tmux liveness check failed",
			slog.String("session_id", sess.ID.String()),
			slog.String("agent_tmux_id", agentTmuxID),
			slog.String("error", err.Error()),
		)
		return domain.AgentStatus{
			Kind:       domain.AgentStatusUnknown,
			DetectedAt: now,
			Source:     "service/tmux-liveness",
			Reason:     fmt.Sprintf("tmux error: %v", err),
		}
	}

	detector, ok := s.registry.DetectorFor(sess.AgentType)
	if !ok {
		return domain.AgentStatus{
			Kind:       domain.AgentStatusUnknown,
			DetectedAt: now,
			Source:     "service/registry",
			Reason:     fmt.Sprintf("no detector registered for agent type %q", sess.AgentType),
		}
	}

	detectCtx, cancel := context.WithTimeout(ctx, s.detectorTimeout)
	defer cancel()

	status, err := detector.Detect(detectCtx, sess)
	if err != nil {
		s.logger.WarnContext(ctx, "agent status detector failed",
			slog.String("session_id", sess.ID.String()),
			slog.String("agent_type", string(sess.AgentType)),
			slog.String("error", err.Error()),
		)
		return domain.AgentStatus{
			Kind:       domain.AgentStatusUnknown,
			DetectedAt: now,
			Source:     "service/detector-error",
			Reason:     err.Error(),
		}
	}
	return status
}
