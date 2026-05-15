<!--
name: overseer-feature
description: Skill for adding new features to the Overseer TUI application following hexagonal architecture
version: 1.0.0
when-to-use: Load this skill when adding a new feature to the Overseer TUI application, creating a new use case, or implementing a new session action.
-->

# overseer-feature Skill

## When to Use This Skill

Load this skill when you hear any of:

- "add a feature to overseer"
- "create a new use case"
- "implement a new session action"
- "add a keybinding for X"
- "implement delete/archive/export/import"

This skill guides you through the full vertical slice: domain → service → secondary adapter (if needed) → TUI component → keybinding → wiring → AGENTS.md → tests.

For a complete worked example, see `references/worked-example-delete.md`.

---

## Step-by-Step Procedure

Every feature in Overseer follows this 9-step path. Do not skip steps. Do not reorder them.

### Step 1: Domain — define or extend the entity

**File:** `internal/core/domain/session/session.go` (or a new domain file in the same package)

Add any new fields, methods, or sentinel errors the feature requires directly on the domain entity. The domain package has zero external dependencies — only stdlib and `github.com/google/uuid`.

If the feature requires a new port (e.g., a new secondary adapter interface), add it to `internal/core/domain/session/ports.go`.

### Step 2: Use Case — implement Execute in the service layer

**File:** `internal/core/service/session/<feature>.go`

Create a new use case struct. The service package is also named `session`, which collides with the domain package — **always import the domain package as `domainsession`**:

```go
import (
    domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
)
```

Pattern: input request struct → validate → load from repo → domain logic → persist → return response.

### Step 3: Secondary Adapter (if needed)

**File:** `internal/adapters/secondary/<technology>/<feature>.go`

Only needed if the feature calls a new external system (tmux, git, agent, etc.). For bootstrap features that only read/write the JSON store, skip this step — the JSON storage adapter already implements the port.

### Step 4: TUI Component — build the BubbleTea sub-model

**File:** `internal/adapters/primary/tui/session/<feature>_form.go` (Form shapes) or extend `list.go` (Direct Action shapes)

Choose the right **Feature Shape** (see the Shape Catalog below). Implement `Init()`, `Update()`, and `View()`. Use BubbleTea v1 (`github.com/charmbracelet/bubbletea`) for sub-models; BubbleTea v2 (`charm.land/bubbletea/v2`) is only for the top-level dashboard.

### Step 5: Register Keybinding via Help Registry

**File:** `internal/adapters/primary/tui/help/registry.go`

Register the new keybinding so it appears in the help bar:

```go
r.Register("session", "pane", key.NewBinding(
    key.WithKeys("d"),
    key.WithHelp("d", "delete"),
))
```

Pane name must match the pane used in the dashboard focus enum. Missing registration = invisible feature.

### Step 6: Wire the Keybinding in the Dashboard

**File:** `internal/adapters/primary/tui/dashboard/dashboard.go`

In `Update()`, add a new case inside the active-pane switch:

```go
case "d":
    // dispatch to the delete use case or open confirmation
    cmd = m.handleDelete()
```

For Form shapes: open a modal overlay. For Direct Action shapes: call the use case immediately via a `tea.Cmd`.

### Step 7: Wire in `cmd/overseer/main.go`

**File:** `cmd/overseer/main.go`

Constructor injection only — no DI framework. Add the use case:

```go
deleteUC := servicesession.NewDeleteUseCase(repo, logger)
```

Then pass it into the dashboard (or the TUI component that needs it):

```go
dashboard := tuidashboard.New(sessions, createUC, renameUC, reorderUC, deleteUC, styles, helpRegistry, logger)
```

### Step 8: Update AGENTS.md (if new patterns introduced)

**File:** `AGENTS.md` (root) and relevant layer AGENTS.md

Only update if the feature introduces a pattern not previously documented (e.g., a new secondary adapter category, a new message type convention). If the feature follows an existing shape, a note that "no updates needed" is sufficient.

### Step 9: Run `make test && make lint`

All tests must pass. Linter must be clean. Commit only when green.

```
make test
make lint
```

Expected output: `ok  github.com/dnlopes/overseer/...` for all packages, zero linter warnings.

---

## Feature Shape Catalog

Use this catalog to identify the right implementation pattern before writing any code.

### Shape A: Form-Driven

**Pattern:** Create / Rename  
**Interaction:** User presses a key → modal overlay opens with text inputs → user fills fields and submits (Enter) or cancels (Esc)  
**When to use:** Any feature that requires named input before acting (create, rename, tag, assign)

**Bootstrap exercises:**
- `internal/adapters/primary/tui/session/create_form.go` — full two-field form with tab cycling and async submit
- `internal/adapters/primary/tui/session/rename_form.go` — single-field form with pre-populated value

**Key implementation notes:**
- Each form is a separate BubbleTea sub-model (not part of `list.go`)
- Dashboard renders it as a modal overlay using `lipgloss.Place` centred over the list pane
- On submit, the form emits a typed result message (e.g., `SessionCreatedMsg`) consumed by the dashboard
- On cancel, it emits `CancelFormMsg`
- Validation (non-empty fields) happens synchronously in `Update`; async work (use case call) is dispatched via `tea.Cmd`

### Shape B: Inline-Edit

**Pattern:** Edit-in-place without a modal  
**Interaction:** User presses a key on a selected item → the list row transforms into an editable text input in-place → Enter confirms, Esc reverts  
**When to use:** Quick single-field edits where a modal is too heavyweight (e.g., inline rename of a short label)

**DOCUMENTED only — not exercised in bootstrap.**  
Reference: standard bubbles `list` + `textinput` combination. The list model must track an `editing bool` flag and render the selected row differently when true.

### Shape C: Direct Action

**Pattern:** Reorder / Delete  
**Interaction:** User presses a key → immediate state change, no modal, no confirmation  
**When to use:** Reversible or low-risk actions where undo is feasible, or actions that are clearly scoped (move one step up/down, delete with implicit "last-chance" undo)

**Bootstrap exercises:**
- `internal/core/service/session/reorder.go` — use case that swaps Order fields
- `internal/adapters/primary/tui/session/list.go` — `J`/`K` keys dispatch `ReorderRequestMsg` from the list sub-model to the dashboard

**Key implementation notes:**
- No confirmation modal — action fires on keypress
- Dashboard receives a `ReorderRequestMsg` (or equivalent), calls the use case via `tea.Cmd`, and refreshes the list on response
- Use case returns `errs.ErrNoOp` for boundary/single-session cases; dashboard silently ignores `ErrNoOp`

### Shape D: Async/Streaming

**Pattern:** Long-running background operation  
**Interaction:** User triggers action → progress indicator appears → result arrives asynchronously via `tea.Cmd` channel → UI updates  
**When to use:** Operations that take >100ms (git operations, agent spawning, file indexing, network calls)

**DOCUMENTED only — not exercised in bootstrap.**  
Reference: `tea.Cmd` returning a `tea.Msg` after the operation completes. For streaming (logs, progress), use a `tea.Tick`-based polling loop or a Go channel fed into a recursive `tea.Cmd`.

### Shape E: Read-Only Detail View

**Pattern:** Non-interactive display  
**Interaction:** User selects item → detail panel renders structured read-only content  
**When to use:** Showing metadata, logs, or structured output without user editing

**DOCUMENTED only — not exercised in bootstrap.**  
Reference: `internal/adapters/primary/tui/preview/preview.go` — viewport-based read-only pane.

---

## Layer-Specific Templates

### Domain Entity Stub

```go
// internal/core/domain/session/session.go — add a method
func (s *Session) Archive() error {
    if s.Archived {
        return ErrAlreadyArchived
    }
    s.Archived = true
    s.UpdatedAt = time.Now()
    return nil
}
```

### Use Case Execute Stub

```go
// internal/core/service/session/delete.go
package session

import (
    "context"

    domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
    "github.com/dnlopes/overseer/internal/shared/errs"
    "log/slog"
)

type DeleteRequest struct {
    ID string
}

type DeleteResponse struct {
    DeletedID string
}

type DeleteUseCase struct {
    repo   domainsession.Repository
    logger *slog.Logger
}

func NewDeleteUseCase(repo domainsession.Repository, logger *slog.Logger) *DeleteUseCase {
    return &DeleteUseCase{repo: repo, logger: logger}
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
    if req.ID == "" {
        return DeleteResponse{}, errs.ErrInvalidInput
    }
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

### TUI Component Stub (Direct Action — Shape C)

```go
// No new file needed for Shape C — extend dashboard.go
// In dashboard Update(), inside the session pane key switch:

case "d":
    selected := m.sessions.Selected()
    if selected == nil {
        return m, nil
    }
    return m, m.deleteSession(selected.ID)

// deleteSession dispatches an async tea.Cmd:
func (m *Model) deleteSession(id string) tea.Cmd {
    return func() tea.Msg {
        resp, err := m.deleteUC.Execute(context.Background(), servicesession.DeleteRequest{ID: id})
        if err != nil {
            return sessionDeleteErrMsg{err: err}
        }
        return sessionDeletedMsg{id: resp.DeletedID}
    }
}
```

---

## MUST / MUST NOT

These rules are sourced from the root `AGENTS.md`. They apply to every feature regardless of shape.

**MUST:**
- Follow hexagonal architecture — domain has zero external dependencies; ports are defined in the domain package
- Use constructor injection — wire everything in `cmd/overseer/main.go`
- Follow TDD — write failing test first (RED), implement to pass (GREEN), refactor
- All file writes are atomic (`tmp + rename` via `internal/shared/paths.AtomicWrite`)
- All persistent paths are XDG-compliant via `internal/shared/paths`
- All log writes go to the log file — never stderr/stdout in TUI mode
- Register every new keybinding in the help registry
- Update the relevant layer's `AGENTS.md` if a new pattern is introduced

**MUST NOT:**
- Add a DI framework (`fx`, `wire`, `dig`)
- Use the `pkg/` directory
- Add a feature without registering it in the help registry
- Import `charm.land/bubbletea/v2` in sub-model files (use `github.com/charmbracelet/bubbletea` v1)
- Import `github.com/charmbracelet/bubbletea` in the dashboard (use `charm.land/bubbletea/v2`)

---

## Common Pitfalls

### 1. Package naming collision: `service/session` vs `domain/session`

Both the service and domain packages are named `session`. Without an alias, Go will complain about the redeclared import. Always alias the domain package:

```go
import (
    domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
    // service package is the current package — no import needed
)
```

This pattern appears in every use case (T13 CreateUseCase, T14 RenameUseCase, T15 ReorderUseCase, T16 ListUseCase).

### 2. Message type exports

Messages passed between sub-models and the dashboard **must be exported** so the dashboard's `Update()` can type-switch on them. The pattern in Overseer:

- Sub-model-internal messages: unexported (e.g., `createErrMsg`)
- Cross-component messages: exported (e.g., `SessionCreatedMsg`, `CancelFormMsg`, `ReorderRequestMsg`)

See `internal/adapters/primary/tui/session/messages.go` for the full list.

### 3. BubbleTea v2 import path

The top-level dashboard uses BubbleTea **v2** (`charm.land/bubbletea/v2`), NOT the standard GitHub path. Sub-models use BubbleTea **v1** (`github.com/charmbracelet/bubbletea`). Getting this wrong produces incompatible type errors at the dashboard boundary.

```go
// dashboard.go (v2)
import tea "charm.land/bubbletea/v2"

// create_form.go, rename_form.go, list.go (v1)
import tea "github.com/charmbracelet/bubbletea"
```

Go resolves imports per-file, so v1 and v2 can coexist in the same package — each file declares its own import.

### 4. `View()` return type differs between v1 and v2

- v1 sub-models: `View() string`
- v2 models (dashboard): `View() tea.View` via `tea.NewView(string)`

If you accidentally return `string` from the dashboard or `tea.View` from a sub-model, you will get a compile error.

### 5. `bubbles` import alias when package name collides

The `help` package in `github.com/charmbracelet/bubbles/help` collides with the local `internal/adapters/primary/tui/help` package. Use the alias pattern established in T22:

```go
import (
    bubblehelp "github.com/charmbracelet/bubbles/help"
)
```

Same pattern applies for `yaml`, `json`, `slog` — any time the stdlib or third-party package name collides with a local one.
