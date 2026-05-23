//go:build live_tmux

package claudecode

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/secondary/tmux"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// TestPaneDetector_Detect_LiveTmux is an integration test that runs the
// detector against a real tmux pane on the host. Opt-in via:
//
//	OVERSEER_LIVE_TMUX_ID=<session-uuid>-agent go test -tags=live_tmux \
//	    ./internal/adapters/secondary/agentstatus/claudecode/ \
//	    -run TestPaneDetector_Detect_LiveTmux -v
//
// The test prints the classified status. It does not assert a specific
// kind — pane content depends on what the agent is doing at the moment of
// capture — so a clean PASS plus printed output is the success signal.
func TestPaneDetector_Detect_LiveTmux(t *testing.T) {
	tmuxIDFromEnv := os.Getenv("OVERSEER_LIVE_TMUX_ID")
	if tmuxIDFromEnv == "" {
		t.Skip("set OVERSEER_LIVE_TMUX_ID=<uuid>-agent to run this test")
	}
	if !strings.HasSuffix(tmuxIDFromEnv, "-agent") {
		t.Fatalf("OVERSEER_LIVE_TMUX_ID must end in -agent: %q", tmuxIDFromEnv)
	}
	sessID, err := uuid.Parse(strings.TrimSuffix(tmuxIDFromEnv, "-agent"))
	if err != nil {
		t.Fatalf("OVERSEER_LIVE_TMUX_ID prefix must be a UUID: %v", err)
	}

	tmuxAdapter, err := tmux.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("tmux adapter: %v", err)
	}
	d := NewPaneDetector(tmuxAdapter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess := domain.Session{ID: sessID, Name: "live-verify", AgentType: domain.AgentTypeClaudeCode}
	status, err := d.Detect(ctx, sess)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	t.Logf("kind=%s source=%s reason=%s", status.Kind, status.Source, status.Reason)
}
