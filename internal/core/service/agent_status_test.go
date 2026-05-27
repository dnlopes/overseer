package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestAgentStatusService_PollAll_RoutesByAgentType(t *testing.T) {
	ctx := context.Background()
	claudeSess := newSessionFixture(domain.AgentTypeClaudeCode)
	openCodeSess := newSessionFixture(domain.AgentTypeOpenCode)

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{claudeSess, openCodeSess}, nil).Once()
	expectAgentTmuxAlive(tmux, claudeSess.ID)
	expectAgentTmuxAlive(tmux, openCodeSess.ID)

	claudeDetector := stubDetectorReturning(t, domain.AgentTypeClaudeCode, runningStatus("claude/test"))
	openCodeDetector := stubDetectorReturning(t, domain.AgentTypeOpenCode, idleStatus("opencode/test"))
	registry.EXPECT().DetectorFor(domain.AgentTypeClaudeCode).Return(claudeDetector, true).Once()
	registry.EXPECT().DetectorFor(domain.AgentTypeOpenCode).Return(openCodeDetector, true).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	if len(resp.Statuses) != 2 {
		t.Fatalf("PollAll returned %d statuses, want 2", len(resp.Statuses))
	}
	if got := resp.Statuses[claudeSess.ID].Kind; got != domain.AgentStatusRunning {
		t.Fatalf("Statuses[claude].Kind = %q, want %q", got, domain.AgentStatusRunning)
	}
	if got := resp.Statuses[openCodeSess.ID].Kind; got != domain.AgentStatusIdle {
		t.Fatalf("Statuses[opencode].Kind = %q, want %q", got, domain.AgentStatusIdle)
	}
}

func TestAgentStatusService_PollAll_ReturnsDeadWhenAgentTmuxMissing(t *testing.T) {
	ctx := context.Background()
	sess := newSessionFixture(domain.AgentTypeClaudeCode)
	agentTmuxID := sess.ID.String() + "-agent"

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{sess}, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).Return(domain.TmuxSession{}, domain.ErrTmuxSessionNotFound).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	status, ok := resp.Statuses[sess.ID]
	if !ok {
		t.Fatalf("PollAll missing status for session")
	}
	if status.Kind != domain.AgentStatusDead {
		t.Fatalf("Kind = %q, want %q", status.Kind, domain.AgentStatusDead)
	}
	if status.Source == "" {
		t.Fatalf("Dead status Source = empty, want non-empty")
	}
	if status.DetectedAt.IsZero() {
		t.Fatalf("Dead status DetectedAt zero, want non-zero")
	}
}

func TestAgentStatusService_PollAll_TmuxError_DegradesToUnknown(t *testing.T) {
	ctx := context.Background()
	sess := newSessionFixture(domain.AgentTypeClaudeCode)
	agentTmuxID := sess.ID.String() + "-agent"

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{sess}, nil).Once()
	tmux.EXPECT().GetSession(mock.Anything, agentTmuxID).Return(domain.TmuxSession{}, errors.New("tmux fail")).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	if resp.Statuses[sess.ID].Kind != domain.AgentStatusUnknown {
		t.Fatalf("Kind = %q, want %q", resp.Statuses[sess.ID].Kind, domain.AgentStatusUnknown)
	}
}

func TestAgentStatusService_PollAll_NoDetectorForAgentType_ReturnsUnknown(t *testing.T) {
	ctx := context.Background()
	sess := newSessionFixture(domain.AgentTypeUnknown)

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{sess}, nil).Once()
	expectAgentTmuxAlive(tmux, sess.ID)
	registry.EXPECT().DetectorFor(domain.AgentTypeUnknown).Return(nil, false).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	status := resp.Statuses[sess.ID]
	if status.Kind != domain.AgentStatusUnknown {
		t.Fatalf("Kind = %q, want %q", status.Kind, domain.AgentStatusUnknown)
	}
	if status.Source == "" {
		t.Fatalf("Unknown status Source = empty, want non-empty")
	}
}

func TestAgentStatusService_PollAll_DetectorError_DegradesToUnknown(t *testing.T) {
	ctx := context.Background()
	sess := newSessionFixture(domain.AgentTypeClaudeCode)

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{sess}, nil).Once()
	expectAgentTmuxAlive(tmux, sess.ID)

	detector := mocks.NewMockAgentStatusDetector(t)
	detector.EXPECT().Detect(mock.Anything, sess).Return(domain.AgentStatus{}, errors.New("boom")).Once()
	registry.EXPECT().DetectorFor(domain.AgentTypeClaudeCode).Return(detector, true).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	if resp.Statuses[sess.ID].Kind != domain.AgentStatusUnknown {
		t.Fatalf("Kind = %q, want %q", resp.Statuses[sess.ID].Kind, domain.AgentStatusUnknown)
	}
}

func TestAgentStatusService_PollAll_RespectsDetectorTimeout(t *testing.T) {
	ctx := context.Background()
	sess := newSessionFixture(domain.AgentTypeClaudeCode)

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{sess}, nil).Once()
	expectAgentTmuxAlive(tmux, sess.ID)

	gotDeadline := atomic.Bool{}
	detector := mocks.NewMockAgentStatusDetector(t)
	detector.EXPECT().Detect(mock.Anything, sess).
		RunAndReturn(func(detectCtx context.Context, _ domain.Session) (domain.AgentStatus, error) {
			if _, ok := detectCtx.Deadline(); ok {
				gotDeadline.Store(true)
			}
			return domain.AgentStatus{Kind: domain.AgentStatusRunning, DetectedAt: time.Now()}, nil
		}).Once()
	registry.EXPECT().DetectorFor(domain.AgentTypeClaudeCode).Return(detector, true).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	if _, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{}); err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	if !gotDeadline.Load() {
		t.Fatal("detector was called without a deadline on its context")
	}
}

func TestAgentStatusService_PollAll_RepoError_PropagatesAsWrappedError(t *testing.T) {
	ctx := context.Background()

	repo, tmux, registry := newAgentStatusMocks(t)
	wantErr := errors.New("repo unavailable")
	repo.EXPECT().List(mock.Anything).Return(nil, wantErr).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	_, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err == nil {
		t.Fatal("PollAll: err = nil, want wrapped repo error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("PollAll: err = %v, want chain to include %v", err, wantErr)
	}
}

func TestAgentStatusService_PollAll_EmptySessionList_ReturnsEmptyMap(t *testing.T) {
	ctx := context.Background()

	repo, tmux, registry := newAgentStatusMocks(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{}, nil).Once()

	svc := newAgentStatusServiceWithDeps(repo, tmux, registry)
	resp, err := svc.PollAll(ctx, PollAllAgentStatusesRequest{})
	if err != nil {
		t.Fatalf("PollAll err = %v", err)
	}
	if len(resp.Statuses) != 0 {
		t.Fatalf("PollAll returned %d statuses, want 0", len(resp.Statuses))
	}
}

func newAgentStatusMocks(t *testing.T) (*mocks.MockSessionRepository, *mocks.MockTmuxAdapter, *mocks.MockAgentStatusDetectorRegistry) {
	t.Helper()
	return mocks.NewMockSessionRepository(t),
		mocks.NewMockTmuxAdapter(t),
		mocks.NewMockAgentStatusDetectorRegistry(t)
}

func newAgentStatusServiceWithDeps(
	repo domain.SessionRepository,
	tmux domain.TmuxAdapter,
	registry domain.AgentStatusDetectorRegistry,
) *AgentStatusService {
	return NewAgentStatusService(repo, tmux, registry, testLogger())
}

func newSessionFixture(at domain.AgentType) domain.Session {
	return domain.Session{ID: uuid.New(), Name: "fixture", AgentType: at}
}

func expectAgentTmuxAlive(tmux *mocks.MockTmuxAdapter, sessID uuid.UUID) {
	tmux.EXPECT().GetSession(mock.Anything, sessID.String()+"-agent").
		Return(domain.TmuxSession{ID: sessID.String() + "-agent"}, nil).Once()
}

func stubDetectorReturning(t *testing.T, at domain.AgentType, status domain.AgentStatus) *mocks.MockAgentStatusDetector {
	t.Helper()
	d := mocks.NewMockAgentStatusDetector(t)
	d.EXPECT().Detect(mock.Anything, mock.Anything).Return(status, nil).Once()
	_ = at
	return d
}

func runningStatus(source string) domain.AgentStatus {
	return domain.AgentStatus{Kind: domain.AgentStatusRunning, DetectedAt: time.Now(), Source: source}
}

func idleStatus(source string) domain.AgentStatus {
	return domain.AgentStatus{Kind: domain.AgentStatusIdle, DetectedAt: time.Now(), Source: source}
}
