---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-12: Key Matching via key.Matches()

## Rule
`Update()` switch cases on `tea.KeyPressMsg` must use `key.Matches(msg, binding)` — never compare raw key strings like `msg.String() == "enter"` or `msg.Type == tea.KeyCtrlC`.

## Why
`key.Matches` respects the binding's full key set and help text; raw string comparisons bypass the centralized key registry and break when bindings change.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/dashboard/model.go
case tea.KeyPressMsg:
    if key.Matches(msg, shared.QuitKey) {
        return m, tea.Quit
    }
    if key.Matches(msg, shared.NewSessionKey) {
        // ...
    }
```

❌ Bad:
```go
case tea.KeyPressMsg:
    if msg.String() == "q" || msg.String() == "ctrl+c" { // raw string — WRONG
        return m, tea.Quit
    }
```
