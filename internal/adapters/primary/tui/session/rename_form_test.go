package session

import (
	"log/slog"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func newTestSession(name string) domain.Session {
	s, err := domain.NewSession(name, "test-project")
	if err != nil {
		panic(err)
	}
	return s
}

func newRenameFormFixture(current domain.Session) (RenameFormModel, *mocks.MockSessionRepository) {
	mock := &mocks.MockSessionRepository{
		GetResult:  current,
		ListResult: []domain.Session{current},
	}
	svc := service.NewSessionService(mock, &mocks.MockTmuxAdapter{}, &mocks.MockGitAdapter{}, slog.Default())
	m := NewRenameForm(styles.New(), svc, current)
	return m, mock
}

func TestRenameForm_HappyPath(t *testing.T) {
	current := newTestSession("old-name")
	m, _ := newRenameFormFixture(current)

	if m.nameInput.Value() != "old-name" {
		t.Fatalf("expected input pre-filled with %q, got %q", "old-name", m.nameInput.Value())
	}

	m.nameInput.SetValue("new-name")

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected a cmd after Enter with valid name")
	}

	msg := cmd()
	renamed, ok := msg.(SessionRenamedMsg)
	if !ok {
		t.Fatalf("expected SessionRenamedMsg, got %T", msg)
	}
	if renamed.Session.Name != "new-name" {
		t.Errorf("expected session.Name=%q, got %q", "new-name", renamed.Session.Name)
	}
}

func TestRenameForm_EmptyName(t *testing.T) {
	current := newTestSession("old-name")
	m, _ := newRenameFormFixture(current)

	m.nameInput.SetValue("")

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatal("expected no cmd for empty name")
	}

	form, ok := updated.(RenameFormModel)
	if !ok {
		t.Fatalf("expected RenameFormModel back, got %T", updated)
	}
	if form.errMsg == "" {
		t.Error("expected errMsg to be set when name is empty")
	}
}

func TestRenameForm_Esc(t *testing.T) {
	current := newTestSession("old-name")
	m, _ := newRenameFormFixture(current)

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
