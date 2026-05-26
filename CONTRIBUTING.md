# Contributing to Overseer

## Development Setup

### Prerequisites

- **Go 1.24+**
- **tmux** (for integration tests)
- **golangci-lint** (for linting)

### Build & Run

```bash
# Build the binary
make build

# Run tests
make test

# Run integration tests
make test-integration

# Run linter
make lint

# Build and run the app
make run
```

### Useful Commands

| Command | What it does |
|---|---|
| `make build` | Build `bin/overseer` |
| `make test` | Unit tests with race detection |
| `make test-integration` | Integration tests (requires tmux) |
| `make update-golden` | Regenerate golden test snapshots |
| `make lint` | Run golangci-lint |
| `make run` | Build and start the app |
| `make clean` | Remove build artifacts |

## Project Structure

Overseer follows **Clean Architecture** (Ports and Adapters / Hexagonal):

```
.
├── cmd/overseer/           # Entry point
├── internal/
│   ├── core/
│   │   ├── domain/         # Business entities (Session, Project, PR, ...)
│   │   └── service/        # Use cases (create session, register project, ...)
│   ├── adapters/
│   │   ├── primary/        # Driving adapters
│   │   │   └── tui/        # Bubble Tea UI (dashboard, forms, inspector)
│   │   └── secondary/      # Driven adapters
│   │       ├── tmux/       # tmux integration
│   │       ├── git/        # Git operations
│   │       ├── github/     # GitHub CLI integration
│   │       └── storage/    # JSON file persistence
│   └── shared/
│       ├── config/         # YAML config loading
│       ├── logger/         # Logging setup
│       └── paths/          # Path resolution
```

Dependencies point inward: TUI → Services → Domain. The domain knows nothing about Bubble Tea, tmux, or GitHub CLI.

## Where to Make Changes

| What you're changing | Where to look |
|---|---|
| UI behavior, layout, styling | `internal/adapters/primary/tui/` |
| Domain rules or validation | `internal/core/domain/` |
| Business logic | `internal/core/service/` |
| External tool integration | `internal/adapters/secondary/` |
| Configuration | `internal/shared/config/` |

## Testing

- **Unit tests**: Test domain logic and services in isolation. Mock secondary adapters.
- **Golden tests**: UI components use golden files for snapshot testing. Run `make update-golden` after intentional UI changes.
- **Integration tests**: Test adapter integration with real tools. Marked with `//go:build integration`.
