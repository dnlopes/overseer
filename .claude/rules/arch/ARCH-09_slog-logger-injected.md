# ARCH-09: Injected slog Logger

## Rule
Use `*slog.Logger` injected via constructor; never `log.Print*` or package-level loggers.

## Why
Injected loggers are testable, configurable, and structured. Package-level loggers create hidden global state that cannot be controlled per-component.

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
