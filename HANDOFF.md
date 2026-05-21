# Agent Status — Handoff (Phase 2 in progress)

> Worktree: `/Users/dnl/.aoe-worktrees/cenas/agent-status` (branch: `agent-status`)
> Last known good: `make test` clean (race + cover), `go vet ./...` clean, all new tests pass

## What's done (Phase 1)

Domain port + service skeleton for agent activity tracking. Files added:

- `internal/core/domain/agent_activity.go` — `ActivityKind` enum (11 states from lazyagent), `AgentSessionRef`, `AgentActivity` value object, `AgentActivityProvider` port, 4 sentinel errors
- `internal/core/domain/agent_activity_test.go` — 7 tests, all GREEN
- `internal/core/service/agent_activity.go` — `AgentActivityService.Observe`
- `internal/core/service/agent_activity_test.go` — 4 tests using mockery, all GREEN
- `.mockery.yml` — added `AgentActivityProvider:` entry
- `internal/testutil/mocks/mock_AgentActivityProvider.go` — generated

Read `agent_activity.go` first — every exported symbol has godoc explaining intent. The CONTRACT is in the code; this doc only captures the WHY.

## Design decisions worth NOT re-deriving

1. **Discovery (lsof + pid-tree walk) lives inside the adapter, NOT as a separate port.** Reason: each agent needs its own discovery+parse pipeline anyway; splitting them into two ports doubles the contract for zero benefit. Auto-detect mode (`kind: auto`) becomes a single composite implementation.

2. **No YAML schema extension** for custom config dirs. Pure runtime discovery via lsof handles `CLAUDE_CONFIG_DIR=~/.claude-work claude` and `claude --config-dir ...` uniformly. Trade-off accepted: 0.5–3 s "cold-start" window where status shows unknown/idle before first file write. See [lazyagent activity.go](file:///tmp/lazyagent/internal/core/activity.go#L78-L116) for the resolver to mirror.

3. **Two soft-failure sentinels** returned unwrapped: `ErrAgentNotRunning` (degrade → `ActivityUnknown`) and `ErrAgentStoreNotResolved` (caller retries). Anything else wrapped with use-case context.

4. **Polling+diff lives in the scheduler, NOT in the service.** Pattern: mirror Overseer's existing PR-status job (every 60 s) but at 3 s + fsnotify, emit `AgentStatusUpdatedMsg` ONLY on transition. Three loops: fsnotify (reactive) + 3 s poll (defensive) + 1 s render tick (cosmetic, in-TUI only, no I/O). See [lazyagent ui/app.go Init()](file:///tmp/lazyagent/internal/ui/app.go#L139-L162).

5. **`AgentSessionRef` carries the tmux session id, not a PID.** PID walking happens inside the adapter via `TmuxAdapter` + `gopsutil`, transparent to retries when tmux pane restarts.

## Reference materials on disk

- `/tmp/lazyagent` — full clone. **May be gone after reboot. Re-clone with:** `git clone --depth=1 https://github.com/illegalstudio/lazyagent.git /tmp/lazyagent`
  - `internal/core/activity.go` — the two-step resolver + timeout constants (30 s tool, 2 min waiting, 10 s grace, 20 min spawning)
  - `internal/claude/jsonl.go` — Claude JSONL parser + `determineStatus()`
  - `internal/opencode/process.go` — OpenCode SQLite query patterns
- `/tmp/superset-agent` — reference for hook-based approach (NOT being used, just precedent). May be gone after reboot.

## Gotchas to inherit from lazyagent (and FIX in our adapters)

1. **macOS OpenCode default path**: lazyagent hardcodes `~/.local/share/opencode/opencode.db` everywhere. **On macOS it's actually `~/Library/Application Support/opencode/opencode.db`.** Don't replicate this bug — let lsof resolve it.
2. **Claude CWD encoding**: lazyagent's encoder replaces only `/` and `.`. **Official Claude docs say "every non-alphanumeric character"** (so `_`, spaces, `@` all → `-`). Doesn't matter for us since lsof bypasses path prediction, but worth knowing.
3. **`OPENCODE_DATA_DIR` does not exist.** Lazyagent reads it but OpenCode never honored it. Real env var is `OPENCODE_DB` (direct path) or `XDG_DATA_HOME` (data dir). Again — lsof bypasses this.
4. **OpenCode has a runtime CLI**: `opencode db path` returns the actual DB path. Useful as a verification step inside the OpenCode adapter.

## What's done (Phase 2 — Task 1 ✅)

- `internal/adapters/secondary/lsof/adapter.go` — discovery helper (NOT a domain-port impl). Public surface:
  - `New(logger) (*Adapter, error)` — verifies `lsof` and `ps` on PATH.
  - `ResolveAgentSession(ctx, rootPID) (Resolution, error)` — BFS the descendants of `rootPID` (root itself NOT inspected — it's the shell), `lsof -p <pid> -F n` per descendant, first `.jsonl` (Claude) or `.db` (not `.db-wal`/`.db-shm`, OpenCode) wins. Two sentinels: `ErrNoDescendants` (cold start) and `ErrNoAgentStore` (descendants exist but no agent file open).
  - `AgentKind` enum (`Claude`/`OpenCode`) — internal to the lsof layer; downstream callers translate as needed.
- `internal/adapters/secondary/lsof/adapter_test.go` — 8 tests using forked `sh -c "tail -f $f & wait $!"` subprocesses with Setpgid + pgroup-kill cleanup; race-clean; 77.8% coverage (uncovered branches are defensive-only, comparable to other adapters).
- **Chose `os/exec ps` over `gopsutil`** for descendant walking — matches existing codebase pattern (tmux/git use raw exec, no new deps). `ps -A -o pid=,ppid=` is portable across macOS and Linux.
- **Smoke-tested end-to-end** against live opencode processes: returns `/Users/dnl/.local/share/opencode/opencode.db` (real path, NOT the macOS-hardcoded default — confirming the lsof design dodges that bug).

## Critical finding for Task 2 (Claude JSONL parser)

**Claude on macOS does NOT keep its JSONL file open between writes.** Empirically verified: `lsof` on all live `claude` PIDs and their entire process trees returned no `.jsonl` FD on this machine. Claude appears to open-append-close per turn. On Linux Claude may behave differently; do not assume.

**OpenCode is unaffected** — its SQLite handle is persistent, so lsof always finds the `.db` (confirmed by the smoke test). The discussion below applies to the Claude parser only.

### Why all path-prediction fallbacks are wrong

The intuitive fallback is "if lsof fails, look in `~/.claude/projects/<encoded-cwd>/`" (or the 2.1.x variant: read `sessions-index.json` in that dir). Both are fragile because **Overseer doesn't control the agent's env**. Any of these break the prediction:

- Session config with `shell_command: bash -c 'CLAUDE_CONFIG_DIR=~/.claude-work claude'` (env set inside tmux, Overseer never sees it).
- A custom launcher script that re-exports `CLAUDE_CONFIG_DIR` or `--config-dir`.
- direnv, `mise`/`asdf` shims, per-project env in the user's shell rc.

Reading Overseer's own env for `CLAUDE_CONFIG_DIR` would silently give the wrong answer in all of the above. **The HANDOFF's original lsof-only design intent was correct; the trap is in the fallback.**

### Strategy A: observe-then-cache (no prediction)

Task 2's `Observe` should do exactly this:

1. Call lsof. If it returns a path, **cache it on the `AgentSessionRef`** (in-memory, in the provider struct, keyed by `SessionID`).
2. If lsof returns `ErrNoAgentStore` AND we have a cached path, read from the cached path (tail-read + `determineStatus()` as in the lazyagent reference).
3. If lsof returns a path that *differs* from the cached one → update the cache (user switched/resumed sessions).
4. If lsof returns `ErrNoAgentStore` AND no cache → return `domain.ErrAgentStoreNotResolved` (transient; scheduler retries).
5. Staleness check: `os.Stat` the cached path before reading; if missing or mtime hasn't advanced past a threshold while lsof keeps coming up empty, invalidate cache and return `ErrAgentStoreNotResolved`.

The cache bootstraps from the first lsof hit that catches Claude mid-write. After that, the cached path is the source of truth and lsof becomes a freshness check. **No path prediction at any layer** — fully transparent to `CLAUDE_CONFIG_DIR`, launcher wrappers, direnv, etc.

Cold-start cost for Claude: up to a few polls (each 3 s) before the first write is caught. Status reads as `unknown`/`idle` during this window, which is correct (the user hasn't actually started a turn yet).

## What's done (Phase 2 — Task 2 ✅)

- `internal/adapters/secondary/claudejsonl/` — Strategy A parser. Public surface:
  - `New(tmux TmuxPaneResolver, agent AgentResolver, logger) *Parser` — implements `domain.AgentActivityProvider`. `TmuxPaneResolver` is satisfied by `tmux.Adapter.GetPanePID`; `AgentResolver` by `lsof.Adapter.ResolveAgentSession`. Both interfaces are package-local — no new domain port.
  - `Parser.Observe(ctx, ref)` — three-stage resolution:
    1. tmux pane → root PID (`ErrTmuxSessionNotFound` → `domain.ErrAgentNotRunning`)
    2. lsof discovery: hit → cache + use; `ErrNoDescendants` → `domain.ErrAgentNotRunning`; `ErrNoAgentStore` → cache fallback or `domain.ErrAgentStoreNotResolved`; OpenCode-kind → wrapped error (guard)
    3. stat path → if missing, invalidate cache + `domain.ErrAgentStoreNotResolved`; else scan JSONL → `resolveActivity` → `domain.AgentActivity`
  - Cache lives in the `Parser` struct, keyed by `AgentSessionRef.SessionID`. **No path prediction** anywhere.
- `internal/adapters/secondary/claudejsonl/activity.go` — `resolveActivity` + `toolActivity` (ported from lazyagent, using `domain.ActivityKind`). Timeouts: 30 s tool/thinking, 2 min waiting, 20 min spawning.
- `internal/adapters/secondary/claudejsonl/jsonl.go` — minimal one-pass scanner; tracks only what `resolveActivity` needs (`LastActivity`, `LastSummaryAt`, last user/assistant `Status` + `CurrentTool`, most recent `tool_use`). 4 MB max line size.
- `internal/adapters/secondary/tmux/adapter.go` — added `GetPanePID(ctx, tmuxID)` using `tmux list-panes -t <id> -F #{pane_pid}` (`list-panes` does strict session matching, unlike `display-message`). NOT added to `domain.TmuxAdapter` port — kept off the contract since only `claudejsonl` and the future `opencodesqlite` parser need it (interface segregation).
- 59 new tests (57 in `claudejsonl` including 14 table-driven `toolActivity` subtests; 2 added to the tmux integration suite for `GetPanePID`). Package coverage 95.8 % for `claudejsonl` (uncovered branches are defensive: `json.Unmarshal` failure on nested message, `parseContent` non-string-non-array, `Observe` non-NotExist stat failure). `make test` clean, `go vet ./...` clean, `make test-integration` clean, `make build` clean.
- **Empirically verified** end-to-end against a live `claude --dangerously-skip-permissions` session in this worktree: parser correctly reports `Kind=reading Tool=Read` immediately after Claude invokes the Read tool, and `Kind=idle` once 30 s passes without activity. lsof never caught the JSONL FD open on macOS across ~25 samples at 0.2 s cadence — Strategy A is the only viable design here.

### Critical finding for Task 2 cold start (re-confirmed)

On this machine, `lsof` never observed Claude with the JSONL FD held during ~25 samples taken at 0.2 s intervals while Claude was actively "Stewing". This means **the cache bootstrap from a single lsof hit is unreliable on macOS** — in production, the first poll cycle that catches a real `tool_use` write will be the exception, not the norm. The parser correctly degrades to `domain.ErrAgentStoreNotResolved` during the unresolved window, and the scheduler in step 5 will retry on the next 3 s tick.

If we want a tighter status during the cold-start window, the right place to add complexity is **the scheduler, not the parser**: trigger a fsnotify watch on `~/.claude/projects` once the user first attaches to an agent, and fast-poll lsof during the first few seconds after attach. The parser stays oblivious; it just receives `Observe` calls more frequently during the bootstrap window. Path prediction remains rejected for the reasons in the Phase 1 doc.

## Phase 2 plan (continuing, in order)

1. ✅ **lsof adapter** — done.
2. ✅ **Claude JSONL parser** — done; see above.
3. **OpenCode SQLite parser** — `internal/adapters/secondary/opencodesqlite/` — same shape as `claudejsonl`. SQLite open in `mode=ro`. Query the last message + parts per session, run an equivalent `determineStatus()` per [lazyagent opencode/process.go](file:///tmp/lazyagent/internal/opencode/process.go#L305-L338). New dependency: `modernc.org/sqlite` (pure-Go, no CGO). The shared discovery primitives (`tmux.GetPanePID`, `lsof.ResolveAgentSession`) are already in place; the parser just needs to define its own `TmuxPaneResolver` + `AgentResolver` interfaces and translate `lsof.AgentKindOpenCode` paths. The lsof adapter's smoke test already proved OpenCode keeps its `.db` FD held continuously, so Strategy A simplifies: cache hits will be common, the "observed but FD closed" path rare. Tests can mirror the `claudejsonl` test layout (fakes for both deps, tempdir SQLite fixtures).
4. **Composite provider** — picks parser by file extension lsof returned. Lives in `internal/adapters/secondary/agentactivity/`. Both parsers expose the same `domain.AgentActivityProvider` interface, so the composite is a small switch by suffix of `lsof.Resolution.Path` (or by `lsof.Resolution.Kind`).
5. **Scheduler** — extend Overseer's existing scheduler (the PR-status job, see `cmd/overseer/main.go:92-122`). Add agent-activity job at 3 s. Diff against cache, emit `AgentStatusUpdatedMsg` on change.
6. **TUI integration**:
   - Add `AgentStatusUpdatedMsg{SessionID, Activity}` to `internal/adapters/primary/tui/shared/messages.go` (mirror `PRStatusUpdatedMsg`)
   - Add `agentActivity map[uuid.UUID]domain.AgentActivity` to `dashboard.Model` (mirror `prStatuses`)
   - Prepend status indicator in `renderSessionNode()` at `internal/adapters/primary/tui/session/model.go:337-351`
   - Use 4-color palette (idle/working/permission/review = neutral/amber/red/green) — visually less noisy than lazyagent's 10-color scheme for a dashboard view
   - Add Braille spinner frames (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) animated via existing 1 s render tick — Overseer may need to add this tick if not present

## How to resume

In a new session, paste this:

> Continue Overseer agent-status feature. Worktree: `/Users/dnl/.aoe-worktrees/cenas/agent-status`. Read `HANDOFF.md` at the worktree root and start Phase 2 task 3 (`opencodesqlite` parser). Phase 1 + lsof adapter + claudejsonl parser are landed — see `internal/core/domain/agent_activity.go` for the port contract, `internal/adapters/secondary/lsof/adapter.go` for the discovery helper, and `internal/adapters/secondary/claudejsonl/parser.go` for the parser pattern to mirror. Add `modernc.org/sqlite` via `go get` for pure-Go SQLite (no CGO).

That's it. The handoff doc, the godoc on exported symbols, and the open todos give you everything.

## Open todos snapshot (carried forward)

- [x] lsof + pid-tree resolver adapter
- [x] claudejsonl parser + two-step activity resolver (Strategy A: observe-then-cache, no path prediction)
- [ ] opencodesqlite parser + two-step activity resolver
- [ ] polling scheduler + `AgentStatusUpdatedMsg` wiring
- [ ] TUI status indicator in `renderSessionNode`
