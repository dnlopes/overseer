package tmux_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/secondary/tmux"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface satisfaction check.
var _ domain.TmuxAdapter = (*tmux.Stub)(nil)

func TestCreateSession_IncrementsCalls(t *testing.T) {
	s := &tmux.Stub{}

	id, err := s.CreateSession(context.Background(), "my-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.CreateSessionCalls != 1 {
		t.Errorf("CreateSessionCalls = %d, want 1", s.CreateSessionCalls)
	}
	if !strings.HasPrefix(id, "tmux-stub-my-session-") {
		t.Errorf("id %q does not have expected prefix", id)
	}
}

func TestKillSession_IncrementsCalls(t *testing.T) {
	s := &tmux.Stub{}

	if err := s.KillSession(context.Background(), "some-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.KillSessionCalls != 1 {
		t.Errorf("KillSessionCalls = %d, want 1", s.KillSessionCalls)
	}
}
