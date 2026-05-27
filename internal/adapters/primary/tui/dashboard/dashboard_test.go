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
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
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

func TestDashboard_AgentStatusesUpdatedMsg_RendersAggregateInSessionsTitle(t *testing.T) {
	m := newTestDashboardNoEmoji(t)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 30})
	m = updated.(Model)

	statuses := map[uuid.UUID]domain.AgentStatus{
		uuid.New(): {Kind: domain.AgentStatusRunning},
		uuid.New(): {Kind: domain.AgentStatusRunning},
		uuid.New(): {Kind: domain.AgentStatusRunning},
		uuid.New(): {Kind: domain.AgentStatusWaiting},
		uuid.New(): {Kind: domain.AgentStatusDead},
	}

	updated, _ = m.Update(shared.AgentStatusesUpdatedMsg{Statuses: statuses})
	m = updated.(Model)

	view := ansi.Strip(m.View().Content)
	want := "Sessions (● 3 running ◐ 1 waiting ■ 1 dead)"
	if !strings.Contains(view, want) {
		t.Errorf("Sessions box title missing aggregate %q in view:\n%s", want, view)
	}

	lines := strings.Split(view, "\n")
	titleLine := ""
	for _, line := range lines {
		if strings.Contains(line, "Sessions (") {
			titleLine = line
			break
		}
	}
	if titleLine == "" {
		t.Fatalf("Sessions box border line with aggregate not found in view:\n%s", view)
	}
	if !strings.Contains(titleLine, want) {
		t.Errorf("aggregate %q should sit immediately after the Sessions title on the border line:\n%s", want, titleLine)
	}
}

func TestDashboard_AgentStatusesUpdatedMsg_OmitsZeroCounts(t *testing.T) {
	m := newTestDashboardNoEmoji(t)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 30})
	m = updated.(Model)

	statuses := map[uuid.UUID]domain.AgentStatus{
		uuid.New(): {Kind: domain.AgentStatusRunning},
	}

	updated, _ = m.Update(shared.AgentStatusesUpdatedMsg{Statuses: statuses})
	m = updated.(Model)

	view := ansi.Strip(m.View().Content)
	want := "Sessions (● 1 running)"
	if !strings.Contains(view, want) {
		t.Errorf("Sessions box title missing %q in view:\n%s", want, view)
	}
	for _, segment := range []string{"◐ 0", "■ 0", "○ 0", "? 0", "waiting", "dead", "idle", "unknown"} {
		if strings.Contains(view, segment) {
			t.Errorf("Sessions box title should omit zero-count segments, found %q in view:\n%s", segment, view)
		}
	}
}

func TestDashboard_AgentStatusesUpdatedMsg_NoStatuses_TitleIsPlainSessions(t *testing.T) {
	m := newTestDashboardNoEmoji(t)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 30})
	m = updated.(Model)

	view := ansi.Strip(m.View().Content)
	if strings.Contains(view, "Sessions (") {
		t.Errorf("plain 'Sessions' title (no parens) expected before any AgentStatusesUpdatedMsg, got:\n%s", view)
	}
	if !strings.Contains(view, "─ Sessions ") {
		t.Errorf("Sessions title not found in view:\n%s", view)
	}
}

func newTestDashboardNoEmoji(t *testing.T) Model {
	t.Helper()

	repo := mocks.NewMockSessionRepository(t)
	projects := mocks.NewMockProjectRepository(t)
	tmux := mocks.NewMockTmuxAdapter(t)
	git := mocks.NewMockGitAdapter(t)
	defaultLauncher, _ := domain.NewLauncher("OpenCode", "opencode", domain.AgentTypeOpenCode)
	defaultEditor, _ := domain.NewEditor("VSCode", "code")

	sessSvc := service.NewSessionService(repo, projects, tmux, git, paths.NewResolver(""), defaultLauncher, defaultEditor, slog.Default())
	projSvc := service.NewProjectService(projects, git, slog.Default())

	return New(
		styles.NewWithTheme("dark", true),
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
