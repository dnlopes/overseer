---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-01: Components Are Pure Functions

## Rule
`components/` exports pure functions returning `string`; NO `Init/Update/View` methods. Stateful UI lives in feature packages.

## Why
Pure functions are trivially testable and composable; they have no hidden state that can cause rendering bugs.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/components/panel.go
func Panel(s *styles.Styles, content string, focused bool) string {
    if focused {
        return s.Border.Focused.Render(content)
    }
    return s.Border.Blurred.Render(content)
}
```

❌ Bad:
```go
// WRONG: stateful model in components/
type PanelModel struct { focused bool }
func (m PanelModel) Init() tea.Cmd  { return nil }
func (m PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m PanelModel) View() string { ... }
```
