---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-03: Styles From Single Source

## Rule
All lipgloss styles come from `*styles.Styles` passed to the component/model; NEVER call `lipgloss.NewStyle()` inside a component or feature model.

## Why
Centralizing styles enables theming; scattered `lipgloss.NewStyle()` calls make theme changes require grep-and-replace across the codebase.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/components/panel.go
func Panel(s *styles.Styles, content string, focused bool) string {
    return s.Border.Focused.Render(content)
}
```

❌ Bad:
```go
func Panel(content string, focused bool) string {
    style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()) // WRONG
    return style.Render(content)
}
```
