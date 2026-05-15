package mocks

import "context"

type MockTmuxAdapter struct {
	CreateSessionCalls  int
	CreateSessionErr    error
	CreateSessionResult string

	KillSessionCalls int
	KillSessionErr   error
}

func (m *MockTmuxAdapter) CreateSession(ctx context.Context, name string) (string, error) {
	m.CreateSessionCalls++
	return m.CreateSessionResult, m.CreateSessionErr
}

func (m *MockTmuxAdapter) KillSession(ctx context.Context, tmuxID string) error {
	m.KillSessionCalls++
	return m.KillSessionErr
}
