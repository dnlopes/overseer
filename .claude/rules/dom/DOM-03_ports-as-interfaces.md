---
paths:
  - "internal/core/domain/**/*.go"
---
# DOM-03: Ports as Interfaces

## Rule
Ports are small, single-responsibility interfaces with verb names (`SessionRepository`, `TmuxAdapter`, `GitAdapter`); they live in domain alongside the aggregate they serve.

## Why
Small interfaces are easier to mock and implement; co-location with the aggregate makes the contract discoverable.

## Example
✅ Good:
```go
// internal/core/domain/session.go — ports co-located with aggregate
type SessionRepository interface {
    Save(ctx context.Context, s Session) error
    Get(ctx context.Context, id uuid.UUID) (Session, error)
    List(ctx context.Context) ([]Session, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
type TmuxAdapter interface {
    CreateSession(ctx context.Context, name string) (tmuxID string, err error)
    KillSession(ctx context.Context, tmuxID string) error
}
```

❌ Bad:
```go
type Infrastructure interface { // WRONG: god interface mixing concerns
    SaveSession(...); GetSession(...); CreateTmux(...); RunGit(...)
}
```
