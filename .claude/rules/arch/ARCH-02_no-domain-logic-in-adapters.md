# ARCH-02: No Domain Logic in Adapters

## Rule
Adapters translate; they don't decide — validation and invariants live in domain, not in adapters.

## Why
Putting business rules in adapters creates duplication and makes them untestable without I/O infrastructure.

## Example
✅ Good:
```go
// internal/core/domain/session.go
func NewSession(name, project string) (Session, error) {
    if name == "" {
        return Session{}, ErrSessionEmptyName // validation in domain
    }
}
```

❌ Bad:
```go
// internal/adapters/secondary/storage/store.go
func (s *Store) Save(ctx context.Context, sess domain.Session) error {
    if sess.Name == "" { // WRONG: domain logic in adapter
        return errors.New("name cannot be empty")
    }
}
```
