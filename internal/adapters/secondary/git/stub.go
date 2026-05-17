// Package git provides git adapter implementations of the domain.GitAdapter port.
package git

import (
	"context"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface check.
var _ domain.GitAdapter = (*Stub)(nil)

// Stub is a stub implementation of domain.GitAdapter.
// It records call counts for test assertions and returns nil for all operations.
// Replace with a real implementation when integrating real git.
type Stub struct {
	CreateWorktreeCalls int
	RemoveWorktreeCalls int
}

// CreateWorktree records the call and returns nil without touching any real git worktree.
func (s *Stub) CreateWorktree(_ context.Context, _, _ string) error {
	s.CreateWorktreeCalls++
	return nil
}

// RemoveWorktree records the call and returns nil without touching any real git worktree.
func (s *Stub) RemoveWorktree(_ context.Context, _ string) error {
	s.RemoveWorktreeCalls++
	return nil
}
