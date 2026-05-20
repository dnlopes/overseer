// Package tmux provides tmux adapter implementations of the domain.TmuxAdapter port.
package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

// Compile-time interface check.
var _ domain.TmuxAdapter = (*Stub)(nil)

// Stub is a stub implementation of domain.TmuxAdapter.
// It satisfies the port interface with canned responses and records call counts for testing.
type Stub struct {
	CreateSessionCalls int
	KillSessionCalls   int
	GetSessionCalls    int
	ListSessionsCalls  int
	AttachSessionCalls int
	AttachCommandCalls int
	CapturePaneCalls   int
	ResizeWindowCalls  int

	LastStartDir     string
	LastShellCommand string
}

// CreateSession returns a deterministic canned tmux session ID of the form "tmux-stub-<name>-<uuid8>".
func (s *Stub) CreateSession(_ context.Context, name, startDir, shellCommand string) (string, error) {
	s.CreateSessionCalls++
	s.LastStartDir = startDir
	s.LastShellCommand = shellCommand
	id := uuid.New().String()[:8]
	return fmt.Sprintf("tmux-stub-%s-%s", name, id), nil
}

// KillSession records the call and returns nil without touching any real tmux session.
func (s *Stub) KillSession(_ context.Context, _ string) error {
	s.KillSessionCalls++
	return nil
}

// GetSession records the call and returns a synthetic session whose ID echoes the requested tmuxID.
func (s *Stub) GetSession(_ context.Context, tmuxID string) (domain.TmuxSession, error) {
	s.GetSessionCalls++
	now := time.Now()
	return domain.TmuxSession{
		ID:        tmuxID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ListSessions records the call and returns an empty slice.
func (s *Stub) ListSessions(_ context.Context) ([]domain.TmuxSession, error) {
	s.ListSessionsCalls++
	return []domain.TmuxSession{}, nil
}

// AttachSession records the call and returns nil without taking over the terminal.
func (s *Stub) AttachSession(_ context.Context, _ string) error {
	s.AttachSessionCalls++
	return nil
}

// AttachCommand records the call and returns a no-op /usr/bin/true command so callers
// can safely invoke Run() during tests without launching tmux.
func (s *Stub) AttachCommand(_ context.Context, _ string) (*exec.Cmd, error) {
	s.AttachCommandCalls++
	return exec.Command("true"), nil
}

// CapturePane records the call and returns a canned placeholder string so tests
// can assert content was rendered without invoking a real tmux server.
func (s *Stub) CapturePane(_ context.Context, _ string) (string, error) {
	s.CapturePaneCalls++
	return "stub capture-pane output", nil
}

// ResizeWindow records the call without touching any real tmux session.
func (s *Stub) ResizeWindow(_ context.Context, _ string, _, _ int) error {
	s.ResizeWindowCalls++
	return nil
}
