package agent_test

import (
	"context"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/secondary/agent"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface satisfaction check.
var _ domain.AgentLauncher = (*agent.Stub)(nil)

func TestLaunch_IncrementsCalls(t *testing.T) {
	s := &agent.Stub{}

	pid, err := s.Launch(context.Background(), "harness", "/workdir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.LaunchCalls != 1 {
		t.Errorf("LaunchCalls = %d, want 1", s.LaunchCalls)
	}
	if pid != 12345 {
		t.Errorf("pid = %d, want 12345", pid)
	}
}
