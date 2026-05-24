package dashboard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
)

type StatusModel struct {
	aggregate map[domain.AgentStatusKind]int
	styles    *styles.Styles
	width     int
}

func newStatus(s *styles.Styles) StatusModel {
	return StatusModel{
		aggregate: map[domain.AgentStatusKind]int{},
		styles:    s,
		width:     80,
	}
}

func (m StatusModel) Init() tea.Cmd {
	return nil
}

func (m *StatusModel) SetSize(width, _ int) {
	m.width = width
}

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case shared.AgentStatusesUpdatedMsg:
		if msg.Err != nil {
			return m, nil
		}
		agg := make(map[domain.AgentStatusKind]int, 5)
		for _, st := range msg.Statuses {
			agg[st.Kind]++
		}
		m.aggregate = agg
	}
	return m, nil
}

var statusBarOrder = []domain.AgentStatusKind{
	domain.AgentStatusRunning,
	domain.AgentStatusWaiting,
	domain.AgentStatusDead,
	domain.AgentStatusIdle,
	domain.AgentStatusUnknown,
}

func (m StatusModel) View() tea.View {
	parts := make([]string, 0, len(statusBarOrder))
	for _, kind := range statusBarOrder {
		count := m.aggregate[kind]
		if count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", m.styles.Glyphs.AgentStatus(kind), count))
	}
	content := strings.Join(parts, " · ")
	return tea.NewView(m.styles.StatusBar.Width(m.width).Render(content))
}
