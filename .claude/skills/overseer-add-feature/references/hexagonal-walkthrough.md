# Hexagonal Architecture Walkthrough

This document traces one complete feature slice through Overseer.TUI's hexagonal architecture: **creating a session** (user keypress → domain → service → adapters → TUI update).

## The Slice: Session Create

### 1. User Keypress → TUI

File: `internal/adapters/primary/tui/dashboard/model.go`

```go
case tea.KeyPressMsg:
    if key.Matches(msg, shared.NewSessionKey) && m.createForm == nil {
        cf := sessionui.NewCreateForm(m.styles, m.sessionsService)
        m.createForm = &cf
        return m, cf.Init()
    }
```

The dashboard detects `n` keypress and opens the create form. No business logic here — just routing.

### 2. Form Submit → tea.Cmd

File: `internal/adapters/primary/tui/session/create_form.go`

The form collects `Name` and `ProjectName`, then on submit dispatches a `tea.Cmd` that calls the service asynchronously:

```go
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

### 3. Service Use Case

File: `internal/core/service/session.go`

`SessionService.Create`:
1. Calls `domain.NewSession(req.Name, req.ProjectName)` — domain validates
2. Checks for duplicates via `s.repo.List(ctx)` — port call
3. Calls `s.tmux.CreateSession(ctx, req.Name)` — secondary adapter
4. Calls `s.git.CreateWorktree(ctx, "main", req.Name)` — secondary adapter
5. Calls `s.repo.Save(ctx, sess)` — persists via secondary adapter
6. Returns `CreateSessionResponse{Session: sess}`

All errors wrapped with `fmt.Errorf("context: %w", err)`. Domain sentinel `ErrSessionAlreadyExists` returned unwrapped.

### 4. Secondary Adapters

File: `internal/adapters/secondary/storage/store.go`

`Store.Save` persists via `paths.AtomicWrite`. Schema version tracked in `fileSchema.SchemaVersion`. Compile-time conformance: `var _ domain.SessionRepository = (*Store)(nil)`.

Files: `internal/adapters/secondary/tmux/stub.go`, `internal/adapters/secondary/git/stub.go` — currently stubs; real implementations follow the same port-only pattern.

### 5. Message Back to TUI

`SessionCreatedMsg` flows back through the event loop to `dashboard.Model.Update`:

```go
case sessionui.SessionCreatedMsg:
    m.createForm = nil
    return m, m.sessionsList.Init()
```

Dashboard closes the form and refreshes the sessions list. The typed message keeps the switch exhaustive.

## Key Principles in This Slice

- Domain ← service ← adapters (no reverse deps)
- Ports defined in domain, implemented in secondary adapters
- Service call dispatched in a `tea.Cmd`, never synchronously in `Update`
- Typed message (`SessionCreatedMsg`) for each async result
- `AtomicWrite` for crash-safe persistence
- Write the failing test before implementing each step
