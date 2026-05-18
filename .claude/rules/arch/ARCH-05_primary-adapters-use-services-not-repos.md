# ARCH-05: Primary Adapters Use Services, Not Repos

## Rule
TUI / CLI / HTTP adapters call services; they never reach into repos or external systems directly.

## Why
Bypassing the service layer skips use-case logic (validation, ordering, side effects), creating inconsistent behavior.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/dashboard/model.go
type Model struct {
    sessionsService *service.SessionService // service injected
}
```

❌ Bad:
```go
type Model struct {
    repo domain.SessionRepository // WRONG: bypasses service layer
}
```
