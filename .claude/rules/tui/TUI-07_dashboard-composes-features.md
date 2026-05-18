---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-07: Dashboard Composes Features

## Rule
`dashboard.Model` owns layout + routing and embeds feature models; it never contains business logic.

## Why
Separation of concerns: dashboard is a compositor, not a business logic owner. Business logic belongs in the service layer.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/dashboard/model.go
type Model struct {
    sessionsList sessionui.Model  // embeds feature model
    createForm   *sessionui.CreateFormModel
    activePane   Pane
    // layout state only — no business logic
}
```

❌ Bad:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // WRONG: session creation logic directly in dashboard
    sess, _ := domain.NewSession(name, project)
    m.sessions = append(m.sessions, sess)
}
```
