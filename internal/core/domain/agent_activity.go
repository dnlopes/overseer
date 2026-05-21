package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ActivityKind classifies what an agent session is currently doing.
// Values are derived from session-store contents by AgentActivityProvider
// implementations (see lazyagent's two-step resolver for the reference algorithm).
type ActivityKind string

const (
	ActivityUnknown    ActivityKind = "unknown"
	ActivityIdle       ActivityKind = "idle"
	ActivityWaiting    ActivityKind = "waiting"
	ActivityThinking   ActivityKind = "thinking"
	ActivityReading    ActivityKind = "reading"
	ActivityWriting    ActivityKind = "writing"
	ActivityRunning    ActivityKind = "running"
	ActivitySearching  ActivityKind = "searching"
	ActivityBrowsing   ActivityKind = "browsing"
	ActivitySpawning   ActivityKind = "spawning"
	ActivityCompacting ActivityKind = "compacting"
)

// IsValid reports whether k is a known ActivityKind.
func (k ActivityKind) IsValid() bool {
	switch k {
	case ActivityUnknown, ActivityIdle, ActivityWaiting, ActivityThinking,
		ActivityReading, ActivityWriting, ActivityRunning, ActivitySearching,
		ActivityBrowsing, ActivitySpawning, ActivityCompacting:
		return true
	default:
		return false
	}
}

// IsActive reports whether the activity represents ongoing work
// (anything except idle, waiting, or unknown). Used by the TUI to
// decide whether to animate the spinner.
func (k ActivityKind) IsActive() bool {
	switch k {
	case ActivityIdle, ActivityWaiting, ActivityUnknown:
		return false
	default:
		return true
	}
}

// AgentSessionRef points at a specific agent process to observe.
// Carries the minimum information an AgentActivityProvider needs to
// (a) locate the agent process via the tmux session, and
// (b) constrain path resolution to the worktree the agent runs in.
type AgentSessionRef struct {
	// SessionID is the Overseer session identifier; echoed back on AgentActivity.
	SessionID uuid.UUID
	// AgentTmuxSessionID is the tmux session running the agent process
	// (typically "<session-id>-agent" per SessionService.AttachAgent).
	AgentTmuxSessionID string
	// WorktreePath is the agent's working directory; used as a hint when
	// disambiguating sessions that share a store but differ by cwd.
	WorktreePath string
}

// AgentActivity is an immutable snapshot of an agent session's current state.
// Construct via NewAgentActivity; callers obtain instances from
// AgentActivityProvider.Observe and never mutate the returned value.
type AgentActivity struct {
	SessionID  uuid.UUID
	Kind       ActivityKind
	Tool       string // optional; populated when Kind reflects a specific tool
	ObservedAt time.Time
}

// NewAgentActivity constructs an AgentActivity, validating that the session
// id is non-nil and the kind is one of the known ActivityKind values.
// Empty Tool is permitted (and expected for non-tool activities).
func NewAgentActivity(sessionID uuid.UUID, kind ActivityKind, tool string) (AgentActivity, error) {
	if sessionID == uuid.Nil {
		return AgentActivity{}, ErrAgentActivityInvalidSessionID
	}
	if !kind.IsValid() {
		return AgentActivity{}, ErrAgentActivityInvalidKind
	}
	return AgentActivity{
		SessionID:  sessionID,
		Kind:       kind,
		Tool:       tool,
		ObservedAt: time.Now(),
	}, nil
}

// AgentActivityProvider is the port that observes the current activity of an
// agent session. A single implementation owns BOTH discovery (which file/db
// the agent process is writing to — typically via lsof + pid-tree walk) AND
// parsing (reading that file and applying the two-step state resolver).
//
// Callers (the TUI scheduler) invoke Observe at a fixed interval and on
// fsnotify events; the implementation should be cheap to call repeatedly.
type AgentActivityProvider interface {
	Observe(ctx context.Context, ref AgentSessionRef) (AgentActivity, error)
}

// Domain-level sentinel errors. Callers use errors.Is to distinguish them.
var (
	// ErrAgentActivityInvalidSessionID is returned when constructing an
	// AgentActivity with a nil session id.
	ErrAgentActivityInvalidSessionID = errors.New("agent activity: session id cannot be nil")

	// ErrAgentActivityInvalidKind is returned when constructing an
	// AgentActivity with an unrecognized ActivityKind.
	ErrAgentActivityInvalidKind = errors.New("agent activity: unknown activity kind")

	// ErrAgentNotRunning is returned by AgentActivityProvider.Observe when the
	// referenced tmux session has no agent process (e.g. the user has not
	// attached, or the agent has exited). Callers should report this as
	// ActivityUnknown / ActivityIdle rather than as a hard failure.
	ErrAgentNotRunning = errors.New("agent activity: agent process not running")

	// ErrAgentStoreNotResolved is returned when the provider cannot determine
	// which session-store file the agent is writing to (e.g. agent launched
	// but has not yet opened its JSONL/SQLite store). Callers should retry.
	ErrAgentStoreNotResolved = errors.New("agent activity: session store not yet resolved")
)
