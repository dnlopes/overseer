---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-11: Alternate Screen Entry

## Rule
TUI entry uses the v2 alt-screen pattern: a thin `altScreenModel` wrapper whose `View()` sets `v.AltScreen = true`; do NOT use v1's `tea.WithAltScreen()` option.

## Why
Bubble Tea v2 uses `tea.View.AltScreen` field; the v1 `tea.WithAltScreen()` program option is not available in v2.

## Example
✅ Good:
```go
// cmd/overseer/main.go — wraps internal/adapters/primary/tui/dashboard/model.go
type altScreenModel struct{ inner tea.Model }
func (m altScreenModel) View() tea.View {
    v := m.inner.View()
    v.AltScreen = true
    return v
}
p := tea.NewProgram(altScreenModel{inner: dash})
```

❌ Bad:
```go
// WRONG: v1 API, not available in Bubble Tea v2
p := tea.NewProgram(dash, tea.WithAltScreen())
```
