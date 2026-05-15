package mocks

import (
	"context"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain/session"
)

type MockSessionRepository struct {
	SaveCalls    int
	SaveErr      error
	SavedSession session.Session

	ListCalls  int
	ListResult []session.Session
	ListErr    error

	GetCalls  int
	GetResult session.Session
	GetErr    error

	DeleteCalls int
	DeleteErr   error
}

func (m *MockSessionRepository) Save(ctx context.Context, s session.Session) error {
	m.SaveCalls++
	m.SavedSession = s
	return m.SaveErr
}

func (m *MockSessionRepository) Get(ctx context.Context, id uuid.UUID) (session.Session, error) {
	m.GetCalls++
	return m.GetResult, m.GetErr
}

func (m *MockSessionRepository) List(ctx context.Context) ([]session.Session, error) {
	m.ListCalls++
	return m.ListResult, m.ListErr
}

func (m *MockSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.DeleteCalls++
	return m.DeleteErr
}
