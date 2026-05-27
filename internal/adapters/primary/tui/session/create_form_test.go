package session

import (
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

func TestCreateForm_DefaultFocusIsName(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if form.currentField() != fieldName {
		t.Fatalf("default focus = %v, want fieldName", form.currentField())
	}
}

func TestCreateForm_DefaultsToCreateWorktreeOn(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if !form.createWorktree {
		t.Fatal("createWorktree default = false, want true")
	}
}

func TestCreateForm_TabCyclesThroughWorktreeFields(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	wantSequence := []formField{
		fieldRepository,
		fieldCreateWorktreeToggle,
		fieldBaseBranchPicker,
		fieldNewBranch,
		fieldLauncher,
		fieldEditor,
		fieldName,
	}
	for i, want := range wantSequence {
		updated, _ := tea.Model(form).Update(formKeyPress("tab"))
		form = updated.(CreateFormModel)
		if form.currentField() != want {
			t.Fatalf("tab %d: focus = %v, want %v", i+1, form.currentField(), want)
		}
	}
}

func TestCreateForm_TabSkipsWorktreeFieldsWhenToggleOff(t *testing.T) {
	form := newCreateFormForTest(t, nil)
	form.createWorktree = false
	form.rebuildFocusOrder()

	wantSequence := []formField{
		fieldRepository,
		fieldCreateWorktreeToggle,
		fieldLauncher,
		fieldEditor,
		fieldName,
	}
	for i, want := range wantSequence {
		updated, _ := tea.Model(form).Update(formKeyPress("tab"))
		form = updated.(CreateFormModel)
		if form.currentField() != want {
			t.Fatalf("tab %d (project-mode): focus = %v, want %v", i+1, form.currentField(), want)
		}
	}
}

func TestCreateForm_SpaceOnToggleSwitchesMode(t *testing.T) {
	form := newCreateFormForTest(t, nil)
	form.focusIdx = focusIdxOf(form.focusOrder, fieldCreateWorktreeToggle)
	form.updateFocusAndBlurs()

	updated, _ := tea.Model(form).Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	form = updated.(CreateFormModel)

	if form.createWorktree {
		t.Fatal("after space on toggle: createWorktree = true, want false")
	}
}

func TestCreateForm_SubmitProjectMode_SendsCreateWorktreeFalseAndNoBranch(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, _ := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), overseer.Path, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), overseer.Path, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, overseer.ID, nil, nil, testLaunchers(t), testEditors(t), 100)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	form = updated.(CreateFormModel)
	form.createWorktree = false
	form.rebuildFocusOrder()

	_, cmd := tea.Model(form).Update(formKeyPress("enter"))
	if cmd == nil {
		t.Fatalf("submit cmd = nil")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.HasWorktree() {
		t.Fatalf("project-mode session HasWorktree() = true, want false")
	}
	if msg.Session.Branch != "" {
		t.Fatalf("project-mode session Branch = %q, want empty", msg.Session.Branch)
	}
}

func TestCreateForm_SubmitWorktreeMode_PassesPickedBaseBranch(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, git := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, overseer.Path, "feat/foo", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

	projectsSvc, _ := newProjectsServiceWithMocks(t)
	branches := map[uuid.UUID][]domain.BranchInfo{
		overseer.ID: {
			{Name: "feat/foo", Scope: domain.BranchScopeLocal},
			{Name: "main", Scope: domain.BranchScopeLocal},
		},
	}
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, overseer.ID, branches, nil, testLaunchers(t), testEditors(t), 100)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	_, cmd := tea.Model(updated.(CreateFormModel)).Update(formKeyPress("enter"))
	if cmd == nil {
		t.Fatalf("submit cmd = nil")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if !msg.Session.HasWorktree() {
		t.Fatal("worktree-mode session HasWorktree() = false, want true")
	}
}

func TestCreateForm_SubmitWithoutNameShowsError(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	form := newCreateFormForTest(t, []domain.Project{overseer})

	_, cmd := tea.Model(form).Update(formKeyPress("enter"))

	if cmd != nil {
		t.Fatalf("submit with empty name returned cmd = %v, want nil", cmd)
	}
}

func TestCreateForm_SubmitWithoutRepoShowsError(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	_, cmd := updated.(CreateFormModel).Update(formKeyPress("enter"))

	if cmd != nil {
		t.Fatalf("submit with no repo returned cmd = %v, want nil", cmd)
	}
}

func TestCreateForm_BranchesLoadedMsg_UpdatesPickerForActiveProject(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	form := newCreateFormForTest(t, []domain.Project{overseer})

	branches := []domain.BranchInfo{
		{Name: "main", Scope: domain.BranchScopeLocal},
		{Name: "feat/foo", Scope: domain.BranchScopeLocal},
	}
	updated, _ := tea.Model(form).Update(shared.BranchesLoadedMsg{ProjectID: overseer.ID, Branches: branches})
	form = updated.(CreateFormModel)

	if len(form.baseBranchPicker.branches) != 2 {
		t.Fatalf("picker branches after load: %d, want 2", len(form.baseBranchPicker.branches))
	}
}

func TestCreateForm_ViewContainsToggleAndCoreLabels(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	view := form.View().Content
	for _, want := range []string{"Name", "Repository", "Create worktree?", "Launcher", "Editor"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q label: %q", want, view)
		}
	}
}

func TestCreateForm_ViewToggleOffHidesBaseBranchField(t *testing.T) {
	form := newCreateFormForTest(t, nil)
	form.createWorktree = false
	form.rebuildFocusOrder()

	view := form.View().Content
	if strings.Contains(view, "Base branch") {
		t.Fatalf("project-mode view should not show 'Base branch' field: %q", view)
	}
	if strings.Contains(view, "New branch") {
		t.Fatalf("project-mode view should not show 'New branch' field: %q", view)
	}
}

func TestCreateForm_ProjectRegisteredMsgAdoptsProject(t *testing.T) {
	form := newCreateFormForTest(t, nil)
	newProject := testutil.MakeProject("/repo/new", "new")

	updated, _ := tea.Model(form).Update(shared.ProjectRegisteredMsg{Project: newProject})
	form = updated.(CreateFormModel)

	proj := form.repoPicker.selectedProject()
	if proj == nil || proj.ID != newProject.ID {
		t.Fatalf("after register msg: picker selected = %+v, want %v", proj, newProject.ID)
	}
}

func newCreateFormForTest(t *testing.T, projects []domain.Project) CreateFormModel {
	t.Helper()
	svc, _, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	return NewCreateForm(styles.New(), svc, projectsSvc, projects, uuid.Nil, nil, nil, testLaunchers(t), testEditors(t), 100)
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
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
	defaultEditor, _ := domain.NewEditor("VSCode", "code")
	return *service.NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), defaultLauncher, defaultEditor, slog.Default()), repo, projects, tmux, git
}

func newProjectsServiceWithMocks(t *testing.T) (service.ProjectService, *mocks.MockProjectRepository) {
	t.Helper()
	repo := mocks.NewMockProjectRepository(t)
	git := mocks.NewMockGitAdapter(t)
	return *service.NewProjectService(repo, git, slog.Default()), repo
}

func testLaunchers(t *testing.T) []domain.Launcher {
	t.Helper()
	opencode, err := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
	if err != nil {
		t.Fatalf("NewLauncher OpenCode: %v", err)
	}
	claude, err := domain.NewLauncher("Claude Code", "claude", domain.AgentTypeClaudeCode)
	if err != nil {
		t.Fatalf("NewLauncher Claude Code: %v", err)
	}
	return []domain.Launcher{opencode, claude}
}

func testEditors(t *testing.T) []domain.Editor {
	t.Helper()
	vscode, err := domain.NewEditor("VSCode", "code")
	if err != nil {
		t.Fatalf("NewEditor VSCode: %v", err)
	}
	cursor, err := domain.NewEditor("Cursor", "cursor")
	if err != nil {
		t.Fatalf("NewEditor Cursor: %v", err)
	}
	return []domain.Editor{vscode, cursor}
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
	case " ":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	}
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}

func focusIdxOf(order []formField, target formField) int {
	for i, f := range order {
		if f == target {
			return i
		}
	}
	return 0
}
