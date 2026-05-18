# Feature Examples

Two worked examples showing how to apply the add-feature workflow.

## Example A: Extend an Existing Aggregate

**Feature**: Add a `description` field to `Session`.

### Step 1 — Domain

```go
// internal/core/domain/session.go
type Session struct {
    // ... existing fields ...
    Description string   // NEW FIELD
}

func NewSession(name, project, description string) (Session, error) {
    // existing validation...
    return Session{..., Description: strings.TrimSpace(description)}, nil
}
```

### Step 2 — Ports

No new ports needed — `SessionRepository.Save` already persists the full struct.

### Step 3 — Service

```go
// internal/core/service/session.go
type CreateSessionRequest struct {
    Name        string
    ProjectName string
    Description string  // NEW FIELD
}
```

### Step 4 — Secondary Adapter

Bump `schemaVersion` to 2; add migration for v1 records in `internal/adapters/secondary/storage/store.go`:

```go
if schema.SchemaVersion < 2 {
    for i := range schema.Sessions {
        schema.Sessions[i].Description = "" // migrate v1 → v2
    }
}
```

### Step 5 — TUI

Add description input to `internal/adapters/primary/tui/session/create_form.go`. Add `DescriptionChangedMsg` to `internal/adapters/primary/tui/session/messages.go`.

### Step 6 — Dashboard

No dashboard changes needed — form is already composed.

### Step 7 — Tests

```go
// internal/core/domain/session_test.go
func TestNewSession_WithDescription_Stores(t *testing.T) { ... }

// internal/core/service/session_test.go
func TestSessionService_Create_WithDescription_Persists(t *testing.T) { ... }
```

---

## Example B: Add a Brand-New Aggregate (Hypothetical)

**Feature**: Add a `Project` aggregate with its own list pane.

> Note: This is a hypothetical example. The files below do not exist yet.

### Step 1 — Domain

Create `internal/core/domain/project.go`:
- `Project` struct with `ID`, `Name`, `Path`
- `NewProject(name, path string) (Project, error)` constructor
- `ProjectRepository` port interface
- Sentinel errors: `ErrProjectNotFound`, `ErrProjectAlreadyExists`

### Step 2 — Ports

`ProjectRepository` interface defined in `internal/core/domain/project.go`.

### Step 3 — Service

Create `internal/core/service/project.go`:
- `ProjectService` struct with `repo domain.ProjectRepository`, `logger *slog.Logger`
- Methods: `Create`, `List`, `Delete`

### Step 4 — Secondary Adapter

Create `internal/adapters/secondary/storage/project_store.go`:
- `var _ domain.ProjectRepository = (*ProjectStore)(nil)`
- `fileSchema` with `SchemaVersion: 1`
- `paths.AtomicWrite` for persistence

### Step 5 — TUI

Create `internal/adapters/primary/tui/project/`:
- `list.go` — `Model` with `Init/Update/View`
- `messages.go` — `ProjectCreatedMsg`, `CancelFormMsg`
- `create_form.go` — `CreateFormModel`
- Add `ProjectsKeyMap` to `internal/adapters/primary/tui/shared/keys.go`

### Step 6 — Dashboard

Add `projectsList project.Model` to `dashboard.Model`. Register pane with `registry.RegisterPane("projects", ...)`.

### Step 7 — Tests

Follow TDD at each layer. Use mockery-generated mocks for `ProjectRepository`.
