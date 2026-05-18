---
paths:
  - "internal/core/service/**/*.go"
---
# SVC-06: Methods Are Use Cases

## Rule
One verb per method (`Create`, `Rename`, `Delete`, `Reorder`, `List`, `Get`); avoid CRUD-omnibus `Save`; if multiple aggregates change atomically, that's a NEW use-case method.

## Why
Use-case methods are named after business operations, not CRUD operations; this makes the service API self-documenting.

## Example
✅ Good:
```go
// internal/core/service/session.go
func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error)
func (s *SessionService) Rename(ctx context.Context, req RenameSessionRequest) (RenameSessionResponse, error)
func (s *SessionService) Delete(ctx context.Context, req DeleteSessionRequest) error
```

❌ Bad:
```go
func (s *SessionService) Save(ctx context.Context, sess domain.Session) error // generic CRUD, hides intent
```
