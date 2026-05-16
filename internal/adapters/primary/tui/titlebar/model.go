package titlebar

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// SetActivePaneMsg is sent to update the active pane label in the title bar.
type SetActivePaneMsg struct {
	Label string
}

type Model struct {
	width      int
	appName    string
	activePane string
	styles     *styles.Styles
}

func New(s *styles.Styles, appName string) Model {
	return Model{styles: s, appName: appName}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case SetActivePaneMsg:
		m.activePane = msg.Label
	}
	return m, nil
}

func (m Model) View() tea.View {
	branding := m.styles.TitleBar.Branding.Render(m.appName)
	right := ""
	if m.activePane != "" {
		right = m.styles.TitleBar.Subtext.Render(m.activePane)
	}
	gap := ""
	if m.width > 0 {
		brandingWidth := lipgloss.Width(branding)
		rightWidth := lipgloss.Width(right)
		gapWidth := m.width - brandingWidth - rightWidth
		if gapWidth > 0 {
			gap = m.styles.TitleBar.Base.Render(strings.Repeat(" ", gapWidth))
		}
	}
	return tea.NewView(branding + gap + right)
}
