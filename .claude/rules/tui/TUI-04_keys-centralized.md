---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-04: Keys Centralized

## Rule
Global key bindings and per-feature KeyMap structs live in `internal/adapters/primary/tui/shared/keys.go`; each KeyMap implements `ShortHelp() []key.Binding` and `FullHelp() [][]key.Binding`.

## Why
Centralizing keys prevents duplicate binding definitions and makes help-bar population automatic.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/shared/keys.go
type SessionsKeyMap struct {
    Up, Down, NewSession key.Binding
}
func (k SessionsKeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Up, k.Down, k.NewSession} }
func (k SessionsKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.Up, k.Down}, {k.NewSession}} }
```

❌ Bad:
```go
// Inside a model's Update method — WRONG: inline binding definition
upKey := key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up"))
```
