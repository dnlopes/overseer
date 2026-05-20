package session

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestCreateForm_DefaultsToNoProject(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil, testLaunchers(t))

	if form.selectedProjectID() != uuid.Nil {
		t.Fatalf("default selected project = %v, want uuid.Nil", form.selectedProjectID())
	}
	if form.selectedProjectLabel() != "(none)" {
		t.Fatalf("default selected label = %q, want %q", form.selectedProjectLabel(), "(none)")
	}
}

func TestCreateForm_TabFocusesProjectSelectorAndArrowsCycle(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	widgets := testutil.MakeProject("/repo/widgets", "Widgets")
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer, widgets}, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("right"))

	got := updated.(CreateFormModel)
	if got.selectedProjectID() != overseer.ID {
		t.Fatalf("after right: selected = %v, want %v", got.selectedProjectID(), overseer.ID)
	}
	if got.selectedProjectLabel() != "Overseer" {
		t.Fatalf("after right: label = %q, want %q", got.selectedProjectLabel(), "Overseer")
	}
}

func TestCreateForm_LeftFromNoneWrapsToLastProject(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	widgets := testutil.MakeProject("/repo/widgets", "Widgets")
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer, widgets}, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("left"))

	got := updated.(CreateFormModel)
	if got.selectedProjectID() != widgets.ID {
		t.Fatalf("after left from (none): selected = %v, want %v", got.selectedProjectID(), widgets.ID)
	}
}

func TestCreateForm_SubmitCreatesSessionWithSelectedProject(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, git := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, overseer.Path, "main", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, []domain.Project{overseer}, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("alpha"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("right"))
	_, cmd := updated.(CreateFormModel).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit command = nil, want create command")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.Name != "alpha" {
		t.Fatalf("created session name = %q, want %q", msg.Session.Name, "alpha")
	}
	if msg.Session.ProjectID != overseer.ID {
		t.Fatalf("created session ProjectID = %v, want %v", msg.Session.ProjectID, overseer.ID)
	}
}

func TestCreateForm_SubmitWithNoneCreatesProjectlessSession(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), "/tmp/overseer-home", "").Return("tmux-orphan", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, nil, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("orphan"))
	_, cmd := updated.(CreateFormModel).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit command = nil, want create command")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.ProjectID != uuid.Nil {
		t.Fatalf("created session ProjectID = %v, want uuid.Nil", msg.Session.ProjectID)
	}
}

func TestCreateForm_ViewShowsCurrentProjectLabel(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer}, testLaunchers(t))

	view := form.View().Content
	if !strings.Contains(view, "(none)") {
		t.Fatalf("View() missing default '(none)' label: %q", view)
	}
}

func TestCreateForm_DefaultsToOpencodeLauncher(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil, testLaunchers(t))

	if form.resolvedAgentCommand() != "opencode" {
		t.Fatalf("default resolved agent command = %q, want %q", form.resolvedAgentCommand(), "opencode")
	}
}

func TestCreateForm_LauncherSelectorTogglesBetweenOpencodeAndClaude(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	got := updated.(CreateFormModel)
	if got.focusIndex.Value() != FieldLauncherSelectedIndex {
		t.Fatalf("after 2 tabs: focus = %d, want %d (launcher)", got.focusIndex.Value(), FieldLauncherSelectedIndex)
	}
	if got.resolvedAgentCommand() != "opencode" {
		t.Fatalf("initial launcher = %q, want %q", got.resolvedAgentCommand(), "opencode")
	}

	updated, _ = got.Update(formKeyPress("right"))
	got = updated.(CreateFormModel)
	if got.resolvedAgentCommand() != "claude" {
		t.Fatalf("after right: launcher = %q, want %q", got.resolvedAgentCommand(), "claude")
	}

	updated, _ = got.Update(formKeyPress("right"))
	got = updated.(CreateFormModel)
	if got.resolvedAgentCommand() != "opencode" {
		t.Fatalf("after 2 rights (wrap): launcher = %q, want %q", got.resolvedAgentCommand(), "opencode")
	}
}

func TestCreateForm_TabCyclesThreeFields(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	got := updated.(CreateFormModel)
	if got.focusIndex.Value() != FieldNameSelectedIndex {
		t.Fatalf("after 3 tabs (wrap): focus = %d, want %d (name)", got.focusIndex.Value(), FieldNameSelectedIndex)
	}
}

func TestCreateForm_SubmitWithDefaultOpencodeSendsOpencodeCommand(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), "/tmp/overseer-home", "").Return("tmux-orphan", nil).Once()
	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, nil, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("orphan"))
	_, cmd := updated.(CreateFormModel).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit command = nil")
	}
	cmd()
	if savedSession.AgentCommand != "opencode" {
		t.Fatalf("savedSession.AgentCommand = %q, want %q", savedSession.AgentCommand, "opencode")
	}
}

func TestCreateForm_SubmitWithClaudeSendsClaudeCommand(t *testing.T) {
	t.Setenv("HOME", "/tmp/overseer-home")
	svc, repo, _, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), "/tmp/overseer-home", "").Return("tmux-orphan", nil).Once()
	var savedSession domain.Session
	repo.EXPECT().Save(mock.Anything, mock.Anything).
		Run(func(_ context.Context, s domain.Session) { savedSession = s }).
		Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, nil, testLaunchers(t))

	updated, _ := form.Update(formKeyPress("orphan"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("right"))
	_, cmd := updated.(CreateFormModel).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit command = nil")
	}
	cmd()
	if savedSession.AgentCommand != "claude" {
		t.Fatalf("savedSession.AgentCommand = %q, want %q", savedSession.AgentCommand, "claude")
	}
}

func TestCreateForm_ViewShowsLauncherOptions(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil, testLaunchers(t))

	view := form.View().Content
	for _, want := range []string{"OpenCode", "Claude Code"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing launcher display name %q: %q", want, view)
		}
	}
}

func newCreateFormSessionService(t *testing.T) service.SessionService {
	t.Helper()
	svc, _, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	return svc
}

func newCreateFormSessionServiceWithMocks(t *testing.T) (service.SessionService, *mocks.MockSessionRepository, *mocks.MockProjectRepository, *mocks.MockTmuxAdapter, *mocks.MockGitAdapter) {
	t.Helper()
	repo := mocks.NewMockSessionRepository(t)
	projects := mocks.NewMockProjectRepository(t)
	tmux := mocks.NewMockTmuxAdapter(t)
	git := mocks.NewMockGitAdapter(t)
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode")
	return *service.NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), defaultLauncher, slog.Default()), repo, projects, tmux, git
}

func testLaunchers(t *testing.T) []domain.Launcher {
	t.Helper()
	opencode, err := domain.NewLauncher("OpenCode", "opencode")
	if err != nil {
		t.Fatalf("NewLauncher OpenCode: %v", err)
	}
	claude, err := domain.NewLauncher("Claude Code", "claude")
	if err != nil {
		t.Fatalf("NewLauncher Claude Code: %v", err)
	}
	return []domain.Launcher{opencode, claude}
}

func formKeyPress(value string) tea.KeyPressMsg {
	switch value {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	}
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}
