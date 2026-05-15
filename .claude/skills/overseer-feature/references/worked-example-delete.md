# Worked Example: Delete Session Feature (Shape C — Direct Action)

This document walks through implementing the Delete session feature end-to-end,
following the full hexagonal architecture path. Use it as a reference when
adding any Shape C (Direct Action) feature.

**Shape:** C — Direct Action (no modal, immediate state change)  
**Key binding:** `d`  
**Use case file:** `internal/core/service/session/delete.go`  
**Steps:** 0 through 9 (10 total)

---

## Step 0: Write Failing teatest Scenario First (TDD RED)

**Motivation:** Lock the expected behavior before any implementation. A failing test is proof of intent.

**File:** `internal/adapters/primary/tui/dashboard/dashboard_test.go`

Add a new test case to the existing dashboard test suite:

```go
func TestDashboard_DeleteSession(t *testing.T) {
    sessions := []servicesession.SessionGroup{
        {
            ProjectName: "myproject",
            Sessions: []domainsession.Session{
                fixtures.MakeSession(fixtures.WithName("alpha"), fixtures.WithProject("myproject")),
                fixtures.MakeSession(fixtures.WithName("beta"),  fixtures.WithProject("myproject")),
            },
        },
    }

    deleteUC := &mocks.MockDeleteUseCase{}
    dash := dashboard.New(sessions, createUC, renameUC, reorderUC, deleteUC, styles, helpReg, logger)

    tm := teatest.NewTestModel(t, dash, teatest.WithInitialTermSize(80, 24))
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

    assert.Equal(t, 1, deleteUC.ExecuteCallCount)
    assert.Equal(t, sessions[0].Sessions[0].ID, deleteUC.LastRequest.ID)
}
```

Run: `go test ./internal/adapters/primary/tui/dashboard/...`

Expected: **compilation error** — `DeleteUseCase` does not exist yet. This is the RED state.

---

## Step 1: Add `DeleteUseCase` in `internal/core/service/session/delete.go`

**Motivation:** Implement the pure business logic before any TUI wiring. The use case is framework-agnostic.

**File touched:** `internal/core/service/session/delete.go` (new file)

```go
package session

import (
    "context"
    "log/slog"

    domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
    "github.com/dnlopes/overseer/internal/shared/errs"
)

// DeleteRequest carries the ID of the session to delete.
type DeleteRequest struct {
    ID string
}

// DeleteResponse carries the ID of the deleted session for post-action refresh.
type DeleteResponse struct {
    DeletedID string
}

// DeleteUseCase removes a session from the repository.
// It is a Shape C use case: no domain mutation, direct repository delete.
type DeleteUseCase struct {
    repo   domainsession.Repository
    logger *slog.Logger
}

// NewDeleteUseCase constructs a DeleteUseCase with required dependencies.
func NewDeleteUseCase(repo domainsession.Repository, logger *slog.Logger) *DeleteUseCase {
    return &DeleteUseCase{repo: repo, logger: logger}
}

// Execute removes the session identified by req.ID.
// Returns errs.ErrNotFound if the session does not exist.
// Returns errs.ErrInvalidInput if req.ID is empty.
func (uc *DeleteUseCase) Execute(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
    if req.ID == "" {
        return DeleteResponse{}, errs.ErrInvalidInput
    }

    // Confirm existence before deleting (prevents silent no-ops on stale IDs).
    if _, err := uc.repo.Get(ctx, req.ID); err != nil {
        return DeleteResponse{}, err
    }

    if err := uc.repo.Delete(ctx, req.ID); err != nil {
        return DeleteResponse{}, err
    }

    uc.logger.Info("session deleted", "id", req.ID)
    return DeleteResponse{DeletedID: req.ID}, nil
}
```

**Note on the `domainsession` alias:** Both this package (`internal/core/service/session`) and the domain package (`internal/core/domain/session`) are named `session`. The alias is mandatory — see the Common Pitfalls section in `SKILL.md`.

---

## Step 2: Write Failing Unit Test for the Use Case (TDD)

**Motivation:** Confirm the use case contract before implementing any TUI code.

**File touched:** `internal/core/service/session/delete_test.go` (new file)

```go
package session_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    servicesession "github.com/dnlopes/overseer/internal/core/service/session"
    "github.com/dnlopes/overseer/internal/shared/errs"
    "github.com/dnlopes/overseer/internal/testutil/fixtures"
    "github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestDeleteUseCase_HappyPath(t *testing.T) {
    s := fixtures.MakeSession(fixtures.WithName("to-delete"), fixtures.WithProject("proj"))
    repo := mocks.NewMockRepository()
    repo.Sessions[s.ID] = s

    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    uc := servicesession.NewDeleteUseCase(repo, logger)

    resp, err := uc.Execute(context.Background(), servicesession.DeleteRequest{ID: s.ID})

    require.NoError(t, err)
    assert.Equal(t, s.ID, resp.DeletedID)
    assert.Empty(t, repo.Sessions, "session must be removed from repo")
}

func TestDeleteUseCase_NotFound(t *testing.T) {
    repo := mocks.NewMockRepository()
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    uc := servicesession.NewDeleteUseCase(repo, logger)

    _, err := uc.Execute(context.Background(), servicesession.DeleteRequest{ID: "nonexistent-id"})

    assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestDeleteUseCase_EmptyID(t *testing.T) {
    repo := mocks.NewMockRepository()
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    uc := servicesession.NewDeleteUseCase(repo, logger)

    _, err := uc.Execute(context.Background(), servicesession.DeleteRequest{ID: ""})

    assert.ErrorIs(t, err, errs.ErrInvalidInput)
}
```

Run: `go test ./internal/core/service/session/...`

Expected: tests fail because `Delete(ctx, id)` is not yet on the `Repository` port. This is still RED.

---

## Step 3: Implement Use Case — Pass the Tests (GREEN)

**Motivation:** Make Step 2's tests green. The use case body was already written in Step 1 — the remaining work is ensuring the `Repository` port has a `Delete` method.

**File touched:** `internal/core/domain/session/ports.go`

Add the `Delete` method to the `Repository` interface:

```go
// Repository is the port for session persistence.
type Repository interface {
    Get(ctx context.Context, id string) (Session, error)
    List(ctx context.Context) ([]Session, error)
    Save(ctx context.Context, s Session) error
    Delete(ctx context.Context, id string) error  // ← add this
}
```

**File touched:** `internal/adapters/secondary/storage/json/store.go`

Implement `Delete` on the JSON store:

```go
func (s *Store) Delete(ctx context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, ok := s.cache[id]; !ok {
        return errs.ErrNotFound
    }

    delete(s.cache, id)
    return s.persistLocked(ctx)
}
```

**File touched:** `internal/testutil/mocks/mock_repository.go`

Implement `Delete` on the mock repository used in tests:

```go
func (m *MockRepository) Delete(ctx context.Context, id string) error {
    if _, ok := m.Sessions[id]; !ok {
        return errs.ErrNotFound
    }
    delete(m.Sessions, id)
    return nil
}
```

Run: `go test ./internal/core/service/session/...`

Expected: **all tests pass** (GREEN).

---

## Step 4: Register `d` Keybinding via Help Registry

**Motivation:** Every user-visible action must appear in the help bar. Missing registration = invisible feature.

**File touched:** `internal/adapters/primary/tui/dashboard/dashboard.go` (or wherever the dashboard constructs the help registry keys)

In the dashboard constructor or init, register the delete binding:

```go
helpReg.Register("session", "list", key.NewBinding(
    key.WithKeys("d"),
    key.WithHelp("d", "delete"),
))
```

The `"session"` pane name must match the pane identifier used in the dashboard focus enum (`focusSession`, etc.). The `"list"` sub-pane identifies the context where this binding is active.

After this step, pressing `?` in the session pane must show `d  delete` in the help bar.

---

## Step 5: Wire `d` in the Dashboard Update Function

**Motivation:** Connect the keypress to the use case dispatch.

**File touched:** `internal/adapters/primary/tui/dashboard/dashboard.go`

Locate the session pane key switch inside `Update()` and add the `"d"` case:

```diff
 // Inside Update(), session pane key handling:
 switch msg.String() {
 case "j", "J":
     return m, m.reorderSession(selected, servicesession.DirectionDown)
 case "k", "K":
     return m, m.reorderSession(selected, servicesession.DirectionUp)
+case "d":
+    if selected != nil {
+        return m, m.deleteSession(selected.ID)
+    }
 case "r":
     m.modal = newRenameModal(selected)
     return m, nil
 }
```

Add the `deleteSession` helper method on `*Model`:

```go
func (m *Model) deleteSession(id string) tea.Cmd {
    return func() tea.Msg {
        resp, err := m.deleteUC.Execute(context.Background(),
            servicesession.DeleteRequest{ID: id})
        if err != nil {
            return sessionDeleteErrMsg{err: err}
        }
        return sessionDeletedMsg{id: resp.DeletedID}
    }
}
```

Add the message types to `messages.go` (or inline in the dashboard file — follow whatever convention the codebase uses):

```go
// SessionDeletedMsg is sent when a session is successfully deleted.
// Exported so external test packages can type-switch on it.
type SessionDeletedMsg struct {
    ID string
}

// sessionDeleteErrMsg is internal; errors are shown in the status bar only.
type sessionDeleteErrMsg struct {
    err error
}
```

Handle the messages in the dashboard `Update()` case:

```go
case SessionDeletedMsg:
    m.sessions = removeFromGroups(m.sessions, msg.ID)
    m.sessionsList.SetGroups(m.sessions)
    return m, nil

case sessionDeleteErrMsg:
    m.statusBar.SetError(msg.err.Error())
    return m, nil
```

---

## Step 6: No Confirmation Modal Needed (Direct Action = Shape C)

**Motivation:** Confirm the design decision explicitly in the implementation record.

Delete follows **Shape C (Direct Action)** — keypress triggers immediate state change with no modal. This decision is intentional:

- Bootstrap sessions are cheap to recreate
- The action scope is clearly visible (one highlighted row)
- Overseer does not have an undo mechanism in v1

**No additional files to create for this step.** If a future version requires a confirmation dialog, it would upgrade Delete to **Shape A (Form-Driven)** and add a `delete_confirm.go` modal following the same pattern as `create_form.go`.

---

## Step 7: Wire in `cmd/overseer/main.go`

**Motivation:** The composition root is the single place where all dependencies are injected. No DI framework — explicit constructor calls only.

**File touched:** `cmd/overseer/main.go`

Find the block where use cases are constructed (after storage and logger are initialised) and add two lines:

```diff
 createUC  := servicesession.NewCreateUseCase(repo, tmuxAdapter, gitAdapter, agentAdapter, logger)
 renameUC  := servicesession.NewRenameUseCase(repo, logger)
 reorderUC := servicesession.NewReorderUseCase(repo, logger)
+deleteUC  := servicesession.NewDeleteUseCase(repo, logger)
```

Then pass `deleteUC` into the dashboard constructor:

```diff
-dash := tuidashboard.New(sessions, createUC, renameUC, reorderUC, styles, helpReg, logger)
+dash := tuidashboard.New(sessions, createUC, renameUC, reorderUC, deleteUC, styles, helpReg, logger)
```

Update the `tuidashboard.New` signature accordingly in `dashboard.go`.

---

## Step 8: Update AGENTS.md If New Patterns Introduced

**Motivation:** Keep the knowledge base current so future implementers learn from completed work.

**Assessment:** Delete follows the existing **Shape C (Direct Action)** pattern fully established by the Reorder feature (T15). No new conventions are introduced. No AGENTS.md updates needed.

If your feature does introduce a new pattern — for example, the first use of a confirmation dialog, the first streaming operation, or the first new secondary adapter type — then:

1. Add a one-paragraph summary to the relevant layer's `AGENTS.md`
2. Add an entry to `.sisyphus/notepads/overseer-bootstrap/learnings.md`
3. Reference the new file paths so future implementers can find the canonical example

---

## Step 9: Run `make test && make lint` — Expected Passing Output

**Motivation:** Nothing ships unless it's green. Tests must pass. Linter must be clean.

```bash
make test
```

Expected output (abbreviated):

```
ok  github.com/dnlopes/overseer/internal/core/domain/session       0.012s
ok  github.com/dnlopes/overseer/internal/core/service/session      0.018s
ok  github.com/dnlopes/overseer/internal/adapters/primary/tui/dashboard  0.045s
ok  github.com/dnlopes/overseer/internal/adapters/secondary/storage/json  0.031s
```

```bash
make lint
```

Expected output:

```
golangci-lint run ./...
(no output — zero warnings)
```

If `make lint` reports an unused variable or missing error check, fix it before committing. Commit only when both commands exit 0.

Commit message for this feature:

```
feat(session): add Delete use case with Shape C keybinding
```
