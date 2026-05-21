package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newRef(t *testing.T) domain.AgentSessionRef {
	t.Helper()
	id := uuid.New()
	return domain.AgentSessionRef{
		SessionID:          id,
		AgentTmuxSessionID: id.String() + "-agent",
		WorktreePath:       "/tmp/worktree",
	}
}

func TestAgentActivityService_Observe_ProviderReturnsActivity_PassesThrough(t *testing.T) {
	t.Parallel()
	ref := newRef(t)
	want, err := domain.NewAgentActivity(ref.SessionID, domain.ActivityReading, "Read")
	require.NoError(t, err)
	provider := mocks.NewMockAgentActivityProvider(t)
	provider.EXPECT().Observe(mock.Anything, ref).Return(want, nil).Once()
	svc := service.NewAgentActivityService(provider, discardLogger())

	resp, err := svc.Observe(context.Background(), service.ObserveAgentActivityRequest{Ref: ref})

	require.NoError(t, err)
	assert.Equal(t, want, resp.Activity)
}

func TestAgentActivityService_Observe_AgentNotRunning_ReturnsSentinelUnwrapped(t *testing.T) {
	t.Parallel()
	ref := newRef(t)
	provider := mocks.NewMockAgentActivityProvider(t)
	provider.EXPECT().Observe(mock.Anything, ref).Return(domain.AgentActivity{}, domain.ErrAgentNotRunning).Once()
	svc := service.NewAgentActivityService(provider, discardLogger())

	_, err := svc.Observe(context.Background(), service.ObserveAgentActivityRequest{Ref: ref})

	assert.ErrorIs(t, err, domain.ErrAgentNotRunning)
}

func TestAgentActivityService_Observe_StoreNotResolved_ReturnsSentinelUnwrapped(t *testing.T) {
	t.Parallel()
	ref := newRef(t)
	provider := mocks.NewMockAgentActivityProvider(t)
	provider.EXPECT().Observe(mock.Anything, ref).Return(domain.AgentActivity{}, domain.ErrAgentStoreNotResolved).Once()
	svc := service.NewAgentActivityService(provider, discardLogger())

	_, err := svc.Observe(context.Background(), service.ObserveAgentActivityRequest{Ref: ref})

	assert.ErrorIs(t, err, domain.ErrAgentStoreNotResolved)
}

func TestAgentActivityService_Observe_OtherError_WrappedWithContext(t *testing.T) {
	t.Parallel()
	ref := newRef(t)
	infraErr := errors.New("lsof returned non-zero")
	provider := mocks.NewMockAgentActivityProvider(t)
	provider.EXPECT().Observe(mock.Anything, ref).Return(domain.AgentActivity{}, infraErr).Once()
	svc := service.NewAgentActivityService(provider, discardLogger())

	_, err := svc.Observe(context.Background(), service.ObserveAgentActivityRequest{Ref: ref})

	assert.ErrorIs(t, err, infraErr)
	assert.NotErrorIs(t, err, domain.ErrAgentNotRunning)
	assert.NotErrorIs(t, err, domain.ErrAgentStoreNotResolved)
	assert.Contains(t, err.Error(), "observe agent activity for session")
}
