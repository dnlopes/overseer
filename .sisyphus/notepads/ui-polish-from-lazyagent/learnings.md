# Learnings — ui-polish-from-lazyagent

## Architecture
- Hexagonal architecture: domain has zero external deps; ports in domain package
- Constructor injection; wiring in `cmd/overseer/main.go`
- Vertical slices: each feature spans domain/{feat}/, service/{feat}/, adapters/primary/tui/{feat}/
- All file writes atomic via `internal/shared/paths.AtomicWrite`
- All persistent paths XDG-compliant via `internal/shared/paths`
- All log writes go to log file (never stderr/stdout in TUI mode)

## TUI Architecture
- Each pane/form is a separate BubbleTea sub-model with Init/Update/View
- Styles defined ONLY in `tui/styles/styles.go` via `New()` function
- MUST NOT define new styles outside the styles registry
- Uses `charm.land/lipgloss/v2` (NOT github.com/charmbracelet/lipgloss v1)
- Uses `charm.land/bubbles/v2/help` for help bar (NOT custom rendering)

## Key Files
- `internal/adapters/primary/tui/styles/styles.go` — Styles struct + New()
- `internal/adapters/primary/tui/dashboard/model.go` — Main composition
- `internal/adapters/primary/tui/session/list.go` — Sessions list sub-model
- `internal/adapters/primary/tui/status/bar.go` — Status bar sub-model
- `internal/adapters/primary/tui/help/bar.go` — Help bar sub-model
- `internal/adapters/primary/tui/preview/pane.go` — Preview pane sub-model
- `internal/testutil/golden/golden.go` — Existing golden test helper (strips ANSI)

## LazyAgent Reference
- Located at `/Users/dnl/repos/dnlopes/lazyagent`
- Uses lipgloss v1 (NOT v2) — API differs from overseer
- Theme struct at `internal/ui/theme.go` (31 fields; overseer uses 15)
- Dark palette at `internal/ui/theme_dark.go`
- Styles at `internal/ui/styles.go`

## Current Overseer Problems
- Only 4 colors: #7D56F4 focus, #73F59F accent, #383838 subtle, #FF6B6B error
- `styles.go:85` uses `lipgloss.HiddenBorder()` for blurred panes → panes disappear (worst bug)
- No title bar / branding
- No key badges
- Forms are plain centered text inputs (no modal styling)
- Status bar is bare text with separators

## Test Infrastructure
- `internal/testutil/golden/` — `xgolden.RequireEqual` (strips ANSI)
- `internal/testutil/teatest/` — e2e harness for Bubble Tea models
- Run tests: `make test`
- Build: `make build`

## lipgloss v2 Color API (critical difference from v1)
- `lipgloss.Color` is a FUNCTION in v2, NOT a type: `func Color(s string) color.Color`
- Returns `color.RGBA` for hex strings (#RRGGBB) — which is comparable
- Theme struct fields must be `color.Color` (from `image/color`), not `lipgloss.Color`
- `color.Color` interface values are comparable when backed by the same concrete type (color.RGBA)
- This means `Theme` struct comparison with `==` works correctly across calls with same hex values

## Theme Package Structure
- `theme.go`: `type Theme struct` (15 `color.Color` fields) + `func LoadTheme(name string) Theme`
- `theme_dark.go`: `func DarkTheme() Theme` calling `lipgloss.Color()` for each of 15 fields
- NO palette.go, NO theme_light.go, NO activity colors, NO Apply method
- TitleSubtext uses "#E0E7FF" (per decisions.md), NOT "#9CA3AF" from lazyagent

## Task 4 — ANSI-preserving golden helper
- Added `internal/testutil/golden.RequireEqualColor(t, name, actual)` for golden assertions that preserve ANSI escape sequences under `testdata/golden/color/{name}.golden`.
- The helper shares the same `-update` flag contract as Charm golden tests and uses atomic writes for updates.
- Verification evidence captured in `.sisyphus/evidence/task-4-color-diff.txt` and `.sisyphus/evidence/task-4-update.txt`.

## Task 9 — Modal renderer

- `lipgloss.WithWhitespaceBackground` does NOT exist in lipgloss v2.0.0
- Use `lipgloss.WithWhitespaceStyle(s Style)` instead (takes a full Style, not a color)
- To keep modal.go free of `lipgloss.NewStyle()` (rule C7), added `Modal.OverlayStyle lipgloss.Style` to `styles.go` — pre-computed as `lipgloss.NewStyle().Background(theme.OverlayBg)`
- `lipgloss.Place` signature: `Place(width, height int, hPos, vPos Position, str string, opts ...WhitespaceOption) string`
- Evidence: `.sisyphus/evidence/task-9-modal-tests.txt`, `.sisyphus/evidence/task-9-rule-check.txt`

## Task 13 — help bar theme wiring

- `bubbles/v2/help.Styles` fields: `ShortKey`, `ShortDesc`, `ShortSeparator`, `FullKey`, `FullDesc`, `FullSeparator`, `Ellipsis` (different from v1)
- `NewHelpBar` signature changed to `(registry *Registry, s *styles.Styles)` — nil-safe for tests
- Dashboard golden files were stale after previous task re-skins; update with `go test ./... -update`
- The multiline `bubblehelp.Styles{}` literal doesn't match single-line grep patterns — document this in evidence file

## Tasks 15 & 16 — Re-skin create/rename forms as modals

- Both `create_form.go` and `rename_form.go` now call `components.Modal(m.styles, body, 0, 0)` instead of `m.styles.Form.Container.Render(body)`
- `Modal(s, body, w, h)` currently ignores `w` and `h` — passes `0, 0` is safe
- `Modal.Box` has `Padding(1, 3)` vs `Form.Container`'s `Padding(1, 2)` — makes modal 2 columns wider
- Dashboard golden for `TestDashboard_OpenCreate` needed updating after the padding change
- Pre-existing dashboard golden staleness (from prior empty-state task) resolved by running `go test ./dashboard/... -update`
- `xgolden.RequireEqual` golden files live in `internal/adapters/primary/tui/dashboard/testdata/`
- No goldens exist for create_form or rename_form tests (they use behavioral assertions only)

## Task 20 — tmux QA and ANSI color goldens

- `scripts/qa-tmux.sh` now exercises C9–C12 with isolated tmux sessions and captures evidence under `.sisyphus/evidence/`.
- Style color goldens live under `internal/adapters/primary/tui/styles/testdata/golden/color/` and preserve ANSI escapes via `golden.RequireEqualColor`.
- Packages that use `RequireEqualColor` with `-update` need to register the shared `update` flag in test init before running `go test ... -update`.
