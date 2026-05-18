---
paths:
  - "internal/core/service/**/*.go"
---
# SVC-04: Canonical Error Wrapping

## Rule
Use `fmt.Errorf("context: %w", err)` for wrapping; domain sentinel errors returned unwrapped so callers can `errors.Is`.

## Why
Consistent wrapping preserves error chains; sentinel errors must not be wrapped so callers can match them precisely.

## Example
✅ Good:
```go
// internal/core/service/session.go
return CreateSessionResponse{}, fmt.Errorf("list sessions: %w", err)  // wrapped
return CreateSessionResponse{}, domain.ErrSessionAlreadyExists         // sentinel: unwrapped
```

❌ Bad:
```go
return CreateSessionResponse{}, fmt.Errorf("list sessions: %v", err)  // loses chain
return CreateSessionResponse{}, fmt.Errorf("already exists: %w", domain.ErrSessionAlreadyExists) // wraps sentinel
```
