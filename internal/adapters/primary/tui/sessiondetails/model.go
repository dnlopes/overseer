package sessiondetails

import (
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
)

// Model renders a read-only details card for the currently selected session.
// It listens for SessionSelectedMsg to track the selection, for
// PRStatusUpdatedMsg to populate its own PR cache (independent of the
// dashboard's cache), and for SessionsLoadedMsg to reconcile when the
// selected session is renamed or removed.
type Model struct {
	styles *styles.Styles
	width  int
	height int

	session *domain.Session
	prCache map[uuid.UUID]shared.PRStatusUpdatedMsg
}

func New(s *styles.Styles) Model {
	return Model{
		styles:  s,
		prCache: make(map[uuid.UUID]shared.PRStatusUpdatedMsg),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.SessionSelectedMsg:
		sess := msg.Session
		m.session = &sess
	case shared.PRStatusUpdatedMsg:
		m.prCache[msg.SessionID] = msg
	case shared.SessionsLoadedMsg:
		if msg.Err == nil {
			m.reconcileSession(msg.Sessions)
		}
	}
	return m, nil
}

// reconcileSession refreshes the cached selected-session struct against
// the latest list. If the selection no longer exists (deleted), the
// session pointer is cleared.
func (m *Model) reconcileSession(sessions []domain.Session) {
	if m.session == nil {
		return
	}
	for _, s := range sessions {
		if s.ID == m.session.ID {
			s := s
			m.session = &s
			return
		}
	}
	m.session = nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.NewView("")
	}
	innerW, _ := components.PanelInnerSize(m.styles, false, m.width, m.height)
	content := m.renderContent(innerW)
	return components.PanelWithSize(m.styles, content, false, m.width, m.height)
}
