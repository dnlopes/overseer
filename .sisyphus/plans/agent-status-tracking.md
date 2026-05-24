# Agent Status Tracking — Implementation Handoff

**Status:** Approved design, ready for implementation
**Author:** Planning session (Claude / Sisyphus)
**Audience:** Implementing engineer (or future LLM session)
**Approach:** Slice-by-slice, one fresh session per slice (or per 2 slices)

---

## How to Use This Document

1. Start a fresh session in this worktree.
2. Load skills: `overseer-add-feature`, `bubbletea-maintenance`.
3. Read this document end-to-end.
4. Pick the lowest-numbered unfinished slice from §10.
5. Execute it following the `overseer-add-feature` 7-step workflow.
6. Mark the slice **DONE** at the bottom of §10 with a short verification note.
7. Stop. Hand off to next session for next slice.

Do **not** combine slices into one session unless explicitly noted. Each slice is sized so a fresh session can complete it cleanly.

---

## 1. Problem & Goal

Overseer.TUI launches AI coding agents (Claude Code, OpenCode, etc.) in tmux sessions. Today the sessions list shows session names, user-assigned `Label` badges, and last-update timestamps — but **nothing about what the agent is actually doing right now**. The user has no way to glance at the list and see "this agent is waiting for my input" vs "this one crashed" vs "this one is busy".

**Goal:** Add per-session agent runtime status tracking to the sessions list, with a swappable detector abstraction so today's tmux-pane-parsing strategy can be replaced with hooks (or any other technique) without re-architecting downstream code.

---

## 2. Final Design Decisions

| Decision | Choice |
|---|---|
| State set | **5 states**: `Running`, `Waiting`, `Idle`, `Dead`, `Unknown` |
| Detection strategy (now) | Tmux pane capture + per-agent pattern matching |
| Detection strategy (future) | Hooks (Claude Code) — pluggable via the same port |
| Agents in v1 | Claude Code, OpenCode |
| Default refresh interval | **5 seconds**, configurable |
| Row highlight | **Subtle tinted background** per status (selection always wins) |
| Status bar | **Aggregate counts** ("🟢 3 · 🟡 1 · 🔴 1") |
| Dead semantics | **Agent tmux session only** — shell session is ignored for status |
| AgentType source | **`LauncherConfig.agentType`** — extends existing config; required for custom launchers |
| Glyph/emoji | Use existing `disableEmoji` duality — add new status glyphs to both variants |

---

## 3. State Specification

| State | Meaning | Glyph (no-emoji) | Emoji | Foreground | Background (subtle tint) |
|---|---|---|---|---|---|
| `Running` | Agent actively processing | `●` | 🟢 | `#10B981` green | very low-saturation green |
| `Waiting` | Agent blocked on user input (approval / prompt) | `◐` | 🟡 | `#F59E0B` amber | very low-saturation amber |
| `Idle` | Agent alive, finished, waiting for next prompt | `○` | ⚪ | `theme.Muted` | none |
| `Dead` | Agent tmux session missing or crashed | `■` | 🔴 | `#EF4444` red | very low-saturation red |
| `Unknown` | Detector failed or session just started | `?` | ❔ | `theme.Subtext` (very dim) | none |

**Tint guidance:** Backgrounds should be subtle enough that the selected-row highlight (`SelectionBg`) stands out clearly on top. Aim for ~10-15% saturation. Test with both Dark and Light themes.

---

## 4. Architecture Overview

Hexagonal, matching existing project conventions.

```
┌─────────────────────────────────────────────────────────────────┐
│ TUI (primary adapter)                                            │
│   - jobs.Scheduler runs agent-status-refresh every 5s            │
│   - dashboard routes AgentStatusesUpdatedMsg                     │
│   - session list renders status glyph + tinted row               │
│   - status bar renders aggregate counts                          │
└────────────────────────────┬────────────────────────────────────┘
                             │ shared.AgentStatusesUpdatedMsg
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ AgentStatusService (application layer)                           │
│   PollAll(ctx) → map[uuid.UUID]AgentStatus                       │
│   - For each session:                                             │
│     1. Check tmux.GetSession(agentTmuxID) → Dead if missing      │
│     2. registry.DetectorFor(sess.AgentType) → call Detect()      │
│     3. Degrade to Unknown on detector error                      │
│   - Fan-out via goroutines (bounded concurrency = 8)             │
│   - 2-second per-detector timeout                                │
└──────┬───────────────────────────────────────┬──────────────────┘
       │                                       │
       │ AgentStatusDetectorRegistry           │ TmuxAdapter
       │ (port)                                │ (existing port)
       ▼                                       ▼
┌─────────────────────┐               ┌────────────────────────┐
│ Detector Registry   │               │ Tmux adapter           │
│ (secondary adapter) │               │ (existing - reused)    │
│                     │               │                        │
│ Maps AgentType →    │               │ CapturePane(tmuxID)    │
│   Detector impl     │               │ GetSession(tmuxID)     │
└──────────┬──────────┘               └────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────┐
│ Per-agent detectors (secondary adapters)                         │
│                                                                  │
│ ClaudeCodePaneDetector ── implements AgentStatusDetector         │
│ OpenCodePaneDetector   ── implements AgentStatusDetector         │
│                                                                  │
│ Each: CapturePane → StripANSI → match patterns → AgentStatus     │
└─────────────────────────────────────────────────────────────────┘
```

**Critical invariant:** `Dead` detection happens in the **service**, not in per-agent detectors. Per-agent detectors only resolve `Running` / `Waiting` / `Idle` / `Unknown`. This means a new detector author never needs to think about process liveness.

---

## 5. Domain Model

### New file: `internal/core/domain/agent_status.go`

```go
package domain

import (
    "context"
    "errors"
    "time"
)

// AgentType identifies which agent runs in a session.
// Set on Session at creation time, copied from the chosen Launcher.
type AgentType string

const (
    AgentTypeUnknown    AgentType = "unknown"
    AgentTypeClaudeCode AgentType = "claude-code"
    AgentTypeOpenCode   AgentType = "opencode"
)

// AgentStatusKind enumerates runtime states.
type AgentStatusKind string

const (
    AgentStatusUnknown AgentStatusKind = "unknown"
    AgentStatusRunning AgentStatusKind = "running"
    AgentStatusWaiting AgentStatusKind = "waiting"
    AgentStatusIdle    AgentStatusKind = "idle"
    AgentStatusDead    AgentStatusKind = "dead"
)

// AgentStatus is a value object describing what an agent is doing right now.
type AgentStatus struct {
    Kind       AgentStatusKind
    DetectedAt time.Time
    Source     string // e.g. "claude-code/pane-parser", "claude-code/hook-file"
    Reason     string // optional human-readable: "matched 'esc to interrupt'"
}

// AgentStatusDetector is THE port. One implementation per (agent, strategy) pair.
// Examples: ClaudeCodePaneDetector, ClaudeCodeHookDetector, OpenCodePaneDetector.
type AgentStatusDetector interface {
    AgentType() AgentType
    Detect(ctx context.Context, session Session) (AgentStatus, error)
}

// AgentStatusDetectorRegistry resolves an AgentType to its detector.
type AgentStatusDetectorRegistry interface {
    DetectorFor(agentType AgentType) (AgentStatusDetector, bool)
}

// Sentinel errors
var (
    ErrAgentTypeRequired = errors.New("agent type is required for custom launchers")
    ErrAgentTypeUnknown  = errors.New("agent type is not recognized")
)
```

### Modified: `internal/core/domain/launcher.go`

Add `AgentType` field:

```go
type Launcher struct {
    DisplayName string
    Command     string
    AgentType   AgentType // NEW
}
```

`NewLauncher()` must validate `AgentType` is non-empty (caller's responsibility to set it; built-in defaults set it in `Default()`, custom ones must declare it in YAML).

### Modified: `internal/core/domain/session.go`

Add `AgentType` field:

```go
type Session struct {
    // ... existing fields
    AgentType AgentType `json:",omitempty"` // NEW
}
```

Add method:

```go
func (s *Session) AssignAgentType(t AgentType) error {
    if t == "" {
        return ErrAgentTypeRequired
    }
    s.AgentType = t
    s.UpdatedAt = time.Now()
    return nil
}
```

---

## 6. Service Layer

### New file: `internal/core/service/agent_status.go`

```go
package service

import (
    "context"
    "fmt"
    "log/slog"
    "sync"
    "time"

    "github.com/google/uuid"
    "your/module/internal/core/domain"
)

type AgentStatusService struct {
    sessions domain.SessionRepository
    tmux     domain.TmuxAdapter
    registry domain.AgentStatusDetectorRegistry
    logger   *slog.Logger

    // tuning
    fanOutConcurrency int           // default 8
    detectorTimeout   time.Duration // default 2s
}

func NewAgentStatusService(
    sessions domain.SessionRepository,
    tmux domain.TmuxAdapter,
    registry domain.AgentStatusDetectorRegistry,
    logger *slog.Logger,
) *AgentStatusService {
    return &AgentStatusService{
        sessions: sessions,
        tmux:     tmux,
        registry: registry,
        logger:   logger,
        fanOutConcurrency: 8,
        detectorTimeout:   2 * time.Second,
    }
}

type PollAllAgentStatusesRequest struct{}
type PollAllAgentStatusesResponse struct {
    Statuses map[uuid.UUID]domain.AgentStatus
}

func (s *AgentStatusService) PollAll(ctx context.Context, _ PollAllAgentStatusesRequest) (PollAllAgentStatusesResponse, error) {
    sessions, err := s.sessions.List(ctx)
    if err != nil {
        return PollAllAgentStatusesResponse{}, fmt.Errorf("list sessions: %w", err)
    }

    statuses := make(map[uuid.UUID]domain.AgentStatus, len(sessions))
    var mu sync.Mutex
    sem := make(chan struct{}, s.fanOutConcurrency)
    var wg sync.WaitGroup

    for _, sess := range sessions {
        sess := sess
        wg.Add(1)
        sem <- struct{}{}
        go func() {
            defer wg.Done()
            defer func() { <-sem }()
            status := s.pollOne(ctx, sess)
            mu.Lock()
            statuses[sess.ID] = status
            mu.Unlock()
        }()
    }
    wg.Wait()

    return PollAllAgentStatusesResponse{Statuses: statuses}, nil
}

func (s *AgentStatusService) pollOne(ctx context.Context, sess domain.Session) domain.AgentStatus {
    now := time.Now()
    agentTmuxID := sess.ID.String() + "-agent"

    // Step 1: Dead check (centralised - per-agent detectors don't handle this)
    if _, err := s.tmux.GetSession(ctx, agentTmuxID); err != nil {
        if errors.Is(err, domain.ErrTmuxSessionNotFound) {
            return domain.AgentStatus{
                Kind: domain.AgentStatusDead, DetectedAt: now,
                Source: "service/tmux-liveness", Reason: "agent tmux session not found",
            }
        }
        return domain.AgentStatus{
            Kind: domain.AgentStatusUnknown, DetectedAt: now,
            Source: "service/tmux-liveness", Reason: fmt.Sprintf("tmux error: %v", err),
        }
    }

    // Step 2: Route to detector by agent type
    detector, ok := s.registry.DetectorFor(sess.AgentType)
    if !ok {
        return domain.AgentStatus{
            Kind: domain.AgentStatusUnknown, DetectedAt: now,
            Source: "service/registry", Reason: fmt.Sprintf("no detector for agent type %q", sess.AgentType),
        }
    }

    // Step 3: Bounded detector call
    detectCtx, cancel := context.WithTimeout(ctx, s.detectorTimeout)
    defer cancel()
    status, err := detector.Detect(detectCtx, sess)
    if err != nil {
        s.logger.WarnContext(ctx, "detector failed",
            slog.String("session", sess.ID.String()),
            slog.String("agent_type", string(sess.AgentType)),
            slog.Any("err", err),
        )
        return domain.AgentStatus{
            Kind: domain.AgentStatusUnknown, DetectedAt: now,
            Source: "service/detector-error", Reason: err.Error(),
        }
    }
    return status
}
```

`PollOne(ctx, id uuid.UUID)` is the single-session variant; both share `pollOne()`.

---

## 7. Detector Adapters

### Structure

```
internal/adapters/secondary/agentstatus/
├── registry/
│   └── registry.go         # in-memory map[AgentType]Detector
├── shared/
│   ├── ansi.go             # StripANSI(string) string
│   ├── pane.go             # CaptureLastLines(ctx, tmux, tmuxID, n) ([]string, error)
│   └── patterns.go         # shared Braille / spinner regex
├── claudecode/
│   ├── detector.go         # PaneDetector struct + Detect()
│   ├── patterns.go         # Claude-specific signals
│   └── detector_test.go    # golden fixtures
└── opencode/
    ├── detector.go
    ├── patterns.go
    └── detector_test.go
```

### Detector contract

Every detector includes:

```go
var _ domain.AgentStatusDetector = (*PaneDetector)(nil)

type PaneDetector struct {
    tmux domain.TmuxAdapter
}

func NewPaneDetector(tmux domain.TmuxAdapter) *PaneDetector {
    return &PaneDetector{tmux: tmux}
}

func (d *PaneDetector) AgentType() domain.AgentType {
    return domain.AgentTypeClaudeCode // or AgentTypeOpenCode
}

func (d *PaneDetector) Detect(ctx context.Context, sess domain.Session) (domain.AgentStatus, error) {
    agentTmuxID := sess.ID.String() + "-agent"
    raw, err := d.tmux.CapturePane(ctx, agentTmuxID)
    if err != nil {
        return domain.AgentStatus{}, fmt.Errorf("capture pane: %w", err)
    }
    clean := shared.StripANSI(raw)
    lines := shared.LastNonEmptyLines(clean, 50)
    return classify(lines), nil
}
```

### Initial signal sets (refine via fixtures)

**Claude Code:**

| Kind | Signal |
|---|---|
| `Running` | last 20 lines contain `"esc to interrupt"`, `"ctrl+c to interrupt"`, OR a Braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`), OR activity verb with ellipsis (`"Working…"`, `"Thinking…"`) |
| `Waiting` | last 10 lines contain `"Do you want to make this change?"`, `"Allow"`, numbered selection `"❯ 1."` / `"❯ 2."`, OR `"(y/n)"` |
| `Idle` | bare prompt `">"` at end, OR completion phrase (`"What else can I help"`), AND no Running/Waiting signal |
| (fallback) | `Unknown` if nothing matches |

**OpenCode:**

| Kind | Signal |
|---|---|
| `Running` | `"esc to interrupt"`, spinner glyphs |
| `Waiting` | `"continue?"`, `"proceed?"`, `"enter to select"`, `"esc to cancel"`, numbered menus |
| `Idle` | `">"` / `"opencode>"` prompt visible, no Running/Waiting signal |

**IMPORTANT — confirm signals against real fixtures.** When implementing Slices 2-3, capture real panes from a running Claude Code / OpenCode session using `tmux capture-pane -p -e -t <session>:` and save as `testdata/<state>_<n>.txt` fixtures. Adjust patterns to match reality. Pattern guesses above are starting points only.

### Registry

```go
// internal/adapters/secondary/agentstatus/registry/registry.go

type Registry struct {
    byType map[domain.AgentType]domain.AgentStatusDetector
}

func New() *Registry { return &Registry{byType: map[domain.AgentType]domain.AgentStatusDetector{}} }

func (r *Registry) Register(d domain.AgentStatusDetector) {
    r.byType[d.AgentType()] = d
}

func (r *Registry) DetectorFor(t domain.AgentType) (domain.AgentStatusDetector, bool) {
    d, ok := r.byType[t]
    return d, ok
}

var _ domain.AgentStatusDetectorRegistry = (*Registry)(nil)
```

---

## 8. TUI Integration

### Messages (`internal/adapters/primary/tui/shared/messages.go`)

```go
type AgentStatusesUpdatedMsg struct {
    Statuses map[uuid.UUID]domain.AgentStatus
    Err      error
}
```

### Glyphs (`internal/adapters/primary/tui/styles/glyphs.go`)

Extend the existing `Glyphs` struct in **both** branches of `NewGlyphs(disableEmoji bool)`:

```go
// In Glyphs struct definition:
StatusRunning string
StatusWaiting string
StatusIdle    string
StatusDead    string
StatusUnknown string

// disableEmoji = true:
StatusRunning: "●", StatusWaiting: "◐", StatusIdle: "○", StatusDead: "■", StatusUnknown: "?",

// disableEmoji = false:
StatusRunning: "🟢", StatusWaiting: "🟡", StatusIdle: "⚪", StatusDead: "🔴", StatusUnknown: "❔",
```

Add helper:

```go
func (g Glyphs) AgentStatus(kind domain.AgentStatusKind) string {
    switch kind {
    case domain.AgentStatusRunning: return g.StatusRunning
    case domain.AgentStatusWaiting: return g.StatusWaiting
    case domain.AgentStatusIdle:    return g.StatusIdle
    case domain.AgentStatusDead:    return g.StatusDead
    default:                        return g.StatusUnknown
    }
}
```

### Theme (`internal/adapters/primary/tui/styles/theme.go`)

Add to `Theme` struct (palette will need new colors in each theme file: `theme_dark.go`, `theme_light.go`, `theme_dracula.go`, etc.):

```go
StatusRunningFg color.Color
StatusRunningBg color.Color // subtle tint
StatusWaitingFg color.Color
StatusWaitingBg color.Color
StatusIdleFg    color.Color
StatusDeadFg    color.Color
StatusDeadBg    color.Color
StatusUnknownFg color.Color
```

### Styles (`internal/adapters/primary/tui/styles/styles.go`)

Extend `ListRow`:

```go
ListRow struct {
    Normal, Selected, Aux, AuxSelected, Number, NumberSelected lipgloss.Style
    StatusRunning, StatusWaiting, StatusDead lipgloss.Style // NEW - bg-tinted non-selected variants
    // Idle and Unknown use Normal (no tint)
}
```

Constructor in `NewWithTheme()`:

```go
StatusRunning: lipgloss.NewStyle().Foreground(theme.Text).Background(theme.StatusRunningBg),
StatusWaiting: lipgloss.NewStyle().Foreground(theme.Text).Background(theme.StatusWaitingBg),
StatusDead:    lipgloss.NewStyle().Foreground(theme.Text).Background(theme.StatusDeadBg),
```

### Session list (`internal/adapters/primary/tui/session/model.go`)

```go
type Model struct {
    // ... existing
    statuses map[uuid.UUID]domain.AgentStatus // NEW
}

type sessionNode struct {
    // ... existing
    status domain.AgentStatusKind // NEW
}
```

In `Update()`, add case:

```go
case shared.AgentStatusesUpdatedMsg:
    if msg.Err != nil { return m, nil }
    m.statuses = msg.Statuses
    m.rebuildTree() // sessionNode.status populated from m.statuses[sess.ID]
    return m, nil
```

In `renderSessionNode()`:

1. **Prepend status glyph**: `body := prefix + glyphs.AgentStatus(item.status) + " " + item.label`
2. **Style selection** (precedence: selection > status tint > normal):
   ```go
   if focused {
       // existing selected branch unchanged
   } else {
       rowStyle := s.ListRow.Normal
       switch item.status {
       case domain.AgentStatusRunning: rowStyle = s.ListRow.StatusRunning
       case domain.AgentStatusWaiting: rowStyle = s.ListRow.StatusWaiting
       case domain.AgentStatusDead:    rowStyle = s.ListRow.StatusDead
       }
       // ... use rowStyle for the body render
   }
   ```

### Status bar (`internal/adapters/primary/tui/dashboard/status.go`)

**⚠️ Pre-existing gap discovered during review:** `StatusModel` is defined in `status.go` but is **NOT** currently wired into `dashboard.Model`. The dashboard's `Model` struct has no `status` field, no initialization, and `View()` renders only the title bar and body — there is no status bar in the rendered output today. Slice 3 must therefore include the scaffolding to wire it up, not just modify an existing wiring.

**Scaffolding work required in Slice 3:**
1. Add `status StatusModel` field to `dashboard.Model` (in `dashboard/root.go`).
2. Initialize it in the dashboard constructor (`NewModel` or equivalent) — pass in `styles.Glyphs` so `StatusModel` can render glyphs.
3. Route `tea.WindowSizeMsg` to `m.status` in `Update()` so the status bar resizes correctly.
4. Route `shared.AgentStatusesUpdatedMsg` to `m.status` in `Update()`.
5. Include `m.status.View()` in the dashboard's `View()` output. **Position: between the body and the help bar** (bottom-aligned single line). Mirror the layout pattern used by `helpBar` — fixed height 1, full width.

**Update the StatusModel itself:**

Replace hardcoded `agentStatus: "idle"`:

```go
type StatusModel struct {
    // ... existing
    aggregate map[domain.AgentStatusKind]int // NEW
    glyphs    styles.Glyphs                  // NEW
}

func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
    switch msg := msg.(type) {
    case shared.AgentStatusesUpdatedMsg:
        m.aggregate = make(map[domain.AgentStatusKind]int)
        for _, st := range msg.Statuses {
            m.aggregate[st.Kind]++
        }
    case tea.WindowSizeMsg:
        // handle width
    }
    return m, nil
}

func (m StatusModel) View() string {
    // Format: "🟢 3 · 🟡 1 · 🔴 1 · ⚪ 2"
    // Omit zero-counts to keep it compact.
    // Use m.glyphs.AgentStatus(kind) for consistency with sessions list.
}
```

### Dashboard root (`internal/adapters/primary/tui/dashboard/root.go`)

Route `AgentStatusesUpdatedMsg` to **both** the session list and the status bar. Pattern:

```go
case shared.AgentStatusesUpdatedMsg:
    var c1, c2 tea.Cmd
    m.leftPane, c1 = shared.UpdateModel(m.leftPane, msg)
    m.status,   c2 = shared.UpdateModel(m.status, msg)
    return m, tea.Batch(c1, c2)
```

### Job scheduler (`cmd/overseer/main.go`)

Mirror the existing PR-status job pattern:

```go
statusJob := jobs.Job{
    ID:       "agent-status-refresh",
    Interval: cfg.AgentStatus.RefreshInterval,
    Run: func() tea.Cmd {
        return shared.Request(
            func(ctx context.Context) (service.PollAllAgentStatusesResponse, error) {
                return agentStatusSvc.PollAll(ctx, service.PollAllAgentStatusesRequest{})
            },
            func(resp service.PollAllAgentStatusesResponse, err error) tea.Msg {
                return shared.AgentStatusesUpdatedMsg{Statuses: resp.Statuses, Err: err}
            },
        )
    },
}
// Register alongside existing jobs.
```

### Detector wiring (`cmd/overseer/main.go`)

```go
registry := agentstatusRegistry.New()
registry.Register(claudecode.NewPaneDetector(tmuxAdapter))
registry.Register(opencode.NewPaneDetector(tmuxAdapter))

agentStatusSvc := service.NewAgentStatusService(sessionRepo, tmuxAdapter, registry, logger)
```

**Swapping in hooks later** is literally one line:

```go
// registry.Register(claudecode.NewPaneDetector(tmuxAdapter))
registry.Register(claudecode.NewHookDetector(hooksDir))
```

---

## 9. Configuration

### Extend `LauncherConfig` (`internal/shared/config/loader.go`)

```go
type LauncherConfig struct {
    DisplayName string `yaml:"displayName"`
    Command     string `yaml:"command"`
    AgentType   string `yaml:"agentType"` // NEW
}
```

Built-in defaults in `Default()`:

```go
Launchers: []LauncherConfig{
    {DisplayName: "OpenCode (default)", Command: "opencode", AgentType: "opencode"},
    {DisplayName: "Claude Code (default)", Command: "claude", AgentType: "claude-code"},
},
```

**Validation:** In `Load()`, after the merge step, validate each launcher:

```go
for i, l := range cfg.Launchers {
    if l.AgentType == "" {
        return Config{}, fmt.Errorf("launchers[%d] %q: agentType is required (one of: claude-code, opencode)", i, l.DisplayName)
    }
    if !isKnownAgentType(l.AgentType) {
        return Config{}, fmt.Errorf("launchers[%d] %q: unknown agentType %q", i, l.DisplayName, l.AgentType)
    }
}
```

Helper `isKnownAgentType` lives in config package; whitelist matches the `AgentType*` constants in domain.

### New top-level section

```go
type AgentStatusConfig struct {
    Enabled         bool                     `yaml:"enabled"`
    RefreshInterval time.Duration            `yaml:"refreshInterval"`
    Display         AgentStatusDisplayConfig `yaml:"display"`
}

type AgentStatusDisplayConfig struct {
    SessionList   bool   `yaml:"sessionList"`
    StatusBar     bool   `yaml:"statusBar"`
    RowHighlight  string `yaml:"rowHighlight"` // "subtle" | "none"
}
```

Default:

```go
AgentStatus: AgentStatusConfig{
    Enabled:         true,
    RefreshInterval: 5 * time.Second,
    Display: AgentStatusDisplayConfig{
        SessionList:  true,
        StatusBar:    true,
        RowHighlight: "subtle",
    },
},
```

### Example YAML

```yaml
launchers:
  - displayName: My Custom Claude
    command: claude --custom-flag
    agentType: claude-code      # REQUIRED for custom launchers
  - displayName: My OpenCode
    command: opencode --foo
    agentType: opencode

agentStatus:
  enabled: true
  refreshInterval: 5s
  display:
    sessionList: true
    statusBar: true
    rowHighlight: subtle
```

---

## 10. Slice-by-Slice Implementation Plan

> ✅ Mark each slice DONE here when finished, with a one-line verification note.

### Slice 1 — Config & Domain Foundation

**Goal:** Add the data model and config schema. Nothing user-visible yet. Unblocks all later slices.

**Scope:**
- `internal/core/domain/agent_status.go` (new) — types only, no detectors yet
- `internal/core/domain/launcher.go` — add `AgentType` field, update `NewLauncher`
- `internal/core/domain/session.go` — add `AgentType` field, add `AssignAgentType`
- `internal/shared/config/loader.go` — extend `LauncherConfig`, add `AgentStatusConfig`, defaults, validation
- `internal/core/service/session.go` — `CreateSessionRequest` carries `AgentType`, populated from selected launcher
- `internal/adapters/primary/tui/session/create_form.go` — pass selected launcher's `AgentType` into `CreateSessionRequest`
- `internal/adapters/secondary/storage/store.go` — implicit migration: existing sessions deserialize with empty `AgentType` (Go JSON `omitempty`); inference logic populates them on load. **There is no `schemaVersion` field in the storage file today** — do not invent one. Inference is: prefix-match `AgentCommand` against known defaults (`claude*` → `claude-code`, `opencode*` → `opencode`), else `AgentTypeUnknown`. Log at INFO when inference fills in a value. Inferred values persist back to disk on next mutation naturally via `Save`.

**Tests (TDD - RED first):**
- `TestNewLauncher_RequiresAgentType` — `NewLauncher("", "claude", "")` returns `ErrAgentTypeRequired`.
- `TestSession_AssignAgentType_Validates` — `AssignAgentType("")` returns error; valid type updates `UpdatedAt`.
- `TestLoad_CustomLauncherWithoutAgentType_Fails` — fixture YAML with one custom launcher missing `agentType` returns validation error containing the launcher's display name.
- `TestLoad_BuiltinDefaultsHaveAgentType` — `Default()` launchers have non-empty `AgentType` matching the constants.
- `TestLoad_DefaultsApplyWhenAgentStatusSectionMissing` — YAML with no `agentStatus` key loads successfully with default `RefreshInterval=5s`, `Enabled=true`, etc.
- `TestStore_MigratesSessionsWithoutAgentType` — write a JSON storage file (use `testdata/store_v0_no_agent_type.json` fixture) containing sessions with `agentCommand: "claude"` and no `agentType`; load via `store.Load()`; assert all sessions have `AgentType` populated correctly (claude → claude-code, opencode → opencode, unknown command → unknown). Assert INFO log was emitted (capture via test logger).
- `TestCreateSession_CapturesAgentTypeFromLauncher` — given a `CreateSessionRequest` with `AgentType: "claude-code"`, assert the persisted `Session.AgentType == "claude-code"`.

**Acceptance criteria:**
- `go build ./...` clean
- All new tests pass; all existing tests still pass
- `lsp_diagnostics` clean on every modified file
- `TestStore_MigratesSessionsWithoutAgentType` proves backward compat with old-format storage files via the committed fixture
- `TestLoad_DefaultsApplyWhenAgentStatusSectionMissing` proves backward compat with old-format config files

**Status:** ✅ DONE — 2026-05-23 — all 7 acceptance-gate tests pass; full `go test ./...` GREEN; build exit 0; lsp_diagnostics clean (only pre-existing stylistic hints on unmodified lines of model_test.go).

---

### Slice 2 — Service + Registry + Claude Code Detector

**Goal:** Build the service layer and the first detector. Detection works end-to-end in logs, no UI yet.

**Scope:**
- `internal/core/service/agent_status.go` (new) — `AgentStatusService` per §6
- `internal/adapters/secondary/agentstatus/registry/registry.go` (new)
- `internal/adapters/secondary/agentstatus/shared/{ansi.go,pane.go,patterns.go}` (new)
- `internal/adapters/secondary/agentstatus/claudecode/{detector.go,patterns.go,detector_test.go}` (new)
- `cmd/overseer/main.go` — wire registry, register Claude detector, construct service
- Optional debug: log polled statuses every 30s so you can verify by tailing logs

**Capture real fixtures FIRST:**
- Launch a Claude Code session inside Overseer
- For each state (running, waiting, idle): `tmux capture-pane -p -e -t <agent-tmux-id>: > testdata/claude_running.txt` (etc.)
- Commit fixtures alongside tests

**Tests:**
- `TestClaudeCodeDetector_Running_*` (one per fixture)
- `TestClaudeCodeDetector_Waiting_*`
- `TestClaudeCodeDetector_Idle_*`
- `TestPollAll_ReturnsDeadWhenAgentTmuxMissing` (service-level, mocked detector)
- `TestPollAll_RoutesByAgentType` (service-level, mocked detectors)
- `TestPollAll_DegradesToUnknownOnDetectorError`
- `TestPollAll_RespectsDetectorTimeout`

**Acceptance criteria:**
- `go build ./...` clean
- All tests pass with real Claude fixtures
- Manual: run Overseer with a real Claude session, tail the log, observe correct status detection across all 4 transitions (idle → running → waiting → idle)

**Status:** ✅ DONE — 2026-05-23 — full `go test -race ./...` GREEN across all 22 packages; build exit 0; `lsp_diagnostics` clean on every changed file. Live integration test (`-tags=live_tmux`, `OVERSEER_LIVE_TMUX_ID=<uuid>-agent`) verified all four transitions against a real Claude Code session: idle → running ("matched \"esc to interrupt\"") → waiting ("matched \"Do you want to proceed?\"") → idle. Detector returns wrapped error on missing pane; service-level test confirms the service intercepts and reports Dead, keeping detectors ignorant of liveness as required by §4. Slice 2 carries a temporary `agent-status-debug` scheduler job that logs status at INFO every `refreshInterval` so detection is observable end-to-end; Slice 3 will replace it with the TUI-bound `AgentStatusesUpdatedMsg` job.

---

### Slice 3 — TUI: Sessions List + Status Bar

**Goal:** Status becomes visible to the user. Claude-only (OpenCode still unknown).

**Starting state from Slice 2** (already wired in `cmd/overseer/main.go`):

- `agentStatusRegistry` (a `*registry.Registry`) holds the Claude Code pane detector. Reuse the variable; do not rename it.
- `agentStatusSvc` (a `*service.AgentStatusService`) is constructed against the existing tmux adapter + session repo. Reuse this too.
- A temporary `agent-status-debug` job (gated on `cfg.AgentStatus.Enabled`) calls `PollAll` every `cfg.AgentStatus.RefreshInterval` and logs each session's classified status at INFO. **Slice 3 deletes `buildAgentStatusDebugJob` and replaces it with `buildAgentStatusJob` (or similar) whose result handler returns `shared.AgentStatusesUpdatedMsg{Statuses: resp.Statuses, Err: err}` instead of logging.** The Bubble Tea / `shared.Request` wiring is identical to the debug job — only the wrap function changes.
- The opt-in `detector_live_test.go` (`-tags=live_tmux`, `OVERSEER_LIVE_TMUX_ID=<uuid>-agent`) is the easiest way to sanity-check live detection without spinning up the whole TUI; useful while iterating on glyph/colour choices.
- The fixture-capture methodology is documented in commit `15a3639`'s message and `internal/adapters/secondary/agentstatus/claudecode/testdata/`. Use the same approach if you want extra fixtures for golden-file rendering tests.

**Scope:**
- `internal/adapters/primary/tui/shared/messages.go` — add `AgentStatusesUpdatedMsg`
- `internal/adapters/primary/tui/styles/glyphs.go` — extend `Glyphs` (both branches)
- `internal/adapters/primary/tui/styles/theme.go` — add status palette fields
- All theme files (`theme_dark.go`, `theme_light.go`, `theme_dracula.go`, others) — populate new fields with subtle tinted colors. **Audit existing themes for completeness.**
- `internal/adapters/primary/tui/styles/styles.go` — extend `ListRow` with `StatusRunning`/`StatusWaiting`/`StatusDead`
- `internal/adapters/primary/tui/session/model.go` — `statuses` map, `sessionNode.status`, `Update` handler, `renderSessionNode` modifications
- `internal/adapters/primary/tui/dashboard/status.go` — extend `StatusModel` with `aggregate` field + glyphs + `Update` handler for `AgentStatusesUpdatedMsg` + `View` rendering
- **`internal/adapters/primary/tui/dashboard/root.go` — scaffolding**: `StatusModel` is NOT currently wired into the dashboard (see plan §8 status-bar warning). Must add `status StatusModel` field, initialize it in the constructor, route `tea.WindowSizeMsg` AND `AgentStatusesUpdatedMsg` to it in `Update()`, and include `m.status.View()` in `View()` between body and help bar. Budget ~30-60 min for this scaffolding alone.
- `cmd/overseer/main.go` — register the new `agent-status-refresh` job with the scheduler

**Tests:**
- `teatest/v2` golden file: session list with mixed-status sessions (one Running, one Waiting, one Dead, one Idle, one Unknown) — verify rendering with both emoji and no-emoji glyphs
- Golden file: status bar aggregate with mixed counts
- Unit: `renderSessionNode` precedence (selection wins over status tint)

**Acceptance criteria:**
- `go build ./...` clean, all tests pass
- Manual: launch Overseer with multiple Claude sessions in different states. Confirm glyphs render correctly in both `disableEmoji: true` and `false`. Confirm tinted backgrounds are subtle and don't fight the selection highlight. Confirm status bar aggregate updates.

**Status:** ✅ DONE — 2026-05-24 — full `go test -race -count=1 ./...` GREEN across all 22 packages; build exit 0; `lsp_diagnostics` clean on every changed file. New TUI tests cover: per-row status glyph rendering (Running ●, Waiting ◐, Dead ■, Idle ○ — no-emoji variants), fallback to Unknown `?` when no status data has arrived for a session, label-badge coexistence (status glyph + WIP badge on the same row), errored `AgentStatusesUpdatedMsg` is dropped so the previous snapshot survives, and selection-precedence (focused row carries `SelectionBg` ANSI triplet `48;2;55;48;163`, NOT `StatusRunningBg` `48;2;10;46;34`, while the unfocused sibling carries the tint). Dashboard tests cover the new status-bar wiring: `AgentStatusesUpdatedMsg` flows through `dashboard.Model.Update` → `shared.Broadcast` → both `leftPane` (for per-row glyphs/tints) and `status` (for aggregate), the aggregate renders as `● 3 · ◐ 1 · ■ 1` (no-emoji) with zero-counts omitted. `StatusModel` was rewritten: the prior stubbed workdir/branch/PR fields are gone, replaced by a `map[domain.AgentStatusKind]int` aggregate + `*styles.Styles` ref. Dashboard `View()` now lays out titlebar → blank → body → status bar → help bar (the status bar adds 1 row, reflected in `bodyHeight` math via the new `StatusBarHeight` const). All 12 themes carry the new 8-field status palette (`StatusRunningFg/Bg`, `StatusWaitingFg/Bg`, `StatusIdleFg`, `StatusDeadFg/Bg`, `StatusUnknownFg`); `TestThemes_NoZeroColors` now expects 25 fields. `cmd/overseer/main.go` swap is mechanical: `buildAgentStatusDebugJob` removed (along with the now-unused `log/slog` import) and replaced by `buildAgentStatusJob` whose wrap-fn returns `shared.AgentStatusesUpdatedMsg{Statuses: resp.Statuses, Err: err}` instead of logging — the `shared.Request` plumbing is otherwise identical. Manual smoke: launched the binary with an isolated `XDG_CONFIG_HOME`/`XDG_DATA_HOME`/`XDG_STATE_HOME` in a 200×60 tmux pane, confirmed the new status row renders between the body and the help bar (empty content when the aggregate is empty — by design) and that `q` quits cleanly. Slice 4 still needs to register the OpenCode detector in `main.go` and capture real OpenCode pane fixtures.

---

### Slice 4 — OpenCode Detector

**Goal:** Multi-agent functionality complete.

**Starting state from Slice 3** (already wired in `cmd/overseer/main.go`):

- The TUI render pipeline is fully live. Whenever `agentStatusSvc.PollAll` resolves a `domain.AgentStatusKind` for a session, the TUI will render the matching glyph + tint in the session list and bump the count in the bottom-bar aggregate. Slice 4 does **not** need to touch any TUI code.
- `agentStatusRegistry` (a `*registry.Registry`) currently holds the Claude Code pane detector at `cmd/overseer/main.go:99`. Slice 4 adds **one line** alongside it: `agentStatusRegistry.Register(opencode.NewPaneDetector(tmuxAdapter))`. The service routes by `sess.AgentType`, which is already populated correctly (Slice 1's `LauncherConfig.agentType: "opencode"` flows through to `Session.AgentType` at creation time and on legacy-session inference).
- The Slice 2 `agent-status-debug` job is gone. Slice 3 replaced it with `buildAgentStatusJob` returning `shared.AgentStatusesUpdatedMsg{Statuses: resp.Statuses, Err: err}`. Slice 4 does not touch the job.
- The Claude Code detector at `internal/adapters/secondary/agentstatus/claudecode/` is the **structural template** for the OpenCode detector. Mirror its layout exactly: `detector.go` (the `PaneDetector` struct, `AgentType()`, `Detect()`, `classify()` and matchers), `patterns.go` (signal constants with rationale comments), `detector_test.go` (fixture-driven table tests across all 4 states + empty-pane Unknown + tmux-error propagation + port satisfaction check). The compile-time port check `var _ domain.AgentStatusDetector = (*PaneDetector)(nil)` belongs at the bottom of `detector.go`.
- Shared helpers exist and MUST be reused — do NOT re-implement: `internal/adapters/secondary/agentstatus/shared/ansi.go::StripANSI` strips ANSI escape codes (lipgloss-aware), `internal/adapters/secondary/agentstatus/shared/pane.go::LastNonEmptyLines(text, n)` returns the last `n` non-empty lines for the scan window. Claude Code uses `scanWindow = 30`; OpenCode should pick the same unless fixtures prove otherwise.
- Source tag convention: Slice 2's Claude Code detector tags its statuses with `Source: "claude-code/pane-parser"`. Slice 4's OpenCode detector MUST use `Source: "opencode/pane-parser"` so logs / debugging output remain disambiguable.
- Fixture-capture methodology is documented in commit `15a3639` and `internal/adapters/secondary/agentstatus/claudecode/testdata/`. To capture real OpenCode panes:
  1. Inside Overseer, create a session that launches OpenCode (the default OpenCode launcher works).
  2. Reach each target state through real interaction (idle prompt, mid-prompt run, approval modal, etc.).
  3. From outside Overseer: `tmux capture-pane -p -e -t <session-uuid>-agent: > internal/adapters/secondary/agentstatus/opencode/testdata/<state>_<n>.txt`. Commit the fixtures alongside the test.
- The pattern guesses in plan §7 (`"esc to interrupt"`, `"continue?"`, `"proceed?"`, `"enter to select"`, `"esc to cancel"`, `">"` / `"opencode>"`) are **starting hypotheses, not facts**. Confirm each one against a captured fixture; OpenCode's bottom status bar may differ from Claude's. Falsify and replace any signal that does not match the real captures.
- Opt-in live integration test mirrors Claude's: add `internal/adapters/secondary/agentstatus/opencode/detector_live_test.go` guarded by `//go:build live_tmux`. Same env-var protocol: `OVERSEER_LIVE_TMUX_ID=<session-uuid>-agent go test -tags=live_tmux ./internal/adapters/secondary/agentstatus/opencode/ -run TestPaneDetector_Detect_LiveTmux -v`. Useful as a sanity check while iterating on patterns.
- `cmd/overseer/main.go` already imports `claudecode` at line 17. Add an `opencode` import next to it. The variable name `agentStatusRegistry` is in scope at the registration site — do not rename it.

**Scope:**
- `internal/adapters/secondary/agentstatus/opencode/{detector.go,patterns.go,detector_test.go,detector_live_test.go}` (new)
- `internal/adapters/secondary/agentstatus/opencode/testdata/*.txt` (new — real captured fixtures)
- `cmd/overseer/main.go` — add `agentStatusRegistry.Register(opencode.NewPaneDetector(tmuxAdapter))` + `opencode` import

**Tests (TDD — RED first):**
- `TestPaneDetector_AgentType_ReturnsOpenCode` — port shape sanity check.
- `TestPaneDetector_Detect_RunningFixtures` — table test, one entry per captured running fixture.
- `TestPaneDetector_Detect_WaitingFixtures` — table test, one entry per captured waiting fixture.
- `TestPaneDetector_Detect_IdleFixtures` — table test, one entry per captured idle fixture.
- `TestPaneDetector_Detect_EmptyPane_ReturnsUnknown` — mocked tmux returns `""`.
- `TestPaneDetector_Detect_TmuxError_PropagatesAsError` — mocked tmux returns an error; assert `errors.Is(err, wantErr)`.
- `TestPaneDetector_Detect_SatisfiesDomainPort` — compile-time check.
- (opt-in, `//go:build live_tmux`) `TestPaneDetector_Detect_LiveTmux` — drives the real tmux adapter against a session pointed at by `OVERSEER_LIVE_TMUX_ID`.

**Acceptance criteria:**
- `go build ./...` clean
- `go test -race -count=1 ./...` GREEN across all packages (the new opencode package included)
- `lsp_diagnostics` clean on every changed file
- At least 2 fixtures per state (Running, Waiting, Idle) committed under `testdata/` and exercised by table tests
- Manual: launch one Claude Code session and one OpenCode session simultaneously inside Overseer. Confirm both rows render the correct status glyph + tint, and the status bar aggregate counts both. Verify with both `disableEmoji: true` and `false`.

**Status:** ✅ DONE — 2026-05-24 — full `go test -race -count=1 ./...` GREEN across all 23 testable packages (the new `opencode` package included); `go build ./...` exit 0; `lsp_diagnostics` clean on every changed file (the single warning on `detector_live_test.go` is the expected `live_tmux` build-tag notice — identical to Claude's). Real OpenCode 1.15.10 fixtures captured fresh under `internal/adapters/secondary/agentstatus/opencode/testdata/` — 2 per state (running_dots + running_thinking, waiting_bash_permission + waiting_edit_permission, idle_fresh + idle_after_response); each fixture's classification verified by the table tests. Pattern decisions falsified the plan §7 guesses: **Running** signal is `"esc interrupt"` (no "to" between, unlike Claude's `"esc to interrupt"`); **Waiting** signal is `"Permission required"` from the bash/edit/webfetch approval modal (no `"continue?"` / `"proceed?"` / `"enter to select"` text appeared in any real capture — those were starting hypotheses; the real OpenCode UI is the `△ Permission required` + `Allow once / Allow always / Reject` modal). Source tag `opencode/pane-parser` matches the plan's convention. Detector mirrors Claude's structural template exactly: `detector.go` (PaneDetector + AgentType + Detect + classify + matchWaiting/matchRunning), `patterns.go` (signal constants with rationale comments), `detector_test.go` (8 tests: AgentType, RunningFixtures table, WaitingFixtures table, IdleFixtures table, EmptyPane_ReturnsUnknown, TmuxError_PropagatesAsError, SatisfiesDomainPort), `detector_live_test.go` (`//go:build live_tmux`-gated opt-in test using the same `OVERSEER_LIVE_TMUX_ID=<uuid>-agent` env-var protocol). Shared helpers `shared.StripANSI` and `shared.LastNonEmptyLines` reused — NOT re-implemented. Live integration test passed against a known-idle OpenCode session: `kind=idle source=opencode/pane-parser reason=no running or waiting signals in last 30 lines`. Manual end-to-end verification: launched Overseer with isolated XDG_*  pointing at a fresh data dir + a test git repo, created one Claude Code session and one OpenCode session in the same project. Both rows rendered the correct status glyph (🟡 Waiting for Claude on the trust-folder prompt, ⚪ Idle for OpenCode at the empty input, then 🟢 Running on OpenCode after sending it a multi-paragraph prompt directly via tmux). Bottom-bar aggregate updated through the transitions: `🟡 1 · ⚪ 1` → `🟢 1 · 🟡 1`. Restarted Overseer with `disableEmoji: true` against the same data dir; same sessions rendered in no-emoji glyphs (`◐ claude-test` Waiting, `● opencode-test` Running) with the bottom-bar aggregate `● 1 · ◐ 1` — confirming both glyph branches of `NewGlyphs(disableEmoji bool)` work end-to-end. `cmd/overseer/main.go` swap is the minimum required — two-line change: new `opencode` import next to `claudecode`, and `agentStatusRegistry.Register(opencode.NewPaneDetector(tmuxAdapter))` added immediately after the Claude registration. The service routes by `sess.AgentType` (already populated by Slice 1's launcher config flow), so no other wiring needed.

---

### Slice 5 — Polish (Optional)

**Goal:** Nice-to-haves that don't block functionality.

**Candidates** (pick any subset based on time / value):
- **Adaptive polling tiers** (à la agent-of-empires): hot tier (Running/Waiting) every 5s, warm (Idle) every 15s, cold (Error/Dead) every 60s. Requires status-aware scheduler — non-trivial change.
- **Fresh-idle indicator**: track `idleEnteredAt` per session; sessions that just transitioned to Idle in the last 30s get a "breathing" subtle background instead of plain. Visually distinguishes "just finished" from "long-idle".
- **Detector confidence**: extend `AgentStatus` with a `Confidence` field (High/Medium/Low); render Low-confidence statuses with a `?` modifier glyph.
- **PollOne on focus**: when user navigates to a session in the list, immediately call `PollOne` for fresher data than the 5s tick.
- **Manual refresh keybinding**: `R` triggers `PollAll` outside the tick.

**Status:** ⬜ NOT STARTED

---

## 11. Gotchas & Important Notes

- **`renderSessionNode` is already tight on width math** — see lines 447-495. The new status glyph adds 2 characters (glyph + space) to the body. Verify the width-overflow fallback still works for narrow terminals.
- **Selection highlight precedence** — when a row is selected AND has a status tint, the selection must win visually. Achieve this by checking `focused` first in the renderer; status tint only applies when `!focused`.
- **Tmux pane capture is already polled every 500ms by the inspector** for the preview view. The new 5s status job is independent and won't interfere. Do **not** try to share the capture — they have different latency requirements.
- **`-e` flag preserves ANSI** in capture output. Detectors MUST strip ANSI before pattern matching. Do not skip `shared.StripANSI()`.
- **Schema migration in storage** — JSON file already supports omitempty, so adding `AgentType` doesn't break old files. The migration is only about *populating* the field for old sessions, not about being able to read them.
- **Per-detector timeout** prevents one hung agent from stalling the whole tick. Keep it at 2s; do not raise without good reason.
- **Bounded concurrency (8 goroutines)** in `PollAll` prevents tmux subprocess flooding on systems with many sessions. Tune later if needed.
- **Glyph rendering width** — `🟢` and `❔` are double-width in most terminal fonts; `●`, `◐`, `■`, `○`, `?` are single-width. Lipgloss handles width correctly, but the layout math in `renderSessionNode` reads widths via `lipgloss.Width(body)` so it'll Just Work. Don't hardcode +1.
- **Existing `Label` field is untouched** — it's user-assigned workflow state ("WIP", "done"), totally orthogonal to agent runtime status. Both coexist in the row.
- **`buildAgentStatusDebugJob` in `cmd/overseer/main.go` is Slice 2 scaffolding.** Slice 3 deletes it (along with the `log/slog` import if it becomes unused) and replaces it with the TUI-bound job. Do **not** carry both jobs — that would double the poll rate.
- **Audit every theme file when adding palette fields.** Adding `StatusRunningFg`/`StatusRunningBg`/etc. to `Theme` without populating every concrete theme (`theme_dark.go`, `theme_light.go`, `theme_dracula.go`, …) breaks the build. Run `git ls-files internal/adapters/primary/tui/styles/theme_*.go` to enumerate.
- **Reuse `shared.StripANSI` and `shared.LastNonEmptyLines` (Slice 2 helpers).** A new per-agent detector that re-implements ANSI stripping or last-N-line extraction is a smell — those helpers are package-shared on purpose. Tests live in `agentstatus/shared/` and any change there affects every detector.
- **Per-agent source tags.** Each detector tags its emitted `AgentStatus.Source` with `<agent-key>/pane-parser` (e.g. `claude-code/pane-parser`, `opencode/pane-parser`). Keep tags collision-free across agents — the service uses `service/...` tags for its own outputs (`service/tmux-liveness`, `service/registry`, `service/detector-error`) so log filtering stays meaningful.
- **Stale dashboard golden files in `internal/adapters/primary/tui/dashboard/testdata/`** (`TestDashboard_Default80x24.golden`, etc.) reference the pre-Slice-3 layout. They are **orphans — no test reads them** (`grep -rn "golden.RequireEqual" internal/` is empty). Leave them alone unless explicitly asked; touching them is out of scope for any agent-status slice.

---

## 12. Open Questions Deferred to Later

These were intentionally not decided in this plan. Address when/if needed:

- Persist last-known status across Overseer restarts? Currently transient.
- Per-agent custom polling intervals? Currently one global value.
- User-configurable status colors / glyphs? Currently theme-controlled, not per-user.
- Notifications on state transitions (e.g. desktop notification when a session enters `Waiting`)? Out of scope for v1.
- Status-based session sorting / filtering? Out of scope for v1.

---

## 13. Reference: Key Existing Files

| Concern | File |
|---|---|
| Session domain | `internal/core/domain/session.go` |
| Launcher domain | `internal/core/domain/launcher.go` |
| Tmux port | `internal/core/domain/tmux.go` |
| Tmux adapter (CapturePane, GetSession) | `internal/adapters/secondary/tmux/adapter.go` |
| Storage (JSON file) | `internal/adapters/secondary/storage/store.go` |
| Session service | `internal/core/service/session.go` |
| Config loader | `internal/shared/config/loader.go` |
| Shared messages | `internal/adapters/primary/tui/shared/messages.go` |
| Session list UI | `internal/adapters/primary/tui/session/model.go` |
| Create-session form | `internal/adapters/primary/tui/session/create_form.go` |
| Status bar | `internal/adapters/primary/tui/dashboard/status.go` |
| Dashboard root | `internal/adapters/primary/tui/dashboard/root.go` |
| Glyphs | `internal/adapters/primary/tui/styles/glyphs.go` |
| Themes | `internal/adapters/primary/tui/styles/theme*.go` |
| Styles | `internal/adapters/primary/tui/styles/styles.go` |
| Job scheduler | `internal/adapters/primary/tui/jobs/scheduler.go` |
| Reference job (PR status pattern to mirror) | `cmd/overseer/main.go` (search `prJob`) |
| Wiring entry point | `cmd/overseer/main.go` |
| Agent status domain types & ports (Slice 1) | `internal/core/domain/agent_status.go` |
| Agent status service (Slice 2) | `internal/core/service/agent_status.go` |
| Detector registry (Slice 2) | `internal/adapters/secondary/agentstatus/registry/registry.go` |
| Claude Code pane detector (Slice 2 — STRUCTURAL TEMPLATE for Slice 4) | `internal/adapters/secondary/agentstatus/claudecode/detector.go` |
| Claude Code patterns (template for opencode/patterns.go) | `internal/adapters/secondary/agentstatus/claudecode/patterns.go` |
| Claude Code fixture-driven tests (template for opencode/detector_test.go) | `internal/adapters/secondary/agentstatus/claudecode/detector_test.go` |
| Claude Code real-pane fixtures (template for opencode/testdata/) | `internal/adapters/secondary/agentstatus/claudecode/testdata/` |
| Shared detector helpers (Slice 2 — reuse, do not duplicate) | `internal/adapters/secondary/agentstatus/shared/{ansi.go,pane.go}` |
| Agent status TUI message (Slice 3) | `internal/adapters/primary/tui/shared/messages.go` (search `AgentStatusesUpdatedMsg`) |
| Status glyphs + theme palette (Slice 3) | `internal/adapters/primary/tui/styles/glyphs.go`, `theme*.go`, `styles.go` |
| Session-row status rendering (Slice 3) | `internal/adapters/primary/tui/session/model.go` (search `renderSessionNode`, `statusRowStyle`) |
| Status-bar aggregate model (Slice 3) | `internal/adapters/primary/tui/dashboard/status.go` |
| Dashboard wiring of status bar (Slice 3) | `internal/adapters/primary/tui/dashboard/root.go` (search `m.status`) |
| TUI-bound refresh job (Slice 3) | `cmd/overseer/main.go` (search `buildAgentStatusJob` / `agent-status-refresh`) |
| Opt-in live integration test (template) | `internal/adapters/secondary/agentstatus/claudecode/detector_live_test.go` |

---

## 14. References — Reference Projects Studied

For pattern ideas if you get stuck on a detector implementation:

- **agent-of-empires** (`njbrake/agent-of-empires`, Rust) — closest architectural cousin. Per-agent detector functions, ANSI stripping, last-N-lines window, adaptive polling tiers. Look at `src/tmux/status_detection.rs` and `src/agents.rs`.
- **rocha** (`dnlopes/rocha`, Go) — Claude Code hooks pattern. If/when we add the hook-based Claude detector in the future, mirror their `cmd/start_claude.go` hook-injection logic.
- **lazyagent** (`illegalstudio/lazyagent`, Go + Bubble Tea) — JSONL transcript parsing alternative. Not relevant to current pane-parsing approach but useful if we ever add a transcript-based detector.

---

**End of handoff document.** Ready for first slice.
