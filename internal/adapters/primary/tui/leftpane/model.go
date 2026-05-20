package leftpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

type Model struct {
	sessions sessionui.Model
	styles   *styles.Styles
	width    int
	height   int
}

func New(s *styles.Styles, sessions sessionui.Model) Model {
	return Model{
		sessions: sessions,
		styles:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return m.sessions.Init()
}

func (m *Model) SetProjectNameLookup(names map[uuid.UUID]string) {
	m.sessions.SetProjectNames(names)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sessions, cmd = shared.UpdateModel(m.sessions, msg)
	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.sessions.SetSize(width, height)
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

func (m Model) View() tea.View {
	return m.sessions.View()
}
