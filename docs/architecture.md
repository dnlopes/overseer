# Architecture

## Overview

Overseer is a terminal-based TUI for managing AI agent sessions. It is built on hexagonal architecture — domain types and ports live in the core with zero external dependencies, the service layer orchestrates the domain via injected ports, and adapters (primary TUI, secondary storage/stubs) sit at the edges. The entire dependency graph is wired at startup in `cmd/overseer/main.go` with plain constructor injection. During bootstrap, three infrastructure adapters (tmux, git, agent launcher) are stubs that return canned data, enabling the TUI to be developed and tested without real system integrations.

## Directory Map

```
cmd/
  overseer/               — composition root; wires all deps, starts BubbleTea program
internal/
  core/
    domain/
      session.go          — Session entity + ports (SessionRepository, TmuxAdapter, GitAdapter, AgentLauncher) + domain errors
    service/
      session.go          — SessionService with Create/Rename/List/Reorder methods
  adapters/
    primary/
      tui/
        dashboard/        — top-level BubbleTea model + screen-local widgets (titlebar, status, preview, help)
        session/          — session list + create/rename forms
        components/       — reusable atoms (badge, divider, modal, panel)
        styles/           — centralised Lipgloss style registry + theme
    secondary/
      storage/store.go    — JSON-backed session repository (atomic writes, corruption recovery)
      tmux/stub.go        — stub TmuxAdapter (canned responses; real adapter lands as a sibling file)
      git/stub.go         — stub GitAdapter
      agent/stub.go       — stub AgentLauncher
  shared/
    config/loader.go      — YAML config loader with defaults
    logger/logger.go      — slog JSON logger wired to XDG log file
    paths/                — XDG path helpers + AtomicWrite
    errs/                 — sentinel errors (ErrNotFound, ErrNoOp, …) + Wrap/Is
  testutil/
    fixtures.go           — shared test session builders
    golden.go             — ANSI-stripping golden file helpers
    color.go              — color-preserving golden file comparison
    teatest.go            — BubbleTea test harness wrapper (fixed terminal sizing)
    mocks/                — handwritten port mocks with call counters (kept as subpackage)
```

## Layer Responsibilities

### Domain (`internal/core/domain/`)

Pure Go; only stdlib and `github.com/google/uuid` allowed. Defines entities (`Session`), ports (interfaces for `SessionRepository`, `TmuxAdapter`, `GitAdapter`, `AgentLauncher`), and sentinel domain errors. Zero I/O.

### Service (`internal/core/service/`)

One `*Service` struct per aggregate. Constructor-injected ports. Each method takes a typed Request and returns a typed Response plus error. Validates the request, drives the domain, coordinates ports in order — domain first, side effects last. No BubbleTea, no adapter imports.

### Primary Adapters (`internal/adapters/primary/`)

BubbleTea TUI only. Each screen is its own package; screen-local widgets live as sibling files within that screen's package. Reusable atoms live in `tui/components/`. Styles come exclusively from the central style registry.

### Secondary Adapters (`internal/adapters/secondary/`)

Implement ports defined in the domain. Allowed to import third-party libs; must translate library errors to domain errors at the boundary. Must never leak library types to callers. Stub adapters provide full interface satisfaction with canned responses.

### Shared (`internal/shared/`)

Bootstrap utilities and cross-cutting helpers that *do not* implement a domain port. Includes config loading, logger construction, XDG path resolution, and sentinel errors.

## Conventions

Standardization is enforced through file shape, not per-layer rule files. These five rules cover the whole layout:

1. **One aggregate = one file** in both `core/domain/` and `core/service/`. Entity, ports, errors, service struct, request/response types, and methods all live in that file (one per layer).
2. **One secondary adapter = one package** named after the port it implements. Files inside name the variant (`stub.go`, `real.go`). Promote a variant to its own subpackage only when it grows to ≥3 files.
3. **Screen-local TUI widgets** live in their screen's package as sibling files. Promote to `tui/components/` only when actually reused across screens.
4. **Bootstrap utilities** (anything that does not implement a domain port) live in `internal/shared/`, never in `secondary/`.
5. **`cmd/overseer/main.go` is the only place** that imports across all layers. No other file may know about both adapters and services.

Bonus: no `pkg/` directory. Everything internal lives under `internal/` or `cmd/`.

## Dependency Direction

```
cmd/overseer (composition root)
        │
        ├──▶ adapters/primary/tui ──calls──▶ service
        │
        └──▶ adapters/secondary/* ◀──injects── service
                                                  │
                                                  ▼
                                              core/domain
                                              (ports + entities)
```

No layer imports anything above it. `cmd/overseer/main.go` is the only place that knows about all layers simultaneously.

## Adding a New Feature

New features follow a vertical-slice path: domain entity/port → service method → primary TUI adapter(s), optionally a new secondary adapter. The Conventions section above defines the file shape at each layer; the existing `session` aggregate is the worked example.

## Persistence Model

Session state is stored as a single JSON file at `$XDG_DATA_HOME/overseer/data.json` (default: `~/.local/share/overseer/data.json`). All writes are atomic: data is written to a temp file in the same directory, then renamed over the target. On load, a corrupted file is renamed to `data.json.corrupted.<unix>.json` and the application starts fresh with an empty state. The model is last-writer-wins with no background sync — the TUI holds full in-memory state and flushes on every mutation. XDG path resolution is centralised in `internal/shared/paths`.

## Stub Mode

Three infrastructure adapters are intentionally stubbed for the bootstrap phase:

- **`secondary/tmux/stub.go`** (`TmuxAdapter`): returns a fixed session ID without launching a real tmux session.
- **`secondary/git/stub.go`** (`GitAdapter`): returns a fixed branch name without running git commands.
- **`secondary/agent/stub.go`** (`AgentLauncher`): acknowledges launch without starting a real agent process.

Stubs exist so the TUI, domain, and service layers can be fully built and tested without system integration. Real adapters will replace stubs in a post-bootstrap phase by landing as sibling files (`real.go`) in the same package. See [ADR 0002](adr/0002-stub-mode-for-bootstrap.md).

## ADRs

| # | Title |
|---|-------|
| [ADR 0001](adr/0001-hexagonal-architecture-with-primary-secondary-naming.md) | Hexagonal Architecture with Primary/Secondary Naming |
| [ADR 0002](adr/0002-stub-mode-for-bootstrap.md) | Stub Mode for Bootstrap |
| [ADR 0003](adr/0003-json-file-with-atomic-write-through.md) | JSON File with Atomic Write-Through |
| [ADR 0004](adr/0004-tdd-with-teatest-for-tui-layer.md) | TDD with Teatest for TUI Layer |
