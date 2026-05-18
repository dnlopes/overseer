# ARCH-04: Ports in Domain, Implementations in Secondary

## Rule
Domain defines port interfaces; secondary adapters implement them.

## Why
Domain owns the contract; adapters are interchangeable implementations — swapping storage backends requires no domain changes.

## Example
✅ Good:
```go
// internal/core/domain/session.go — port defined in domain
type SessionRepository interface {
    Save(ctx context.Context, s Session) error
    Get(ctx context.Context, id uuid.UUID) (Session, error)
    List(ctx context.Context) ([]Session, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

// internal/adapters/secondary/storage/store.go — impl in secondary
var _ domain.SessionRepository = (*Store)(nil)
```

❌ Bad:
```go
// internal/adapters/secondary/storage/store.go
type SessionRepository interface { ... } // WRONG: interface belongs in domain
```
