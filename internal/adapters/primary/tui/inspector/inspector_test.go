package inspector

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	return New(styles.New(), service.SessionService{}, 500*time.Millisecond)
}

func keyPress(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}

func TestInspector_StartsOnAgentView(t *testing.T) {
	m := newTestModel(t)
	if got := m.views[m.activeIx].Label(); got != "Agent" {
		t.Errorf("active view label = %q, want %q", got, "Agent")
	}
}

func TestInspector_NextKey_CyclesForward(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(keyPress("p"))
	m = updated.(Model)
	if got := m.views[m.activeIx].Label(); got != "Shell" {
		t.Errorf("after p, active view label = %q, want %q", got, "Shell")
	}
}

func TestInspector_NextKey_WrapsAround(t *testing.T) {
	m := newTestModel(t)
	for i := 0; i < 2; i++ {
		updated, _ := m.Update(keyPress("p"))
		m = updated.(Model)
	}
	if got := m.views[m.activeIx].Label(); got != "Agent" {
		t.Errorf("after 2x p, active view label = %q, want %q", got, "Agent")
	}
}

func TestInspector_PrevKey_CyclesBackward(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(keyPress("P"))
	m = updated.(Model)
	if got := m.views[m.activeIx].Label(); got != "Shell" {
		t.Errorf("after P, active view label = %q, want %q", got, "Shell")
	}
}

func TestInspector_SessionSelectedMsg_PropagatesToAllViews(t *testing.T) {
	m := newTestModel(t)
	id := uuid.New()
	updated, _ := m.Update(shared.SessionSelectedMsg{Session: domain.Session{ID: id}})
	m = updated.(Model)
	if m.sessionID != id {
		t.Errorf("model sessionID = %v, want %v", m.sessionID, id)
	}
	for i, v := range m.views {
		sv, ok := v.(*streamView)
		if !ok {
			t.Fatalf("view[%d] is %T, want *streamView", i, v)
		}
		if sv.sessionID != id {
			t.Errorf("view[%d] sessionID = %v, want %v", i, sv.sessionID, id)
		}
	}
}

func TestInspector_PreviewCapturedMsg_OnlyActiveViewProcesses(t *testing.T) {
	m := newTestModel(t)
	id := uuid.New()
	for i := range m.views {
		m.views[i].SetSession(id)
	}
	msg := previewCapturedMsg{
		kind:         viewKindAgent,
		sessionID:    id,
		content:      "agent stream",
		sessionReady: true,
	}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	agent := m.views[0].(*streamView)
	if agent.content != "agent stream" {
		t.Errorf("agent content = %q, want %q", agent.content, "agent stream")
	}

	// Switch to Shell and send an Agent-kind message; Shell should ignore it.
	updated, _ = m.Update(keyPress("p"))
	m = updated.(Model)
	staleMsg := previewCapturedMsg{
		kind:         viewKindAgent,
		sessionID:    id,
		content:      "stale agent",
		sessionReady: true,
	}
	updated, _ = m.Update(staleMsg)
	m = updated.(Model)

	shell := m.views[1].(*streamView)
	if shell.content != "" {
		t.Errorf("shell received agent-kind message: content = %q, want empty", shell.content)
	}
}

func TestInspector_SessionSelectionClearedMsg_ResetsAllViews(t *testing.T) {
	m := newTestModel(t)
	id := uuid.New()
	updated, _ := m.Update(shared.SessionSelectedMsg{Session: domain.Session{ID: id}})
	m = updated.(Model)
	for i := range m.views {
		sv := m.views[i].(*streamView)
		sv.content = "stale-content"
		sv.ready = true
	}

	updated, cmd := m.Update(shared.SessionSelectionClearedMsg{})
	m = updated.(Model)

	if cmd != nil {
		t.Errorf("Update(SessionSelectionClearedMsg) cmd = %#v, want nil (no further polling)", cmd)
	}
	if m.sessionID != uuid.Nil {
		t.Errorf("model sessionID = %v, want uuid.Nil", m.sessionID)
	}
	for i, v := range m.views {
		sv := v.(*streamView)
		if sv.sessionID != uuid.Nil {
			t.Errorf("view[%d] sessionID = %v, want uuid.Nil", i, sv.sessionID)
		}
		if sv.content != "" {
			t.Errorf("view[%d] content = %q, want empty after clear", i, sv.content)
		}
		if sv.ready {
			t.Errorf("view[%d] ready = true, want false after clear", i)
		}
	}
}
