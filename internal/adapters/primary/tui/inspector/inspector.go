package inspector

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

const tabStripHeight = 1

var (
	ToggleViewKeyBinding = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle view"))
)

// Model is the dashboard's right panel. It owns a fixed list of preview views
// (Agent, Shell, …) rendered as a tab strip on top of the active view's body.
// Only the active view polls; views become quiescent when inactive because
// the inspector stops forwarding messages to them.
type Model struct {
	views     []View
	activeIx  int
	width     int
	height    int
	focused   bool
	sessionID uuid.UUID
	styles    *styles.Styles
}

func New(s *styles.Styles, sessionService service.SessionService, previewRefreshInterval time.Duration) Model {
	return Model{
		views: []View{
			newAgentView(sessionService, s, previewRefreshInterval),
			newShellView(sessionService, s, previewRefreshInterval),
		},
		activeIx: 0,
		styles:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return m.views[m.activeIx].Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.SessionSelectedMsg:
		m.sessionID = msg.Session.ID
		m.activeIx = 0
		for i := range m.views {
			m.views[i].SetSession(msg.Session.ID)
		}
		return m, m.views[m.activeIx].Init()

	case shared.SessionSelectionClearedMsg:
		m.sessionID = uuid.Nil
		for i := range m.views {
			m.views[i].SetSession(uuid.Nil)
		}
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, ToggleViewKeyBinding) {
			m.activeIx = (m.activeIx + 1) % len(m.views)
			return m, m.views[m.activeIx].Init()
		}
		return m, nil

	case previewCapturedMsg:
		updated, cmd := m.views[m.activeIx].Update(msg)
		m.views[m.activeIx] = updated
		return m, cmd
	}
	return m, nil
}

func (m Model) View() tea.View {
	innerW, _ := components.TitledPanelInnerSize(m.styles, m.focused, m.width, m.height)
	tabsRow := m.renderTabStrip(innerW)
	body := m.views[m.activeIx].Body()
	content := lipgloss.JoinVertical(lipgloss.Left, tabsRow, body)
	return components.PanelWithTitle(m.styles, content, "Preview", m.focused, m.width, m.height)
}

func (m Model) renderTabStrip(width int) string {
	labels := make([]string, 0, len(m.views))
	for i, v := range m.views {
		if i == m.activeIx {
			labels = append(labels, m.styles.Tab.Active.Render(v.Label()))
		} else {
			labels = append(labels, m.styles.Tab.Inactive.Render(v.Label()))
		}
	}
	row := strings.Join(labels, "")
	if pad := width - lipgloss.Width(row); pad > 0 {
		row += strings.Repeat(" ", pad)
	}
	return row
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	innerW, innerH := components.TitledPanelInnerSize(m.styles, m.focused, width, height)
	bodyH := max(innerH-tabStripHeight, 1)
	for i := range m.views {
		m.views[i].SetSize(innerW, bodyH)
	}
}

func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

func (m Model) ActiveViewLabel() string {
	return m.views[m.activeIx].Label()
}
