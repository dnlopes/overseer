---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-05: Typed Messages, One Per Result

## Rule
Each async result is its own typed message defined in the feature's `messages.go`; no generic `EventMsg` with a string discriminator.

## Why
Typed messages make `Update` switch cases exhaustive and compiler-checked; string discriminators are stringly-typed and error-prone.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/session/messages.go
type SessionCreatedMsg struct{ Session domain.Session }
type SessionRenamedMsg struct{ Session domain.Session }
type CancelFormMsg struct{}
```

❌ Bad:
```go
type EventMsg struct {
    Type string      // "session_created", "session_renamed" — stringly-typed
    Data interface{}
}
```
