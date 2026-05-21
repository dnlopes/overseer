package session

import (
	"log/slog"
	"strings"
	"testing"
	"time"

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

	if form.focusIndex.Value() != fieldName {
		t.Fatalf("default focus = %d, want %d (name)", form.focusIndex.Value(), fieldName)
	}
}

func TestCreateForm_TabCyclesSixFields(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	for i, want := range []int{fieldRepository, fieldBaseBranch, fieldFeatureBranch, fieldLauncher, fieldEditor, fieldName} {
		updated, _ := tea.Model(form).Update(formKeyPress("tab"))
		form = updated.(CreateFormModel)
		if form.focusIndex.Value() != want {
			t.Fatalf("tab %d: focus = %d, want %d", i+1, form.focusIndex.Value(), want)
		}
	}
}

func TestCreateForm_RepoPickerCyclesProjects(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	widgets := testutil.MakeProject("/repo/widgets", "Widgets")
	form := newCreateFormForTest(t, []domain.Project{overseer, widgets})

	updated, _ := tea.Model(form).Update(formKeyPress("tab"))
	form = updated.(CreateFormModel)
	initial := form.repoPicker.selectedProject()
	if initial == nil {
		t.Fatalf("initial picker selected = nil, want first project")
	}

	updated, _ = tea.Model(form).Update(formKeyPress("right"))
	form = updated.(CreateFormModel)
	after := form.repoPicker.selectedProject()
	if after == nil || after.ID == initial.ID {
		t.Fatalf("after right: selected did not advance from %v to a different project (got %v)", initial.ID, after)
	}
}

func TestCreateForm_RepoPickerPreselectsInitialProjectID(t *testing.T) {
	older := testutil.MakeProject("/repo/older", "Older")
	newer := testutil.MakeProject("/repo/newer", "Newer")
	older.UpdatedAt = time.Now().Add(-time.Hour)
	newer.UpdatedAt = time.Now()

	svc, _, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{older, newer}, older.ID, testLaunchers(t), testEditors(t), 100)

	selected := form.repoPicker.selectedProject()
	if selected == nil {
		t.Fatalf("selected = nil, want %s", older.ID)
	}
	if selected.ID != older.ID {
		t.Fatalf("selected = %s (%s), want %s (older) — newer should NOT win when older is explicitly initial", selected.ID, selected.Name, older.ID)
	}
}

func TestCreateForm_RepoPickerFallsBackWhenInitialIDUnknown(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, _, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, uuid.New(), testLaunchers(t), testEditors(t), 100)

	selected := form.repoPicker.selectedProject()
	if selected == nil || selected.ID != overseer.ID {
		t.Fatalf("selected = %+v, want fallback to %s", selected, overseer.ID)
	}
}

func TestCreateForm_RepoPickerSentinelEntersPasteMode(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	form := newCreateFormForTest(t, []domain.Project{overseer})

	updated, _ := tea.Model(form).Update(formKeyPress("tab"))
	form = updated.(CreateFormModel)
	updated, _ = tea.Model(form).Update(formKeyPress("right"))
	form = updated.(CreateFormModel)
	if !form.repoPicker.onSentinel() {
		t.Fatalf("after cycling past project: picker not on sentinel, listIdx=%d", form.repoPicker.listIdx)
	}

	updated, _ = tea.Model(form).Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	form = updated.(CreateFormModel)
	if !form.repoPicker.isPasteMode() {
		t.Fatalf("after ctrl+p on sentinel: picker not in paste mode")
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

func TestCreateForm_SubmitWithExistingProjectCallsServiceCreate(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, git := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, overseer.Path, "develop", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, uuid.Nil, testLaunchers(t), testEditors(t), 100)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	form = updated.(CreateFormModel)
	form.baseBranchInput.SetValue("develop")
	_, cmd := tea.Model(form).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit cmd = nil")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.ProjectID != overseer.ID {
		t.Fatalf("created session ProjectID = %v, want %v", msg.Session.ProjectID, overseer.ID)
	}
	if msg.Session.BaseBranch != "develop" {
		t.Fatalf("created session BaseBranch = %q, want %q", msg.Session.BaseBranch, "develop")
	}
	if msg.Session.EditorCommand != "code" {
		t.Fatalf("created session EditorCommand = %q, want %q (default VSCode)", msg.Session.EditorCommand, "code")
	}
}

func TestCreateForm_FeatureBranchDefaultsToEmpty(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if got := form.featureBranchInput.Value(); got != "" {
		t.Fatalf("default feature branch input = %q, want empty (auto-generate signal)", got)
	}
}

func TestCreateForm_BaseBranchDefaultsToEmpty(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if got := form.baseBranchInput.Value(); got != "" {
		t.Fatalf("default base branch input = %q, want empty (repo-default signal)", got)
	}
}

func TestCreateForm_SubmitWithEmptyBaseBranchResolvesRepoDefault(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, git := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().GetDefaultBranch(mock.Anything, overseer.Path).Return("trunk", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, overseer.Path, "trunk", mock.Anything, mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, uuid.Nil, testLaunchers(t), testEditors(t), 100)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	_, cmd := tea.Model(updated.(CreateFormModel)).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit cmd = nil")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.BaseBranch != "trunk" {
		t.Fatalf("created session BaseBranch = %q, want resolved default %q", msg.Session.BaseBranch, "trunk")
	}
}

func TestCreateForm_SubmitForwardsUserProvidedFeatureBranch(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, projects, tmux, git := newCreateFormSessionServiceWithMocks(t)
	projects.EXPECT().Get(mock.Anything, overseer.ID).Return(overseer, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, overseer.Path, "develop", "my-feature", mock.Anything).Return(nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.UUIDString(), mock.Anything, "").Return("tmux-alpha", nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, testutil.AgentTmuxIDString(), mock.Anything, "opencode").Return("tmux-alpha-agent", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projects.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	form := NewCreateForm(styles.New(), svc, projectsSvc, []domain.Project{overseer}, uuid.Nil, testLaunchers(t), testEditors(t), 100)

	updated, _ := tea.Model(form).Update(formKeyPress("alpha"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	form = updated.(CreateFormModel)
	form.baseBranchInput.SetValue("develop")
	form.featureBranchInput.SetValue("my-feature")
	_, cmd := tea.Model(form).Update(formKeyPress("enter"))

	if cmd == nil {
		t.Fatalf("submit cmd = nil")
	}
	msg, ok := cmd().(shared.SessionCreatedMsg)
	if !ok {
		t.Fatalf("submit msg type = %T, want shared.SessionCreatedMsg", cmd())
	}
	if msg.Session.FeatureBranch != "my-feature" {
		t.Fatalf("created session FeatureBranch = %q, want %q", msg.Session.FeatureBranch, "my-feature")
	}
}

func TestCreateForm_ViewShowsAllFieldLabels(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	view := form.View().Content
	for _, want := range []string{"Name", "Repository", "Base branch", "Feature branch", "Launcher", "Editor"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q label: %q", want, view)
		}
	}
}

func TestCreateForm_ViewShowsSelectedLauncher(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	view := form.View().Content
	if !strings.Contains(view, "OpenCode") {
		t.Fatalf("View() missing selected launcher %q: %q", "OpenCode", view)
	}
	if strings.Contains(view, "Claude Code") {
		t.Fatalf("View() unexpectedly shows non-selected launcher %q (picker should show only current): %q", "Claude Code", view)
	}
}

func TestCreateForm_ViewShowsSelectedEditor(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	view := form.View().Content
	if !strings.Contains(view, "VSCode") {
		t.Fatalf("View() missing selected editor %q: %q", "VSCode", view)
	}
	if strings.Contains(view, "Cursor") {
		t.Fatalf("View() unexpectedly shows non-selected editor %q (picker should show only current): %q", "Cursor", view)
	}
}

func TestCreateForm_DefaultsToOpencodeLauncher(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if form.resolvedAgentCommand() != "opencode" {
		t.Fatalf("default agent command = %q, want %q", form.resolvedAgentCommand(), "opencode")
	}
}

func TestCreateForm_DefaultsToVSCodeEditor(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	if form.resolvedEditorCommand() != "code" {
		t.Fatalf("default editor command = %q, want %q", form.resolvedEditorCommand(), "code")
	}
}

func TestCreateForm_EditorSelectorTogglesBetweenEntries(t *testing.T) {
	form := newCreateFormForTest(t, nil)

	// Tab past Name, Repository, Base branch, Feature branch, Launcher to reach Editor.
	updated, _ := tea.Model(form).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("tab"))
	got := updated.(CreateFormModel)
	if got.focusIndex.Value() != fieldEditor {
		t.Fatalf("after 5 tabs: focus = %d, want %d (editor)", got.focusIndex.Value(), fieldEditor)
	}
	if got.resolvedEditorCommand() != "code" {
		t.Fatalf("initial editor = %q, want %q", got.resolvedEditorCommand(), "code")
	}

	updated, _ = tea.Model(got).Update(formKeyPress("right"))
	got = updated.(CreateFormModel)
	if got.resolvedEditorCommand() != "cursor" {
		t.Fatalf("after right: editor = %q, want %q", got.resolvedEditorCommand(), "cursor")
	}

	updated, _ = tea.Model(got).Update(formKeyPress("right"))
	got = updated.(CreateFormModel)
	if got.resolvedEditorCommand() != "code" {
		t.Fatalf("after 2 rights (wrap): editor = %q, want %q", got.resolvedEditorCommand(), "code")
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

// ---- helpers ----

func newCreateFormForTest(t *testing.T, projects []domain.Project) CreateFormModel {
	t.Helper()
	svc, _, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	projectsSvc, _ := newProjectsServiceWithMocks(t)
	return NewCreateForm(styles.New(), svc, projectsSvc, projects, uuid.Nil, testLaunchers(t), testEditors(t), 100)
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
	}
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}
