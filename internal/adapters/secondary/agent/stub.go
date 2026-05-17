// Package agent provides agent launcher implementations of the domain.AgentLauncher port.
package agent

import (
	"context"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface check.
var _ domain.AgentLauncher = (*Stub)(nil)

// Stub is a stub implementation of domain.AgentLauncher.
// It records call counts for test assertions and returns a fixed canned PID.
// Replace with a real implementation when integrating real agent launching.
type Stub struct {
	LaunchCalls int
}

// Launch records the call and returns a canned PID of 12345 without spawning any process.
func (s *Stub) Launch(_ context.Context, _, _ string) (int, error) {
	s.LaunchCalls++
	return 12345, nil
}
