package leftpane

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/sessiondetails"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// sessionDetailsHeightPercent is the share of the left-pane height
// reserved for the session-details card. The remainder goes to the
// session list above. No minimum floor: on short terminals the card
// clips gracefully (least-important rows drop first via the renderer).
const sessionDetailsHeightPercent = 50

type Model struct {
	sessions       sessionui.Model
	sessionDetails sessiondetails.Model
	styles         *styles.Styles
	width          int
	height         int
}

func New(s *styles.Styles, sessions sessionui.Model, details sessiondetails.Model) Model {
	return Model{
		sessions:       sessions,
		sessionDetails: details,
		styles:         s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.sessions.Init(), m.sessionDetails.Init())
}

func (m *Model) SetProjectNameLookup(names map[uuid.UUID]string) {
	m.sessions.SetProjectNames(names)
	m.sessionDetails.SetProjectNames(names)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case shared.SessionsLoadedMsg:
		return m, shared.Broadcast(typed,
			shared.Forward(&m.sessions),
			shared.Forward(&m.sessionDetails),
		)
	case shared.SessionSelectedMsg, shared.SessionSelectionClearedMsg, shared.PRStatusUpdatedMsg, shared.ProjectCurrentBranchLoadedMsg:
		var cmd tea.Cmd
		m.sessionDetails, cmd = shared.UpdateModel(m.sessionDetails, typed)
		return m, cmd
	case shared.AgentStatusesUpdatedMsg:
		var cmd tea.Cmd
		m.sessions, cmd = shared.UpdateModel(m.sessions, typed)
		return m, cmd
	}

	var cmd tea.Cmd
	m.sessions, cmd = shared.UpdateModel(m.sessions, msg)
	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	listH, detailsH := splitSessionsHeight(height)
	m.sessions.SetSize(width, listH)
	m.sessionDetails.SetSize(width, detailsH)
}

func splitSessionsHeight(contentHeight int) (listH, detailsH int) {
	detailsH = contentHeight * sessionDetailsHeightPercent / 100
	listH = max(contentHeight-detailsH, 1)
	return
}

func (m *Model) SetFocus(focused bool) {
	m.sessions.SetFocus(focused)
}

func (m Model) IsFocused() bool {
	return m.sessions.IsFocused()
}

func (m Model) SelectedSessionID() string {
	return m.sessions.SelectedSessionID()
}

func (m Model) SelectedSession() (domain.Session, bool) {
	return m.sessions.SelectedSession()
}

func (m Model) View() tea.View {
	listH, detailsH := splitSessionsHeight(m.height)
	sessions := m.sessions
	sessions.SetSize(m.width, listH)
	details := m.sessionDetails
	details.SetSize(m.width, detailsH)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		sessions.View().Content,
		details.View().Content,
	))
}
