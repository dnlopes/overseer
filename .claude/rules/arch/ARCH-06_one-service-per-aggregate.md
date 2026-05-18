# ARCH-06: One Service Per Aggregate

## Rule
One service struct per domain aggregate; methods are use-cases named with verbs (Create, Rename, Delete, List, Reorder).

## Why
Bounded scope prevents god-services; each service maps 1:1 to a domain aggregate, making the API self-documenting.

## Example
✅ Good:
```go
// internal/core/service/session.go
type SessionService struct { ... }

func (s *SessionService) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResponse, error)
func (s *SessionService) Rename(ctx context.Context, req RenameSessionRequest) (RenameSessionResponse, error)
func (s *SessionService) Delete(ctx context.Context, req DeleteSessionRequest) error
```

❌ Bad:
```go
type AppService struct { ... } // mixes sessions, users, settings
func (s *AppService) SaveAnything(ctx context.Context, thing interface{}) error
```
