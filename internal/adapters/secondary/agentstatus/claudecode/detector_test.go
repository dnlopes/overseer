package claudecode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
	"github.com/stretchr/testify/mock"
)

func TestPaneDetector_AgentType_ReturnsClaudeCode(t *testing.T) {
	d := NewPaneDetector(nil)
	if got := d.AgentType(); got != domain.AgentTypeClaudeCode {
		t.Fatalf("AgentType() = %q, want %q", got, domain.AgentTypeClaudeCode)
	}
}

func TestPaneDetector_Detect_RunningFixtures(t *testing.T) {
	cases := []string{
		"running_beaming.txt",
		"running_doing_with_shell.txt",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			status := detectFromFixture(t, name)
			if status.Kind != domain.AgentStatusRunning {
				t.Fatalf("Detect(%s) = %q, want %q (reason=%q)", name, status.Kind, domain.AgentStatusRunning, status.Reason)
			}
			if status.Source != "claude-code/pane-parser" {
				t.Fatalf("Detect(%s).Source = %q, want %q", name, status.Source, "claude-code/pane-parser")
			}
			if status.Reason == "" {
				t.Fatalf("Detect(%s).Reason is empty, want a signal explanation", name)
			}
		})
	}
}

func TestPaneDetector_Detect_WaitingFixtures(t *testing.T) {
	cases := []string{
		"waiting_bash_approval.txt",
		"waiting_trust_folder.txt",
		"waiting_agents_menu.txt",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			status := detectFromFixture(t, name)
			if status.Kind != domain.AgentStatusWaiting {
				t.Fatalf("Detect(%s) = %q, want %q (reason=%q)", name, status.Kind, domain.AgentStatusWaiting, status.Reason)
			}
			if status.Source != "claude-code/pane-parser" {
				t.Fatalf("Detect(%s).Source = %q, want %q", name, status.Source, "claude-code/pane-parser")
			}
			if status.Reason == "" {
				t.Fatalf("Detect(%s).Reason is empty, want a signal explanation", name)
			}
		})
	}
}

func TestPaneDetector_Detect_IdleFixtures(t *testing.T) {
	cases := []string{
		"idle_fresh.txt",
		"idle_after_response.txt",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			status := detectFromFixture(t, name)
			if status.Kind != domain.AgentStatusIdle {
				t.Fatalf("Detect(%s) = %q, want %q (reason=%q)", name, status.Kind, domain.AgentStatusIdle, status.Reason)
			}
			if status.Source != "claude-code/pane-parser" {
				t.Fatalf("Detect(%s).Source = %q, want %q", name, status.Source, "claude-code/pane-parser")
			}
		})
	}
}

func TestPaneDetector_Detect_EmptyPane_ReturnsUnknown(t *testing.T) {
	d, tmux, sess := newDetectorWithStub(t)
	tmux.EXPECT().CapturePane(mock.Anything, sess.ID.String()+"-agent").Return("", nil).Once()

	got, err := d.Detect(context.Background(), sess)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if got.Kind != domain.AgentStatusUnknown {
		t.Fatalf("Detect(empty) = %q, want %q", got.Kind, domain.AgentStatusUnknown)
	}
}

func TestPaneDetector_Detect_TmuxError_PropagatesAsError(t *testing.T) {
	d, tmux, sess := newDetectorWithStub(t)
	wantErr := errors.New("tmux exploded")
	tmux.EXPECT().CapturePane(mock.Anything, sess.ID.String()+"-agent").Return("", wantErr).Once()

	_, err := d.Detect(context.Background(), sess)
	if err == nil {
		t.Fatal("Detect: err = nil, want wrapped tmux error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Detect: err = %v, want chain to include %v", err, wantErr)
	}
}

func TestPaneDetector_Detect_SatisfiesDomainPort(t *testing.T) {
	var _ domain.AgentStatusDetector = (*PaneDetector)(nil)
}

func detectFromFixture(t *testing.T, name string) domain.AgentStatus {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	d, tmux, sess := newDetectorWithStub(t)
	tmux.EXPECT().CapturePane(mock.Anything, sess.ID.String()+"-agent").Return(string(data), nil).Once()

	got, err := d.Detect(context.Background(), sess)
	if err != nil {
		t.Fatalf("Detect(%s) error = %v", name, err)
	}
	if got.DetectedAt.IsZero() {
		t.Fatalf("Detect(%s).DetectedAt is zero", name)
	}
	return got
}

func newDetectorWithStub(t *testing.T) (*PaneDetector, *mocks.MockTmuxAdapter, domain.Session) {
	t.Helper()
	tmux := mocks.NewMockTmuxAdapter(t)
	d := NewPaneDetector(tmux)
	sess := domain.Session{ID: uuid.New(), AgentType: domain.AgentTypeClaudeCode}
	return d, tmux, sess
}
