---
paths:
  - "**/*_test.go"
---
# TST-02: Domain Pure Unit Tests

## Rule
Domain tests use no mocks (domain has no ports to call); test invariants, constructor validation, and sentinel errors.

## Why
Domain has no I/O dependencies; mocks would be unnecessary complexity.

## Example
✅ Good:
```go
// internal/core/domain/session_test.go
func TestNewSession_EmptyName_ReturnsError(t *testing.T) {
    _, err := domain.NewSession("", "project")
    if !errors.Is(err, domain.ErrSessionEmptyName) {
        t.Fatalf("expected ErrSessionEmptyName, got %v", err)
    }
}
```

❌ Bad:
```go
func TestNewSession(t *testing.T) {
    mockRepo := &MockSessionRepository{} // WRONG: domain needs no mocks
}
```
