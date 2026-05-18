---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-09: Handle Resize and Too-Small

## Rule
Every top-level model handles `tea.WindowSizeMsg`, recomputes child sizes, and renders a "too small" message below a configured minimum size.

## Why
Terminal windows are resizable; models that don't handle resize produce garbled output or panics.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/dashboard/model.go
case tea.WindowSizeMsg:
    return m.resize(msg)
// resize() sets m.tooSmall = msg.Width < 60 || msg.Height < 15
// View() returns s.TooSmall.Message.Render("Terminal too small") when tooSmall
```

❌ Bad:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // tea.WindowSizeMsg not handled — layout breaks on resize
    return m, nil
}
```
