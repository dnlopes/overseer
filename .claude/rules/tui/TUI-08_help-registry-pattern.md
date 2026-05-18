---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-08: Help Registry Pattern

## Rule
Active-pane help uses `HelpRegistry`: each pane registers its bindings; help bar reads bindings for the active pane + globals.

## Why
Decouples pane keybindings from the help bar; adding a new pane only requires `registry.RegisterPane(...)`.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/dashboard/help.go
registry.RegisterPane("sessions", sessions.Keybindings())
// help bar automatically shows sessions bindings when sessions pane is active
```

❌ Bad:
```go
// WRONG: help bar hardcodes pane bindings
func (h HelpModel) View() string {
    return renderBindings([]key.Binding{sessionsUp, sessionsDown, sessionsNew})
}
```
