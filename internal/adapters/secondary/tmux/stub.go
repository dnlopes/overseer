// Package tmux provides tmux adapter implementations of the domain.TmuxAdapter port.
package tmux

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface check.
var _ domain.TmuxAdapter = (*Stub)(nil)

// Stub is a stub implementation of domain.TmuxAdapter.
// It satisfies the port interface with canned responses and records call counts for testing.
// Replace with a real implementation when integrating real tmux.
type Stub struct {
	CreateSessionCalls int
	KillSessionCalls   int
}

// CreateSession returns a deterministic canned tmux session ID of the form "tmux-stub-<name>-<uuid8>".
func (s *Stub) CreateSession(_ context.Context, name string) (string, error) {
	s.CreateSessionCalls++
	id := uuid.New().String()[:8]
	return fmt.Sprintf("tmux-stub-%s-%s", name, id), nil
}

// KillSession records the call and returns nil without touching any real tmux session.
func (s *Stub) KillSession(_ context.Context, _ string) error {
	s.KillSessionCalls++
	return nil
}
