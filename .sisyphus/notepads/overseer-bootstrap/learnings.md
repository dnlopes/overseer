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
