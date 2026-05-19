package session

import (
	"log/slog"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestModel_SessionsLoadedRendersProjectTree(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(nil))
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	beta := testutil.MakeSession("beta", overseerID)

	updated, cmd := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	if cmd == nil {
		t.Fatalf("Update() command = nil, want initial selection command")
	}
	msg, ok := cmd().(shared.SessionSelectedMsg)
	if !ok {
		t.Fatalf("initial selection msg type = %T, want shared.SessionSelectedMsg", cmd())
	}
	if msg.ID != alpha.ID.String() {
		t.Fatalf("initial SessionSelectedMsg.ID = %q, want %q", msg.ID, alpha.ID.String())
	}
	view := updated.(Model).View().Content
	for _, want := range []string{"▼ overseer", "alpha", "beta"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q: %q", want, view)
		}
	}
}

func TestModel_RawGroupingModeRendersSessionsWithoutVirtualRows(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	model := New(styles.New(), newSessionService(nil))
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer", otherID: "other"})
	model.groupingMode = sessionGroupingNone
	model.SetSize(80, 20)
	alpha := testutil.MakeSession("alpha", overseerID)
	beta := testutil.MakeSession("beta", otherID)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	view := updated.(Model).View().Content
	if strings.Contains(view, "▼ overseer") || strings.Contains(view, "▼ other") {
		t.Fatalf("View() rendered virtual group rows in raw mode: %q", view)
	}
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q: %q", want, view)
		}
	}
}

func TestModel_SelectionOnlyEmitsForSessionNodes(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(nil))
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})

	updated, cmd := updated.(Model).Update(keyPress("k"))
	if cmd != nil {
		t.Fatalf("Update(k) command = %#v, want nil at top boundary", cmd)
	}

	updated, cmd = updated.(Model).Update(keyPress("j"))
	if cmd == nil {
		t.Fatalf("Update(j) command = nil, want session selection")
	}
	msg, ok := cmd().(shared.SessionSelectedMsg)
	if !ok {
		t.Fatalf("selection msg type = %T, want shared.SessionSelectedMsg", cmd())
	}
	if msg.ID != alpha.ID.String() {
		t.Fatalf("SessionSelectedMsg.ID = %q, want %q", msg.ID, alpha.ID.String())
	}

}

func TestModel_LoadSessionsUsesRawSessions(t *testing.T) {
	alpha := testutil.MakeSession("alpha", uuid.New())
	repo := &mocks.MockSessionRepository{ListResult: []domain.Session{alpha}}
	model := New(styles.New(), newSessionService(repo))

	msg := model.loadSessions()().(shared.SessionsLoadedMsg)

	if msg.Err != nil {
		t.Fatalf("loadSessions() err = %v", msg.Err)
	}
	if len(msg.Sessions) != 1 || msg.Sessions[0].ID != alpha.ID {
		t.Fatalf("loadSessions() sessions = %+v, want raw session list", msg.Sessions)
	}
}

func TestModel_SessionsLoadedWithUnassignedProjectShowsNoProjectGroup(t *testing.T) {
	model := New(styles.New(), newSessionService(nil))
	model.SetSize(80, 20)
	model.SetFocus(true)
	orphan := testutil.MakeSession("orphan", uuid.Nil)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{orphan}})

	view := updated.(Model).View().Content
	if !strings.Contains(view, "(no project)") {
		t.Fatalf("View() missing '(no project)' label for unassigned session: %q", view)
	}
}

func TestModel_GroupRowRendersDifferentlyWhenCursorMovesToIt(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(nil))
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})
	viewSessionFocused := updated.(Model).View().Content

	updated, _ = updated.(Model).Update(keyPress("k"))
	viewGroupFocused := updated.(Model).View().Content

	if viewSessionFocused == viewGroupFocused {
		t.Fatalf("View() did not change when cursor moved from session to group: %q", viewGroupFocused)
	}
}

func newSessionService(repo *mocks.MockSessionRepository) service.SessionService {
	if repo == nil {
		repo = &mocks.MockSessionRepository{}
	}
	return *service.NewSessionService(repo, &mocks.MockTmuxAdapter{}, &mocks.MockGitAdapter{}, slog.Default())
}

func keyPress(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}
