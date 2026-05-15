# Learnings — overseer-bootstrap

## [2026-05-15] Session Start
- Plan: 30 implementation tasks + 4 final review tasks
- Module path: `github.com/dnlopes/overseer`
- Go version: 1.22
- Architecture: Hexagonal with primary/secondary naming
- Config: YAML only
- Test strategy: TDD with teatest
- Persistence: JSON file with atomic writes (tmp + rename)
- XDG paths: data/config/state dirs
- No DI framework (constructor injection only)
- No `pkg/` directory
- Logging: slog to file only (never stderr in TUI mode)
- Stub mode: tmux/git/agent adapters are stubbed with canned responses
- Scaffolded root Go module with Go 1.22/toolchain 1.22.0, bootstrap main, and empty hexagonal directories using .gitkeep files.
- Evidence captured for build, vet, no-pkg check, and directory tree under .sisyphus/evidence/.
- Added root Makefile with help-driven PHONY targets for build/test/lint/run/clean/tidy, and added `coverage.out` to .gitignore.
- Verified `make help`, `make build`, `make test`, `make`, and `make clean`; saved evidence under `.sisyphus/evidence/`.
- Added `internal/shared/paths` with XDG-aware `DataDir`, `ConfigDir`, `StateDir`, file helpers, `EnsureDir`, and atomic temp-file rename writes.
- Added `internal/shared/errs` with stdlib sentinel errors plus thin `Wrap`/`Is` helpers.
- Verified shared package tests, XDG override behavior, and atomic write behavior; saved coverage and targeted test evidence under `.sisyphus/evidence/`.
- Added `internal/testutil/golden` with ANSI-stripping setup and byte reader helper; verified output stripping via test.
- Added `internal/testutil/teatest` harness wrapper around `teatest.NewTestModel` with fixed terminal sizing and golden setup; test uses a minimal dummy model.
- Added handwritten mock template docs and a session fixture placeholder for future T8 Session type.
- `go get` pulled the teatest dependency stack; compatibility required bumping transitive `x/ansi` and `x/cellbuf` via the module graph.
- Verified `go test -v ./internal/testutil/...` and saved harness/tree evidence under `.sisyphus/evidence/task-4-*.txt`.
- T5 complete: `internal/adapters/primary/tui/styles/styles.go` with 20 named styles in 9 nested structs. All styles returned from `New() *Styles` — zero package-level vars. `lipgloss.SetColorProfile(termenv.Ascii)` in TestMain strips escape sequences cleanly; HiddenBorder vs RoundedBorder renders differ even in ASCII mode (space-only vs box-drawing chars). `SetString(" | ")` on Separator pre-sets content for zero-arg `Render()` calls; `Render("x")` still works as expected (overrides). Evidence: task-5-borders-differ.txt, task-5-no-globals.txt.

- T8 complete: added `internal/core/domain/session` with pure domain Session entity, sentinel errors, and domain-owned ports; `New`/`Rename` trim and enforce 100-character name/project limits. Added `github.com/google/uuid` for domain IDs and replaced session fixture placeholder with `MakeSession` helper. Evidence: task-8-new-validation.txt, task-8-imports-clean.txt.

- T9 complete: `internal/adapters/secondary/storage/json/` — JSON-backed session.Repository with atomic tmp+rename writes, corruption recovery (rename to `.corrupted.<unix>.json` + warn via slog), full in-memory map cache. Package named `json` requires `encodingjson "encoding/json"` alias in store.go. `session.Session` has no JSON tags — marshals with PascalCase keys; `uuid.UUID` and `time.Time` both implement json.Marshaler so round-trip works. `io.Discard` (not `os.Discard`) for slog test handler. 10 unit tests + 3 integration tests (corruption recovery, 100 concurrent saves, missing parent dirs). Evidence: task-9-persistence.txt, task-9-corruption.txt, task-9-atomic.txt.

- T10 complete: `internal/adapters/secondary/config/yaml/` — YAML config loader with `Config`/`Default()`/`Load()`/`Validate()`. Package named `yaml` requires `yamlv3 "gopkg.in/yaml.v3"` alias in loader.go. Merge-defaults pattern: start with `Default()`, then `yamlv3.Unmarshal(data, &cfg)` to overlay only fields present in YAML. `gopkg.in/yaml.v3` error messages naturally contain "line X" info for parse errors. `Validate()` exported as method on `Config` for reuse. 8 unit tests covering: defaults, missing file, invalid YAML (with line info), partial YAML (defaults filled), invalid focusOnStart, valid full config, invalid minWidth, all valid focus values. Evidence: task-10-defaults.txt, task-10-invalid.txt.

- T11 complete: `internal/adapters/secondary/logger/slog/` — slog JSON logger wired to XDG log file. Package named `slog` requires `stdslog "log/slog"` alias (same trick as T9's `encodingjson` and T10's `yamlv3`). `OVERSEER_LOG_LEVEL` env var takes precedence over `level` param; `lvl.UnmarshalText([]byte(level))` parses slog level strings ("debug", "info", "warn", "error") case-insensitively. Tests override `XDG_STATE_HOME` via `t.Setenv` to redirect `paths.LogFile()` to temp dir; `t.Setenv("OVERSEER_LOG_LEVEL", "")` clears any ambient env in T1. Evidence: task-11-json-log.txt, task-11-level-env.txt.

- T12 complete: `internal/adapters/secondary/{tmux,git,agent}/stub/` — three stub adapters implementing `session.TmuxAdapter`, `session.GitAdapter`, `session.AgentLauncher`. Tmux stub uses `uuid.New().String()[:8]` for deterministic-enough canned IDs. All three packages use `package stub` matching directory name. Each has a `_test.go` in `package stub_test` with compile-time `var _ Interface = (*stub.Stub)(nil)` checks and call-counter assertions. Package doc comment carries the task-required stub disclaimer ("Replace with real implementation when integrating real X"). Evidence: task-12-compile.txt, task-12-recording.txt, task-12-no-todo.txt.

- T13 complete: `internal/core/service/session/CreateUseCase` follows service package aliasing with `domainsession` to avoid name collision. Create validates via domain factory before mocks/ports, computes per-project max order + 1, checks duplicate name+project before side effects, then calls tmux, git, and repo in order. Handwritten port mocks live in `internal/testutil/mocks` with call counters and canned errors/results. Evidence: task-13-happy.txt, task-13-duplicate.txt, task-13-order.txt.
