---
paths:
  - "internal/adapters/primary/tui/**/*.go"
---
# TUI-06: Service Calls Wrapped in tea.Cmd

## Rule
Service calls happen inside `tea.Cmd` closures that return a typed message; models NEVER call services synchronously in `Update` or `View`.

## Why
Synchronous service calls block the event loop, causing UI freezes and unresponsive input.

## Example
✅ Good:
```go
// internal/adapters/primary/tui/session/create_form.go
func createSessionCmd(svc *service.SessionService, req service.CreateSessionRequest) tea.Cmd {
    return func() tea.Msg {
        resp, err := svc.Create(context.Background(), req)
        if err != nil {
            return errMsg{err}
        }
        return SessionCreatedMsg{Session: resp.Session}
    }
}
```

❌ Bad:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    resp, err := m.svc.Create(ctx, req) // WRONG: blocks event loop
}
```
