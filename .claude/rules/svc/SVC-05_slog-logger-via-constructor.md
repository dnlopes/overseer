---
paths:
  - "internal/core/service/**/*.go"
---
# SVC-05: slog Logger via Constructor

## Rule
Constructor injection of `*slog.Logger`; methods log at INFO for use-case start/end and WARN/ERROR for failures.

## Why
Injected loggers are testable and configurable; structured logging enables filtering and alerting.

## Example
✅ Good:
```go
// internal/core/service/session.go
func NewSessionService(repo domain.SessionRepository, tmux domain.TmuxAdapter,
    git domain.GitAdapter, logger *slog.Logger) *SessionService {
    return &SessionService{repo: repo, tmux: tmux, git: git, logger: logger}
}
```

❌ Bad:
```go
func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) ... {
    log.Printf("creating session: %s", req.Name) // package-level logger
}
```
