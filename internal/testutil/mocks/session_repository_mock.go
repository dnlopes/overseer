package mocks

import (
	"context"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

type MockSessionRepository struct {
	SaveCalls    int
	SaveErr      error
	SavedSession domain.Session

	ListCalls  int
	ListResult []domain.Session
	ListErr    error

	GetCalls  int
	GetResult domain.Session
	GetErr    error

	DeleteCalls int
	DeleteErr   error
}

func (m *MockSessionRepository) Save(ctx context.Context, s domain.Session) error {
	m.SaveCalls++
	m.SavedSession = s
	return m.SaveErr
}

func (m *MockSessionRepository) Get(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	m.GetCalls++
	return m.GetResult, m.GetErr
}

func (m *MockSessionRepository) List(ctx context.Context) ([]domain.Session, error) {
	m.ListCalls++
	return m.ListResult, m.ListErr
}

func (m *MockSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.DeleteCalls++
	return m.DeleteErr
}
