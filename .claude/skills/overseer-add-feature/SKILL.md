---
name: overseer-add-feature
description: Step-by-step hexagonal feature-addition workflow for Overseer.TUI. Activate when adding a new feature, implementing a feature in Overseer, building something in Overseer.TUI, creating a new aggregate, or extending Overseer with new functionality.
---

# Overseer Add Feature

## When to Use

Activate this skill when you are about to add, extend, or implement any feature in Overseer.TUI. It complements (does not replace) `bubbletea-designer` and `bubbletea-maintenance` — those skills provide generic Bubble Tea guidance; this skill provides the Overseer-specific workflow.

**Precedence**: When this skill conflicts with `bubbletea-designer` or `bubbletea-maintenance`, Overseer rules WIN.

## Trigger Phrases

- "add a new feature"
- "implement \<X\> feature in Overseer"
- "build \<X\> in Overseer.TUI"
- "create a new \<aggregate\>"
- "extend Overseer with"

## Workflow

Follow this 7-step inside-out order.

### Step 1 — Domain Shape
Identify the domain shape: new aggregate or extend existing?
- New aggregate: create `internal/core/domain/<aggregate>.go` with `NewXxx()` constructor, sentinel errors, and port interfaces.
- Extend existing: add fields/methods to the existing aggregate.

### Step 2 — Port Interfaces
Define or extend port interfaces if new external collaborators are needed.
- Interfaces live in domain alongside the aggregate.
- Keep interfaces small and single-responsibility.

### Step 3 — Service Use Case
Implement the service use-case method(s).
- One `<Aggregate>Service` struct; add a new method per use-case verb.
- Use `<Verb><Aggregate>Request` / `<Verb><Aggregate>Response` structs.
- `ctx context.Context` first parameter, passed through to all ports.
- Wrap errors with `fmt.Errorf("context: %w", err)`; return domain sentinels unwrapped.
- Inject `*slog.Logger` via constructor.

### Step 4 — Secondary Adapter(s)
Implement secondary adapter(s) if new ports were added.
- One package per technology; implement only the port interface.
- Use `paths.AtomicWrite` for file mutations.
- Include schema version on persisted data.
- Rename corrupt files on parse failure; never delete.
- Add `var _ domain.Port = (*Impl)(nil)` compile-time check.

### Step 5 — TUI Feature Package
Add or extend the TUI feature package.
- Package shape: `model.go`, `messages.go`, optional `*_form.go`.
- Components are pure functions in `components/`; stateful models in feature packages.
- All styles from `*styles.Styles`; never `lipgloss.NewStyle()` inline.
- Keys in `shared/keys.go` with `ShortHelp()`/`FullHelp()`; match with `key.Matches()`.
- One typed message per async result in `messages.go`.
- Service calls wrapped in `tea.Cmd` closures.

### Step 6 — Dashboard Composition
Compose the new feature into the dashboard.
- `dashboard.Model` embeds the feature model; routes messages; no business logic.
- Register pane keybindings with `HelpRegistry`.
- Handle `tea.WindowSizeMsg` and `tooSmall` threshold.

### Step 7 — Tests (TDD — RED first)
Write tests at each layer following strict TDD.
- Domain: pure unit tests, no mocks.
- Service: mockery-generated mocks.
- Secondary: integration tests with `t.TempDir()`.
- TUI: `teatest/v2` + golden files.
- Naming: `TestSubject_Scenario_ExpectedOutcome`.

## Common Pitfalls

- **Don't call services synchronously from `Update()`** — wrap in `tea.Cmd`
- **Don't `lipgloss.NewStyle()` inside a component** — use `*styles.Styles`
- **Don't put domain logic in adapters** — validation belongs in `NewXxx()`
- **Don't wrap domain sentinel errors** — return them unwrapped for `errors.Is`
- **Don't use `errs.Wrap`** — it's deprecated; use `fmt.Errorf("ctx: %w", err)`
- **Don't write tests after implementation** — RED-GREEN-REFACTOR

## Example Walk-through

**Feature**: Add a `description` field to Session.

1. **Domain**: Add `Description string` to `Session`; update `NewSession()` to accept and validate it.
2. **Ports**: No new ports needed — `SessionRepository.Save` already persists the full struct.
3. **Service**: Add `Description string` to `CreateSessionRequest`; pass through to `domain.NewSession`.
4. **Secondary**: Bump `schemaVersion` to 2; add migration for v1 records (set `Description: ""`).
5. **TUI**: Add `DescriptionChangedMsg`; update `create_form.go` with a description input field.
6. **Dashboard**: No dashboard changes needed — form is already composed.
7. **Tests**: Write `TestNewSession_WithDescription_Stores` (RED), implement (GREEN), refactor.
