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
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func TestCreateForm_DefaultsToNoProject(t *testing.T) {
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), nil)

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
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer, widgets})

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
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer, widgets})

	updated, _ := form.Update(formKeyPress("tab"))
	updated, _ = updated.(CreateFormModel).Update(formKeyPress("left"))

	got := updated.(CreateFormModel)
	if got.selectedProjectID() != widgets.ID {
		t.Fatalf("after left from (none): selected = %v, want %v", got.selectedProjectID(), widgets.ID)
	}
}

func TestCreateForm_SubmitCreatesSessionWithSelectedProject(t *testing.T) {
	overseer := testutil.MakeProject("/repo/overseer", "Overseer")
	svc, repo, tmux, git := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "alpha").Return("tmux-alpha", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "alpha").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, []domain.Project{overseer})

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
	svc, repo, tmux, git := newCreateFormSessionServiceWithMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, nil).Once()
	tmux.EXPECT().CreateSession(mock.Anything, "orphan").Return("tmux-orphan", nil).Once()
	git.EXPECT().CreateWorktree(mock.Anything, "main", "orphan").Return(nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
	form := NewCreateForm(styles.New(), svc, nil)

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
	form := NewCreateForm(styles.New(), newCreateFormSessionService(t), []domain.Project{overseer})

	view := form.View().Content
	if !strings.Contains(view, "(none)") {
		t.Fatalf("View() missing default '(none)' label: %q", view)
	}
}

func newCreateFormSessionService(t *testing.T) service.SessionService {
	t.Helper()
	svc, _, _, _ := newCreateFormSessionServiceWithMocks(t)
	return svc
}

func newCreateFormSessionServiceWithMocks(t *testing.T) (service.SessionService, *mocks.MockSessionRepository, *mocks.MockTmuxAdapter, *mocks.MockGitAdapter) {
	t.Helper()
	repo := mocks.NewMockSessionRepository(t)
	tmux := mocks.NewMockTmuxAdapter(t)
	git := mocks.NewMockGitAdapter(t)
	return *service.NewSessionService(repo, tmux, git, slog.Default()), repo, tmux, git
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
