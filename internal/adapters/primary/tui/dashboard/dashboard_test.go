package dashboard

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/jobs"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
	"github.com/dnlopes/overseer/internal/shared/paths"
	"github.com/dnlopes/overseer/internal/testutil"
	"github.com/dnlopes/overseer/internal/testutil/mocks"
)

func newTestDashboard(t *testing.T) Model {
	t.Helper()

	repo := mocks.NewMockSessionRepository(t)
	projects := mocks.NewMockProjectRepository(t)
	tmux := mocks.NewMockTmuxAdapter(t)
	git := mocks.NewMockGitAdapter(t)
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode")
	defaultEditor, _ := domain.NewEditor("VSCode", "code")

	sessSvc := service.NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), defaultLauncher, defaultEditor, slog.Default())
	projSvc := service.NewProjectService(projects, git, slog.Default())

	return New(
		styles.New(),
		*sessSvc,
		*projSvc,
		jobs.Model{},
		[]domain.Launcher{defaultLauncher},
		[]domain.Editor{defaultEditor},
		domain.DefaultLabels,
		60, 15,
		500*time.Millisecond,
	)
}

func TestDashboard_SessionSelectedMsg_ForwardsToInspector(t *testing.T) {
	m := newTestDashboard(t)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	before := ansi.Strip(m.View().Content)
	if !strings.Contains(before, "Select a session to preview") {
		t.Fatalf("setup: expected inspector to show 'Select a session to preview' before selection, got:\n%s", before)
	}

	sess := testutil.MakeSession("alpha", uuid.New())
	updated, _ = m.Update(shared.SessionSelectedMsg{Session: sess})
	m = updated.(Model)

	after := ansi.Strip(m.View().Content)
	if strings.Contains(after, "Select a session to preview") {
		t.Errorf("inspector still shows 'Select a session to preview' after SessionSelectedMsg; the message was not forwarded to the inspector. View:\n%s", after)
	}
}

func TestDashboard_SessionSelectionClearedMsg_ForwardsToInspector(t *testing.T) {
	m := newTestDashboard(t)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	sess := testutil.MakeSession("alpha", uuid.New())
	updated, _ = m.Update(shared.SessionSelectedMsg{Session: sess})
	m = updated.(Model)

	before := ansi.Strip(m.View().Content)
	if strings.Contains(before, "Select a session to preview") {
		t.Fatalf("setup: expected inspector to show a session after selection, got:\n%s", before)
	}

	updated, _ = m.Update(shared.SessionSelectionClearedMsg{})
	m = updated.(Model)

	after := ansi.Strip(m.View().Content)
	if !strings.Contains(after, "Select a session to preview") {
		t.Errorf("inspector should show 'Select a session to preview' after SessionSelectionClearedMsg; the message was not forwarded to the inspector. View:\n%s", after)
	}
}
