package domain

import (
	"context"
	"errors"
	"time"
)

// AgentType identifies which agent runs in a session. It is set on Session
// at creation time, copied from the chosen Launcher. Detection strategies
// route by AgentType so a session running Claude Code is parsed differently
// than one running OpenCode.
type AgentType string

const (
	AgentTypeUnknown    AgentType = "unknown"
	AgentTypeClaudeCode AgentType = "claude-code"
	AgentTypeOpenCode   AgentType = "opencode"
)

// AgentStatusKind enumerates the runtime states an agent can be in. The
// "Dead" status means the agent's tmux session is missing or crashed; the
// shell session is ignored for status purposes. "Unknown" is the safe
// fallback when the detector cannot classify the state — never a hard
// failure.
type AgentStatusKind string

const (
	AgentStatusUnknown AgentStatusKind = "unknown"
	AgentStatusRunning AgentStatusKind = "running"
	AgentStatusWaiting AgentStatusKind = "waiting"
	AgentStatusIdle    AgentStatusKind = "idle"
	AgentStatusDead    AgentStatusKind = "dead"
)

// AgentStatus is a value object describing what an agent is doing right
// now. DetectedAt records when the snapshot was taken; Source identifies
// the detector that produced it (e.g. "claude-code/pane-parser"); Reason
// is an optional human-readable hint (e.g. "matched 'esc to interrupt'").
type AgentStatus struct {
	Kind       AgentStatusKind
	DetectedAt time.Time
	Source     string
	Reason     string
}

// AgentStatusDetector is the port for per-(agent, strategy) detection
// implementations. Examples: ClaudeCodePaneDetector, ClaudeCodeHookDetector,
// OpenCodePaneDetector. Per-agent detectors only resolve Running / Waiting
// / Idle / Unknown; Dead detection lives in the service layer so detector
// authors never need to think about process liveness.
type AgentStatusDetector interface {
	AgentType() AgentType
	Detect(ctx context.Context, session Session) (AgentStatus, error)
}

// AgentStatusDetectorRegistry resolves an AgentType to its detector. Used
// by AgentStatusService.PollAll to route each session to the right
// detector implementation.
type AgentStatusDetectorRegistry interface {
	DetectorFor(agentType AgentType) (AgentStatusDetector, bool)
}

// AgentStatus sentinel errors.
var (
	ErrAgentTypeRequired = errors.New("agent type is required")
	ErrAgentTypeUnknown  = errors.New("agent type is not recognized")
)
