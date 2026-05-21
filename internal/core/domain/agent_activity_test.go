package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnlopes/overseer/internal/core/domain"
)

func TestActivityKind_IsValid(t *testing.T) {
	t.Parallel()

	known := []domain.ActivityKind{
		domain.ActivityUnknown, domain.ActivityIdle, domain.ActivityWaiting,
		domain.ActivityThinking, domain.ActivityReading, domain.ActivityWriting,
		domain.ActivityRunning, domain.ActivitySearching, domain.ActivityBrowsing,
		domain.ActivitySpawning, domain.ActivityCompacting,
	}
	for _, k := range known {
		assert.Truef(t, k.IsValid(), "expected %q to be valid", k)
	}

	assert.False(t, domain.ActivityKind("bogus").IsValid())
	assert.False(t, domain.ActivityKind("").IsValid())
}

func TestActivityKind_IsActive(t *testing.T) {
	t.Parallel()

	inactive := []domain.ActivityKind{
		domain.ActivityIdle, domain.ActivityWaiting, domain.ActivityUnknown,
	}
	for _, k := range inactive {
		assert.Falsef(t, k.IsActive(), "expected %q to NOT be active", k)
	}

	active := []domain.ActivityKind{
		domain.ActivityThinking, domain.ActivityReading, domain.ActivityWriting,
		domain.ActivityRunning, domain.ActivitySearching, domain.ActivityBrowsing,
		domain.ActivitySpawning, domain.ActivityCompacting,
	}
	for _, k := range active {
		assert.Truef(t, k.IsActive(), "expected %q to be active", k)
	}
}

func TestNewAgentActivity_ValidInputs_ReturnsActivity(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()

	got, err := domain.NewAgentActivity(sessionID, domain.ActivityReading, "Read")

	require.NoError(t, err)
	assert.Equal(t, sessionID, got.SessionID)
	assert.Equal(t, domain.ActivityReading, got.Kind)
	assert.Equal(t, "Read", got.Tool)
	assert.False(t, got.ObservedAt.IsZero(), "ObservedAt must be set by constructor")
}

func TestNewAgentActivity_EmptyTool_Allowed(t *testing.T) {
	t.Parallel()

	got, err := domain.NewAgentActivity(uuid.New(), domain.ActivityIdle, "")

	require.NoError(t, err)
	assert.Empty(t, got.Tool)
}

func TestNewAgentActivity_NilSessionID_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	_, err := domain.NewAgentActivity(uuid.Nil, domain.ActivityIdle, "")

	assert.ErrorIs(t, err, domain.ErrAgentActivityInvalidSessionID)
}

func TestNewAgentActivity_UnknownKind_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	_, err := domain.NewAgentActivity(uuid.New(), domain.ActivityKind("bogus"), "")

	assert.ErrorIs(t, err, domain.ErrAgentActivityInvalidKind)
}

func TestAgentActivitySentinels_AreDistinct(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		domain.ErrAgentActivityInvalidSessionID,
		domain.ErrAgentActivityInvalidKind,
		domain.ErrAgentNotRunning,
		domain.ErrAgentStoreNotResolved,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			assert.Falsef(t, errors.Is(a, b),
				"sentinel %d (%v) must not match sentinel %d (%v)", i, a, j, b)
		}
	}
}
