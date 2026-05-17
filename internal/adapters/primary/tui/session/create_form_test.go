package session

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func newCreateFormService(sessions []domain.Session) *service.SessionService {
	repo := &mocks.MockSessionRepository{ListResult: sessions}
	tmux := &mocks.MockTmuxAdapter{}
	git := &mocks.MockGitAdapter{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewSessionService(repo, tmux, git, logger)
}

func newCreateFormFixture() CreateFormModel {
	svc := newCreateFormService([]domain.Session{})
	return NewCreateForm(styles.New(), svc)
}

func TestCreateForm_HappyPath(t *testing.T) {
	m := newCreateFormFixture()

	m.nameInput.SetValue("my-session")
	m.projectInput.SetValue("my-project")

	result, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected a cmd after Enter with valid inputs")
	}

	msg := cmd()
	created, ok := msg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("expected SessionCreatedMsg, got %T", msg)
	}
	if created.Session.Name != "my-session" {
		t.Errorf("session.Name: want %q, got %q", "my-session", created.Session.Name)
	}
	if created.Session.ProjectName != "my-project" {
		t.Errorf("session.ProjectName: want %q, got %q", "my-project", created.Session.ProjectName)
	}

	_, ok = result.(CreateFormModel)
	if !ok {
		t.Fatalf("expected CreateFormModel back, got %T", result)
	}
}

func TestCreateForm_EmptyName(t *testing.T) {
	m := newCreateFormFixture()

	result, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatal("expected no cmd when name is empty")
	}

	form, ok := result.(CreateFormModel)
	if !ok {
		t.Fatalf("expected CreateFormModel back, got %T", result)
	}
	if form.errMsg == "" {
		t.Error("expected errMsg to be set for empty name")
	}
	if !strings.Contains(form.errMsg, "required") {
		t.Errorf("errMsg should mention 'required', got %q", form.errMsg)
	}
}

func TestCreateForm_Esc(t *testing.T) {
	m := newCreateFormFixture()

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd == nil {
		t.Fatal("expected a cmd after Esc")
	}

	msg := cmd()
	_, ok := msg.(CancelFormMsg)
	if !ok {
		t.Fatalf("expected CancelFormMsg, got %T", msg)
	}
}

func TestCreateForm_TabCycles(t *testing.T) {
	m := newCreateFormFixture()

	if m.focusIndex != 0 {
		t.Fatalf("initial focusIndex: want 0, got %d", m.focusIndex)
	}

	result, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if cmd != nil {
		t.Fatal("Tab should not produce a cmd")
	}
	form := result.(CreateFormModel)
	if form.focusIndex != 1 {
		t.Fatalf("focusIndex after Tab: want 1, got %d", form.focusIndex)
	}
	if form.projectInput.Focused() != true {
		t.Error("projectInput should be focused after Tab")
	}
	if form.nameInput.Focused() != false {
		t.Error("nameInput should be blurred after Tab")
	}

	result2, _ := form.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	form2 := result2.(CreateFormModel)
	if form2.focusIndex != 0 {
		t.Fatalf("focusIndex after second Tab: want 0, got %d", form2.focusIndex)
	}
	if form2.nameInput.Focused() != true {
		t.Error("nameInput should be focused after second Tab")
	}
}

func TestCreateForm_EmptyProject(t *testing.T) {
	m := newCreateFormFixture()
	m.nameInput.SetValue("my-session")

	result, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatal("expected no cmd when project is empty")
	}

	form, ok := result.(CreateFormModel)
	if !ok {
		t.Fatalf("expected CreateFormModel back, got %T", result)
	}
	if form.errMsg == "" {
		t.Error("expected errMsg to be set for empty project")
	}
	if !strings.Contains(form.errMsg, "required") {
		t.Errorf("errMsg should mention 'required', got %q", form.errMsg)
	}
}

func TestCreateForm_ViewContainsHelp(t *testing.T) {
	m := newCreateFormFixture()
	view := m.View().Content
	if !strings.Contains(view, "Tab") || !strings.Contains(view, "Enter") || !strings.Contains(view, "Esc") {
		t.Errorf("View should contain help hints, got: %q", view)
	}
}
