package session

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/shared/paths"
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestModel_SessionsLoadedRendersProjectTree(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
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
	if msg.Session.ID != alpha.ID {
		t.Fatalf("initial SessionSelectedMsg.Session.ID = %v, want %v", msg.Session.ID, alpha.ID)
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
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
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

func TestModel_SelectionEmitsClearedWhenCursorLandsOnGroup(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})

	updated, cmd := updated.(Model).Update(keyPress("k"))
	if cmd == nil {
		t.Fatalf("Update(k) command = nil, want SessionSelectionClearedMsg when cursor lands on group")
	}
	if _, ok := cmd().(shared.SessionSelectionClearedMsg); !ok {
		t.Fatalf("selection msg type = %T, want shared.SessionSelectionClearedMsg", cmd())
	}

	updated, cmd = updated.(Model).Update(keyPress("j"))
	if cmd == nil {
		t.Fatalf("Update(j) command = nil, want session selection")
	}
	msg, ok := cmd().(shared.SessionSelectedMsg)
	if !ok {
		t.Fatalf("selection msg type = %T, want shared.SessionSelectedMsg", cmd())
	}
	if msg.Session.ID != alpha.ID {
		t.Fatalf("SessionSelectedMsg.Session.ID = %v, want %v", msg.Session.ID, alpha.ID)
	}
}

func TestModel_LoadSessionsUsesRawSessions(t *testing.T) {
	alpha := testutil.MakeSession("alpha", uuid.New())
	svc, repo := newSessionServiceWithRepo(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{alpha}, nil).Once()
	model := New(styles.New(), svc, domain.DefaultLabels)

	msg := model.loadSessions()().(shared.SessionsLoadedMsg)

	if msg.Err != nil {
		t.Fatalf("loadSessions() err = %v", msg.Err)
	}
	if len(msg.Sessions) != 1 || msg.Sessions[0].ID != alpha.ID {
		t.Fatalf("loadSessions() sessions = %+v, want raw session list", msg.Sessions)
	}
}

func TestModel_GroupRowRendersDifferentlyWhenCursorMovesToIt(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
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

func TestModel_RenderHasNoNumberPrefix(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})

	view := updated.(Model).View().Content
	if strings.Contains(view, "01.") || strings.Contains(view, "02.") {
		t.Fatalf("View() should not contain number prefixes: %q", view)
	}
}

func TestModel_NextGroupJumpsToNextProject(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer", otherID: "other"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	beta := testutil.MakeSession("beta", otherID)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})
	updated = navigateToTop(t, updated)

	updated, _ = updated.(Model).Update(keyPress("g"))

	got := updated.(Model).tree.SelectedID()
	if got != "project:"+overseerID.String() {
		t.Fatalf("SelectedID after 'g' = %q, want overseer group id", got)
	}
}

func TestModel_PrevGroupJumpsToPreviousProject(t *testing.T) {
	overseerID := uuid.New()
	otherID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer", otherID: "other"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	beta := testutil.MakeSession("beta", otherID)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	updated, _ = updated.(Model).Update(keyPress("G"))

	got := updated.(Model).tree.SelectedID()
	if got != "project:"+overseerID.String() {
		t.Fatalf("SelectedID after 'G' from alpha = %q, want overseer group id", got)
	}
}

func TestModel_CtrlDownMovesCursorFiveRows(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	sessions := make([]domain.Session, 0, 10)
	for i := 0; i < 10; i++ {
		sessions = append(sessions, testutil.MakeSession("session-"+string(rune('a'+i)), overseerID))
	}
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: sessions})

	updated, cmd := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl})

	if cmd == nil {
		t.Fatalf("Update(ctrl+down) cmd = nil, want selection emit")
	}
	msg, ok := cmd().(shared.SessionSelectedMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want shared.SessionSelectedMsg", cmd())
	}
	wantID := sessions[5].ID
	if msg.Session.ID != wantID {
		t.Fatalf("SessionSelectedMsg.Session.ID = %v, want %v (initial session-a + 5 rows = session-f)", msg.Session.ID, wantID)
	}
}

func TestModel_CtrlUpMovesCursorFiveRowsClampedAtTop(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	beta := testutil.MakeSession("beta", overseerID)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	updated, _ = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})

	got := updated.(Model).tree.SelectedID()
	if !strings.HasPrefix(got, "project:") {
		t.Fatalf("SelectedID after ctrl+up from top = %q, want clamped at group row", got)
	}
}

func TestModel_ShiftDownReordersSelectedSession(t *testing.T) {
	overseerID := uuid.New()
	alpha := testutil.MakeSession("alpha", overseerID)
	alpha.Order = 1
	beta := testutil.MakeSession("beta", overseerID)
	beta.Order = 2
	svc, repo := newSessionServiceWithRepo(t)
	repo.EXPECT().Get(mock.Anything, alpha.ID).Return(alpha, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{alpha, beta}, nil).Twice()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Twice()
	model := New(styles.New(), svc, domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	_, cmd := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})

	if cmd == nil {
		t.Fatalf("Update(shift+down) cmd = nil, want reorder command")
	}
	msg, ok := cmd().(shared.SessionReorderedMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want shared.SessionReorderedMsg", cmd())
	}
	if msg.Err != nil {
		t.Fatalf("SessionReorderedMsg.Err = %v, want nil", msg.Err)
	}
	if msg.FocusID != alpha.ID.String() {
		t.Fatalf("SessionReorderedMsg.FocusID = %q, want %q (the moved session)", msg.FocusID, alpha.ID.String())
	}
}

func TestModel_ShiftDownNoOpOnGroupRow(t *testing.T) {
	overseerID := uuid.New()
	alpha := testutil.MakeSession("alpha", overseerID)
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})
	updated, _ = updated.(Model).Update(keyPress("k"))

	_, cmd := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})

	if cmd != nil {
		t.Fatalf("Update(shift+down) on group row cmd = %#v, want nil", cmd)
	}
}

func TestModel_SessionReorderedMsgRestoresCursorOnMovedSession(t *testing.T) {
	overseerID := uuid.New()
	alpha := testutil.MakeSession("alpha", overseerID)
	alpha.Order = 1
	beta := testutil.MakeSession("beta", overseerID)
	beta.Order = 2
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	updated, _ = updated.(Model).Update(shared.SessionReorderedMsg{
		Sessions: []domain.Session{beta, alpha},
		FocusID:  beta.ID.String(),
	})

	got := updated.(Model).tree.SelectedID()
	want := "session:" + beta.ID.String()
	if got != want {
		t.Fatalf("SelectedID after reorder = %q, want %q (focused on moved session)", got, want)
	}
}

func navigateToTop(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	for i := 0; i < 50; i++ {
		next, _ := m.(Model).Update(keyPress("k"))
		if next.(Model).tree.SelectedID() == m.(Model).tree.SelectedID() {
			return next
		}
		m = next
	}
	t.Fatalf("did not reach top after 50 'k' presses")
	return m
}

func TestModel_DKeyEmitsDeleteRequestedForSelectedSession(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})
	_, cmd := updated.(Model).Update(keyPress("d"))

	if cmd == nil {
		t.Fatalf("Update(d) command = nil, want delete-requested emit")
	}
	msg, ok := cmd().(shared.SessionDeleteRequestedMsg)
	if !ok {
		t.Fatalf("Update(d) msg type = %T, want shared.SessionDeleteRequestedMsg", cmd())
	}
	if msg.Session.ID != alpha.ID {
		t.Fatalf("SessionDeleteRequestedMsg.Session.ID = %v, want %v", msg.Session.ID, alpha.ID)
	}
	if msg.Session.Name != "alpha" {
		t.Fatalf("SessionDeleteRequestedMsg.Session.Name = %q, want %q", msg.Session.Name, "alpha")
	}
}

func TestModel_DKeyWithGroupNodeSelected_NoOp(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})
	updated, _ = updated.(Model).Update(keyPress("k"))
	_, cmd := updated.(Model).Update(keyPress("d"))

	if cmd != nil {
		t.Fatalf("Update(d) command = %#v, want nil when group node is selected", cmd)
	}
}

func TestModel_SessionDeletedMsg_TriggersReload(t *testing.T) {
	alpha := testutil.MakeSession("alpha", uuid.New())
	svc, repo := newSessionServiceWithRepo(t)
	repo.EXPECT().List(mock.Anything).Return([]domain.Session{alpha}, nil).Once()
	model := New(styles.New(), svc, domain.DefaultLabels)

	_, cmd := model.Update(shared.SessionDeletedMsg{})

	if cmd == nil {
		t.Fatalf("Update(SessionDeletedMsg) command = nil, want reload command")
	}
	msg, ok := cmd().(shared.SessionsLoadedMsg)
	if !ok {
		t.Fatalf("Reload msg type = %T, want shared.SessionsLoadedMsg", cmd())
	}
	if msg.Err != nil {
		t.Fatalf("Reload SessionsLoadedMsg.Err = %v, want nil", msg.Err)
	}
}

func newSessionService(t *testing.T) service.SessionService {
	t.Helper()
	svc, _ := newSessionServiceWithRepo(t)
	return svc
}

func newSessionServiceWithRepo(t *testing.T) (service.SessionService, *mocks.MockSessionRepository) {
	t.Helper()
	repo := mocks.NewMockSessionRepository(t)
	projects := mocks.NewMockProjectRepository(t)
	tmux := mocks.NewMockTmuxAdapter(t)
	git := mocks.NewMockGitAdapter(t)
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode")
	defaultEditor, _ := domain.NewEditor("VSCode", "code")
	return *service.NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), defaultLauncher, defaultEditor, slog.Default()), repo
}

func keyPress(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}

func TestModel_View_RendersUpdatedAtDurationRightAlignedOnEachSessionRow(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	alpha.UpdatedAt = time.Now().Add(-3 * time.Hour)
	beta := testutil.MakeSession("beta", overseerID)
	beta.UpdatedAt = time.Now().Add(-2 * 24 * time.Hour)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha, beta}})

	view := ansi.Strip(updated.(Model).View().Content)
	alphaRow := findRow(t, view, "alpha")
	betaRow := findRow(t, view, "beta")

	if !strings.Contains(alphaRow, "3h") {
		t.Errorf("alpha row missing '3h' duration: %q", alphaRow)
	}
	if !strings.Contains(betaRow, "2d") {
		t.Errorf("beta row missing '2d' duration: %q", betaRow)
	}

	if !rowSuffixIs(alphaRow, "3h") {
		t.Errorf("alpha row does not end with '3h' (not right-aligned): %q", alphaRow)
	}
	if !rowSuffixIs(betaRow, "2d") {
		t.Errorf("beta row does not end with '2d' (not right-aligned): %q", betaRow)
	}
}

func TestModel_View_DoesNotRenderDurationOnGroupRows(t *testing.T) {
	overseerID := uuid.New()
	model := New(styles.New(), newSessionService(t), domain.DefaultLabels)
	model.SetProjectNames(map[uuid.UUID]string{overseerID: "overseer"})
	model.SetSize(80, 20)
	model.SetFocus(true)
	alpha := testutil.MakeSession("alpha", overseerID)
	alpha.UpdatedAt = time.Now().Add(-3 * time.Hour)

	updated, _ := model.Update(shared.SessionsLoadedMsg{Sessions: []domain.Session{alpha}})

	view := ansi.Strip(updated.(Model).View().Content)
	groupRow := findRow(t, view, "overseer")

	if strings.Contains(groupRow, "3h") {
		t.Errorf("group row should not carry a per-session duration label, got: %q", groupRow)
	}
}

func findRow(t *testing.T, view, marker string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, marker) {
			return line
		}
	}
	t.Fatalf("no row contains %q in view:\n%s", marker, view)
	return ""
}

func rowSuffixIs(row, suffix string) bool {
	trimmed := strings.TrimRight(row, " │")
	return strings.HasSuffix(trimmed, suffix)
}
