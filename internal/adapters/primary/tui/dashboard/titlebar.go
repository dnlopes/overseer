package dashboard

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// TitlebarSetActivePaneMsg updates the active pane label in the title bar.
type TitlebarSetActivePaneMsg struct {
	Label string
}

// TitlebarModel renders the top title bar with app branding + active pane label.
type TitlebarModel struct {
	width   int
	appName string
	styles  *styles.Styles
}

func newTitlebar(s *styles.Styles, appName string) TitlebarModel {
	return TitlebarModel{styles: s, appName: appName}
}

func (m TitlebarModel) Init() tea.Cmd { return nil }

func (m TitlebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m TitlebarModel) View() tea.View {
	branding := m.styles.TitleBar.Branding.Render(m.appName)
	right := ""
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
