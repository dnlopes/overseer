---
paths:
  - "internal/core/service/**/*.go"
---
# SVC-02: Request/Response Pattern

## Rule
Each use-case method has `<Verb><Aggregate>Request` / `<Verb><Aggregate>Response` structs; no bare parameter lists for use-cases that take >1 logical input.

## Why
Named structs are self-documenting, extensible without breaking callers, and make test setup readable.

## Example
✅ Good:
```go
// internal/core/service/session.go
type CreateSessionRequest struct {
    Name        string
    ProjectName string
}
type CreateSessionResponse struct {
    Session domain.Session
}
func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error)
```

❌ Bad:
```go
func (s *SessionService) Create(ctx context.Context, name, project string) (domain.Session, error)
// bare params — adding a field breaks all callers
```
