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
