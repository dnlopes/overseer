---
paths:
  - "internal/core/domain/**/*.go"
---
# DOM-04: Sentinel Errors

## Rule
Errors that callers must distinguish are sentinel `var Err...` at package level; callers use `errors.Is`.

## Why
Sentinel errors enable precise error handling without string matching; `errors.Is` works through wrapping chains.

## Example
✅ Good:
```go
// internal/core/domain/session.go
var (
    ErrSessionEmptyName     = errors.New("session name cannot be empty")
    ErrSessionAlreadyExists = errors.New("session already exists")
    ErrSessionNotFound      = errors.New("session not found")
)

// caller:
if errors.Is(err, domain.ErrSessionAlreadyExists) { ... }
```

❌ Bad:
```go
return fmt.Errorf("session already exists") // callers must string-match
```
