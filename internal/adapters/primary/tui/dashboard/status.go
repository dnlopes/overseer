package dashboard

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// StatusModel renders the bottom-right status segment: workdir, branch, PR, agent.
type StatusModel struct {
	workdir     string
	branch      string
	prStatus    string
	agentStatus string
	styles      *styles.Styles
	width       int
}

func newStatus(s *styles.Styles) StatusModel {
	wd, _ := os.Getwd()
	return StatusModel{
		workdir:     wd,
		branch:      "stubbed",
		prStatus:    "—",
		agentStatus: "idle",
		styles:      s,
		width:       80,
	}
}

func (m StatusModel) Init() tea.Cmd {
	return nil
}

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m StatusModel) View() tea.View {
	branchSeg := m.styles.StatusSegment.Default.Render(m.branch)
	prSeg := m.styles.StatusSegment.Default.Render(m.prStatus)
	agentSeg := m.styles.StatusSegment.Highlight.Render(m.agentStatus)

	trailing := branchSeg + prSeg + agentSeg

	wdSegPadWidth := lipgloss.Width(m.styles.StatusSegment.Default.Render(""))
	available := m.width - lipgloss.Width(trailing) - wdSegPadWidth
	available = max(available, 0)

	wd := truncate(m.workdir, available)
	wdSeg := m.styles.StatusSegment.Default.Render(wd)

	return tea.NewView(wdSeg + trailing)
}

func truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth < 3 {
		return "..."
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}
