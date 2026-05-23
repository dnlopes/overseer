// Package claudecode implements an AgentStatusDetector that classifies a
// Claude Code session by tmux-pane content. It does NOT decide whether the
// agent process is alive — that liveness check is the service layer's job
// (see internal/core/service/agent_status.go). The detector only resolves
// Running / Waiting / Idle / Unknown.
package claudecode

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dnlopes/overseer/internal/adapters/secondary/agentstatus/shared"
	"github.com/dnlopes/overseer/internal/core/domain"
)

const (
	sourceTag = "claude-code/pane-parser"

	scanWindow = 30
)

// PaneDetector classifies a Claude Code session's runtime state by
// pattern-matching the last few lines of its tmux pane. It is constructed
// once at startup and is safe for concurrent use because it holds no
// per-call state — every Detect call resolves through its tmux adapter.
type PaneDetector struct {
	tmux domain.TmuxAdapter
}

func NewPaneDetector(tmux domain.TmuxAdapter) *PaneDetector {
	return &PaneDetector{tmux: tmux}
}

func (d *PaneDetector) AgentType() domain.AgentType {
	return domain.AgentTypeClaudeCode
}

func (d *PaneDetector) Detect(ctx context.Context, sess domain.Session) (domain.AgentStatus, error) {
	agentTmuxID := sess.ID.String() + "-agent"
	raw, err := d.tmux.CapturePane(ctx, agentTmuxID)
	if err != nil {
		return domain.AgentStatus{}, fmt.Errorf("capture pane: %w", err)
	}
	clean := shared.StripANSI(raw)
	lines := shared.LastNonEmptyLines(clean, scanWindow)
	return classify(lines, time.Now()), nil
}

func classify(lines []string, now time.Time) domain.AgentStatus {
	if len(lines) == 0 {
		return domain.AgentStatus{
			Kind:       domain.AgentStatusUnknown,
			DetectedAt: now,
			Source:     sourceTag,
			Reason:     "pane is empty",
		}
	}

	if kind, reason := matchWaiting(lines); kind != "" {
		return domain.AgentStatus{
			Kind:       kind,
			DetectedAt: now,
			Source:     sourceTag,
			Reason:     reason,
		}
	}

	if reason, ok := matchRunning(lines); ok {
		return domain.AgentStatus{
			Kind:       domain.AgentStatusRunning,
			DetectedAt: now,
			Source:     sourceTag,
			Reason:     reason,
		}
	}

	return domain.AgentStatus{
		Kind:       domain.AgentStatusIdle,
		DetectedAt: now,
		Source:     sourceTag,
		Reason:     "no running or waiting signals in last " + strconv.Itoa(len(lines)) + " lines",
	}
}

func matchWaiting(lines []string) (domain.AgentStatusKind, string) {
	for _, l := range lines {
		if strings.Contains(l, signalWaitingProceed) {
			return domain.AgentStatusWaiting, "matched " + quote(signalWaitingProceed)
		}
	}
	hasEscCancel := false
	hasConfirm := false
	hasTabAmend := false
	for _, l := range lines {
		if strings.Contains(l, signalWaitingEscCancel) {
			hasEscCancel = true
		}
		if strings.Contains(l, signalWaitingConfirm) {
			hasConfirm = true
		}
		if strings.Contains(l, signalWaitingTabToAmend) {
			hasTabAmend = true
		}
	}
	switch {
	case hasEscCancel && hasTabAmend:
		return domain.AgentStatusWaiting, "matched " + quote(signalWaitingEscCancel) + " + " + quote(signalWaitingTabToAmend)
	case hasEscCancel && hasConfirm:
		return domain.AgentStatusWaiting, "matched " + quote(signalWaitingConfirm) + " + " + quote(signalWaitingEscCancel)
	}
	return "", ""
}

func matchRunning(lines []string) (string, bool) {
	for _, l := range lines {
		if strings.Contains(l, signalRunningInterrupt) {
			return "matched " + quote(signalRunningInterrupt), true
		}
	}
	return "", false
}

func quote(s string) string { return "\"" + s + "\"" }

var _ domain.AgentStatusDetector = (*PaneDetector)(nil)
