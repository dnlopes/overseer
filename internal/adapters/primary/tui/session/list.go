package session

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

type groupsLoadedMsg struct {
	groups         []service.SessionGroup
	err            error
	selectedID     uuid.UUID
	preserveCursor bool
}

type ReorderRequestMsg struct {
	Direction int
}

type SessionGroup = service.SessionGroup

type Model struct {
	groups  []SessionGroup
	cursor  int
	styles  *styles.Styles
	focused bool
	width   int
	height  int
	svc     *service.SessionService
}

func New(s *styles.Styles, svc *service.SessionService) Model {
	return Model{
		styles: s,
		svc:    svc,
	}
}

func (m Model) Init() tea.Cmd {
	if m.svc == nil {
		return nil
	}
	return func() tea.Msg {
		resp, err := m.svc.List(context.Background(), service.ListSessionsRequest{})
		if err != nil {
			return groupsLoadedMsg{err: err}
		}
		return groupsLoadedMsg{groups: resp.Groups}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case groupsLoadedMsg:
		if msg.err == nil {
			m.groups = msg.groups
			if msg.preserveCursor {
				m.cursor = m.indexOf(msg.selectedID)
			} else {
				m.cursor = 0
			}
		}

	case tea.KeyPressMsg:
		total := m.totalItems()
		switch msg.String() {
		case "j":
			if total > 0 && m.cursor < total-1 {
				m.cursor++
			}
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "J":
			return m, func() tea.Msg { return ReorderRequestMsg{Direction: +1} }
		case "K":
			return m, func() tea.Msg { return ReorderRequestMsg{Direction: -1} }
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

func (m Model) render() string {
	var content string
	if m.totalItems() == 0 {
		content = m.renderEmpty()
	} else {
		content = m.renderGroups()
	}

	var border lipgloss.Style
	if m.focused {
		border = m.styles.Border.Focused
	} else {
		border = m.styles.Border.Blurred
	}
	if m.width > 0 {
		contentWidth := m.width - border.GetHorizontalFrameSize()
		contentWidth = max(contentWidth, 1)
		content = lipgloss.NewStyle().Width(contentWidth).Render(content)
	}
	if m.height > 0 {
		contentHeight := m.height - border.GetVerticalFrameSize()
		contentHeight = max(contentHeight, 1)
		content = lipgloss.NewStyle().Height(contentHeight).Render(content)
	}
	return border.Render(content)
}

func (m Model) renderEmpty() string {
	title := m.styles.EmptyState.Title.Render("No sessions yet")
	hint := components.KeyBadge(m.styles, "n", "create session")
	content := title + "\n" + hint

	if m.width <= 0 || m.height <= 0 {
		return content
	}

	var border lipgloss.Style
	if m.focused {
		border = m.styles.Border.Focused
	} else {
		border = m.styles.Border.Blurred
	}

	w := max(m.width-border.GetHorizontalFrameSize(), 1)
	h := max(m.height-border.GetVerticalFrameSize(), 1)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderGroups() string {
	lines := make([]string, 0, len(m.groups)*2)
	flatIdx := 0

	for _, g := range m.groups {
		lines = append(lines, m.styles.Group.Header.Render(g.ProjectName))
		for _, s := range g.Sessions {
			if flatIdx == m.cursor {
				lines = append(lines, m.styles.ListRow.Selected.Render(s.Name))
			} else {
				lines = append(lines, m.styles.ListRow.Normal.Render(s.Name))
			}
			flatIdx++
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) totalItems() int {
	n := 0
	for _, g := range m.groups {
		n += len(g.Sessions)
	}
	return n
}

func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

func (m Model) SelectedSession() (domain.Session, bool) {
	if m.totalItems() == 0 {
		return domain.Session{}, false
	}
	flatIdx := 0
	for _, g := range m.groups {
		for _, s := range g.Sessions {
			if flatIdx == m.cursor {
				return s, true
			}
			flatIdx++
		}
	}
	return domain.Session{}, false
}

func (m Model) Cursor() int {
	return m.cursor
}

func (m Model) ReloadPreservingSelection(id uuid.UUID) tea.Cmd {
	if m.svc == nil {
		return nil
	}
	return func() tea.Msg {
		resp, err := m.svc.List(context.Background(), service.ListSessionsRequest{})
		if err != nil {
			return groupsLoadedMsg{err: err, selectedID: id, preserveCursor: true}
		}
		return groupsLoadedMsg{groups: resp.Groups, selectedID: id, preserveCursor: true}
	}
}

func (m Model) indexOf(id uuid.UUID) int {
	flatIdx := 0
	for _, g := range m.groups {
		for _, s := range g.Sessions {
			if s.ID == id {
				return flatIdx
			}
			flatIdx++
		}
	}
	if total := m.totalItems(); total > 0 && m.cursor >= total {
		return total - 1
	}
	return m.cursor
}

func (m Model) Keybindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "move down")),
		key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "move up")),
		key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "reorder down")),
		key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "reorder up")),
	}
}
