---
paths:
  - "internal/core/domain/**/*.go"
---
# DOM-05: Value Objects Are Immutable

## Rule
Value-object types are created via constructor and never mutated after creation; mutation produces a new value.

## Why
Immutable value objects are safe to share across goroutines and prevent accidental state corruption.

## Example
✅ Good:
```go
type SessionName struct{ value string }
func NewSessionName(s string) (SessionName, error) {
    if s == "" { return SessionName{}, ErrSessionEmptyName }
    return SessionName{value: strings.TrimSpace(s)}, nil
}
// To "change" a name: create a new SessionName, don't mutate the old one
```

❌ Bad:
```go
type SessionName struct{ Value string }
name.Value = "new name" // direct mutation — WRONG for value objects
```
