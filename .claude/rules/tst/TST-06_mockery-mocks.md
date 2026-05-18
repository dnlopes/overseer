---
paths:
  - "**/*_test.go"
---
# TST-06: Mocks via mockery

## Rule
Generate mocks with `mockery`; run `go generate ./...` to regenerate; never hand-write mock structs.

## Why
mockery produces consistent, interface-complete mocks from a single source of truth; hand-rolled mocks drift when port signatures change.

## Example
✅ Good:
```go
// internal/core/domain/session.go
//go:generate mockery --name=SessionRepository --output=../../../testutil/mocks --outpkg=mocks

// internal/testutil/mocks/mock_SessionRepository.go — generated, do not edit
type MockSessionRepository struct { mock.Mock }
func (m *MockSessionRepository) Save(ctx context.Context, s domain.Session) error {
    args := m.Called(ctx, s)
    return args.Error(0)
}
```

❌ Bad:
```go
// Manually written mock that silently misses a new port method
type MockSessionRepository struct { SaveErr error }
func (m *MockSessionRepository) Save(_ context.Context, s domain.Session) error { return m.SaveErr }
```
