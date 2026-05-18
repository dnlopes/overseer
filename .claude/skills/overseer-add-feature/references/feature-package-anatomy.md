# Feature Package Anatomy

This document describes the required and optional files in a TUI feature package, using `internal/adapters/primary/tui/session/` as the canonical example.

## Required Files

### `model.go` (or `list.go` for list-style features)

Contains the feature's `Model` struct and `Init/Update/View` methods.

```go
// internal/adapters/primary/tui/session/list.go
type Model struct {
    styles          *styles.Styles
    sessionsService *service.SessionService
    sessions        []domain.Session
    cursor          int
}

func New(styles *styles.Styles, svc *service.SessionService) Model { ... }
func (m Model) Init() tea.Cmd { ... }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m Model) View() string { ... }
```

One model per file. Inject the service, not the repository.

### `messages.go`

Contains all typed messages produced by this feature's async operations.

```go
// internal/adapters/primary/tui/session/messages.go
type SessionCreatedMsg struct{ Session domain.Session }
type SessionRenamedMsg struct{ Session domain.Session }
type CancelFormMsg struct{}
```

One typed message per async result. No generic `EventMsg` with a string discriminator.

## Optional Files

### `*_form.go`

A sub-model for form input. Only present when the feature has a create/edit form.

```go
// internal/adapters/primary/tui/session/create_form.go
type CreateFormModel struct {
    styles *styles.Styles
    svc    *service.SessionService
}
func NewCreateForm(styles *styles.Styles, svc *service.SessionService) CreateFormModel { ... }
func (m CreateFormModel) Init() tea.Cmd { ... }
func (m CreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m CreateFormModel) View() string { ... }
```

### Per-feature KeyMap (in `shared/keys.go`)

If the feature has unique keybindings, add a `<Feature>KeyMap` struct to `internal/adapters/primary/tui/shared/keys.go`.

```go
type SessionsKeyMap struct {
    Up, Down, NewSession key.Binding
}
func (k SessionsKeyMap) ShortHelp() []key.Binding  { ... }
func (k SessionsKeyMap) FullHelp() [][]key.Binding { ... }
```

Keys are centralized in `shared/keys.go`. Match them with `key.Matches()` in `Update` — never compare raw strings.

## Naming Conventions

| File | Convention |
|------|-----------|
| Model file | `list.go`, `model.go`, or `<feature>.go` |
| Messages | `messages.go` |
| Form | `<action>_form.go` (e.g., `create_form.go`) |
| Tests | `list_test.go`, `model_test.go` |
| Golden files | `testdata/golden/<test_name>.golden` |

## Style Source

All styles come from `*styles.Styles` passed to the model constructor. The `Styles` struct is defined in `internal/adapters/primary/tui/styles/styles.go` and instantiated via `styles.New()`. Never call `lipgloss.NewStyle()` inside a feature package.

## What Does NOT Belong Here

- `lipgloss.NewStyle()` calls — use `*styles.Styles` from `internal/adapters/primary/tui/styles/styles.go`
- Direct service calls in `Update` — wrap in `tea.Cmd`
- Business logic — belongs in service layer
- Key binding definitions — belong in `internal/adapters/primary/tui/shared/keys.go`
