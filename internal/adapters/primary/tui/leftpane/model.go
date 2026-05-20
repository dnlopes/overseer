package leftpane

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	projectui "github.com/dnlopes/overseer/internal/adapters/primary/tui/project"
	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

type Model struct {
	sessions sessionui.Model
	projects projectui.Model
	active   shared.LeftPaneTab
	styles   *styles.Styles
	width    int
	height   int
	focused  bool
}

func New(s *styles.Styles, sessions sessionui.Model, projects projectui.Model) Model {
	return Model{
		sessions: sessions,
		projects: projects,
		active:   shared.LeftPaneTabSessions,
		styles:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.sessions.Init(), m.projects.Init())
}

func (m Model) ActiveTab() shared.LeftPaneTab {
	return m.active
}

func (m *Model) SetProjectNameLookup(names map[uuid.UUID]string) {
	m.sessions.SetProjectNames(names)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && m.focused {
		if key.Matches(keyMsg, sessionsTabKeyBinding) {
			if m.active == shared.LeftPaneTabSessions {
				return m, nil
			}
			m.active = shared.LeftPaneTabSessions
			m.sessions.SetFocus(true)
			m.projects.SetFocus(false)
			m.applySize()
			return m, shared.Emit(shared.LeftPaneTabChangedMsg{Tab: shared.LeftPaneTabSessions})
		}
		if key.Matches(keyMsg, projectsTabKeyBinding) {
			if m.active == shared.LeftPaneTabProjects {
				return m, nil
			}
			m.active = shared.LeftPaneTabProjects
			m.projects.SetFocus(true)
			m.sessions.SetFocus(false)
			m.applySize()
			return m, shared.Emit(shared.LeftPaneTabChangedMsg{Tab: shared.LeftPaneTabProjects})
		}
	}

	switch typed := msg.(type) {
	case shared.SessionsLoadedMsg, shared.SessionCreatedMsg, shared.SessionReorderedMsg, shared.SessionDeletedMsg:
		var cmd tea.Cmd
		m.sessions, cmd = shared.UpdateModel(m.sessions, typed)
		return m, cmd
	case shared.ProjectsLoadedMsg, shared.ProjectRegisteredMsg:
		var cmds []tea.Cmd
		var c1, c2 tea.Cmd
		m.projects, c1 = shared.UpdateModel(m.projects, typed)
		m.sessions, c2 = shared.UpdateModel(m.sessions, typed)
		if c1 != nil {
			cmds = append(cmds, c1)
		}
		if c2 != nil {
			cmds = append(cmds, c2)
		}
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	switch m.active {
	case shared.LeftPaneTabSessions:
		m.sessions, cmd = shared.UpdateModel(m.sessions, msg)
	case shared.LeftPaneTabProjects:
		m.projects, cmd = shared.UpdateModel(m.projects, msg)
	}
	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.applySize()
}

func (m *Model) applySize() {
	tabsHeight := 1
	childHeight := max(m.height-tabsHeight, 1)
	m.sessions.SetSize(m.width, childHeight)
	m.projects.SetSize(m.width, childHeight)
}

func (m *Model) SetFocus(focused bool) {
	m.focused = focused
	switch m.active {
	case shared.LeftPaneTabSessions:
		m.sessions.SetFocus(focused)
		m.projects.SetFocus(false)
	case shared.LeftPaneTabProjects:
		m.projects.SetFocus(focused)
		m.sessions.SetFocus(false)
	}
}

func (m Model) IsFocused() bool {
	return m.focused
}

func (m Model) SessionsActive() bool {
	return m.active == shared.LeftPaneTabSessions
}

func (m Model) ProjectsActive() bool {
	return m.active == shared.LeftPaneTabProjects
}

func (m Model) SelectedSessionID() string {
	if m.active != shared.LeftPaneTabSessions {
		return ""
	}
	return m.sessions.SelectedSessionID()
}

func (m Model) View() tea.View {
	tabsRow := m.renderTabs()
	tabsHeight := lipgloss.Height(tabsRow)
	childHeight := max(m.height-tabsHeight, 1)

	var childContent string
	switch m.active {
	case shared.LeftPaneTabSessions:
		sessions := m.sessions
		sessions.SetSize(m.width, childHeight)
		childContent = sessions.View().Content
	case shared.LeftPaneTabProjects:
		projects := m.projects
		projects.SetSize(m.width, childHeight)
		childContent = projects.View().Content
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, tabsRow, childContent))
}

func (m Model) renderTabs() string {
	sessionsTab := tabLabel(m.styles, "1. Sessions", m.active == shared.LeftPaneTabSessions)
	projectsTab := tabLabel(m.styles, "2. Projects", m.active == shared.LeftPaneTabProjects)
	row := sessionsTab + projectsTab
	rendered := lipgloss.Width(row)
	if m.width > rendered {
		row += strings.Repeat(" ", m.width-rendered)
	}
	return row
}

func tabLabel(s *styles.Styles, label string, active bool) string {
	if active {
		return s.Tab.Active.Render(label)
	}
	return s.Tab.Inactive.Render(label)
}
