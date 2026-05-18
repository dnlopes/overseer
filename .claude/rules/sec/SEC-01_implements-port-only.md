---
paths:
  - "internal/adapters/secondary/**/*.go"
---
# SEC-01: Implements Port Only

## Rule
A secondary adapter implements one domain port and contains no domain logic.

## Why
Single-responsibility keeps adapters replaceable; domain logic in adapters creates duplication and untestable code.

## Example
✅ Good:
```go
// internal/adapters/secondary/storage/store.go
var _ domain.SessionRepository = (*Store)(nil)
func (s *Store) Save(ctx context.Context, sess domain.Session) error {
    s.sessions[sess.ID] = sess
    return s.persist() // pure I/O, no validation
}
```

❌ Bad:
```go
func (s *Store) Save(ctx context.Context, sess domain.Session) error {
    if sess.Name == "" { return errors.New("name required") } // domain logic in adapter
}
```
