---
paths:
  - "internal/core/domain/**/*.go"
---
# DOM-02: Aggregate Constructor Validation

## Rule
Use `NewXxx(...)` factory functions that validate inputs and return `(Aggregate, error)`; invariants enforced in the constructor.

## Why
Constructors are the single enforcement point for invariants; exported fields allow marshaling while the constructor prevents invalid state.

## Example
✅ Good:
```go
// internal/core/domain/session.go
func NewSession(name, project string) (Session, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return Session{}, ErrSessionEmptyName
    }
    if len(name) > 100 {
        return Session{}, ErrSessionNameTooLong
    }
    return Session{ID: uuid.New(), Name: name, ...}, nil
}
```

❌ Bad:
```go
sess := domain.Session{Name: "", ProjectName: ""} // bypasses validation
```
