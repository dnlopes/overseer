package project

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

type projectNode struct {
	projectID string
	label     string
	path      string
}

type Model struct {
	projects []domain.Project
	styles   *styles.Styles
	service  service.ProjectService
	tree     components.TreeModel[projectNode]
	focused  bool
	width    int
	height   int
	err      error
}

func New(s *styles.Styles, projectService service.ProjectService) Model {
	tree := components.NewTree(renderProjectNode(s))
	return Model{
		styles:  s,
		service: projectService,
		tree:    tree,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadProjects()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.ProjectsLoadedMsg:
		m.err = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.projects = msg.Projects
		m.tree = m.tree.SetNodes(m.projectTreeNodes()).ExpandAll()
		firstID := firstProjectID(m.projects)
		if firstID == "" {
			return m, nil
		}
		m.tree = m.tree.SelectID("project:" + firstID)
		return m, shared.Emit(shared.ProjectSelectedMsg{ID: firstID})
	case shared.ProjectRegisteredMsg:
		return m, m.loadProjects()
	case components.TreeSelectMsg[projectNode]:
		return m, shared.Emit(shared.ProjectSelectedMsg{ID: msg.Item.projectID})
	case tea.KeyPressMsg:
		if m.focused && key.Matches(msg, jumpToProjectKeyBinding) {
			return m.jumpToIndex(int(msg.String()[0] - '1'))
		}
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, m.translateTreeSelection(cmd)
}

func (m Model) jumpToIndex(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= m.tree.RowCount() {
		return m, nil
	}
	m.tree = m.tree.SelectIndex(idx)
	return m, m.tree.EmitSelection()
}

func (m Model) translateTreeSelection(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	cur, ok := m.tree.Selected()
	if !ok {
		return nil
	}
	return shared.Emit(shared.ProjectSelectedMsg{ID: cur.projectID})
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	innerW, innerH := components.PanelInnerSize(m.styles, m.focused, width, height)
	m.tree = m.tree.SetSize(innerW, innerH)
}

func (m *Model) SetFocus(focus bool) {
	m.focused = focus
	if focus {
		m.tree = m.tree.Focus()
		return
	}
	m.tree = m.tree.Blur()
}

func (m Model) IsFocused() bool {
	return m.focused
}

func (m Model) View() tea.View {
	content := m.tree.View()
	if m.err != nil {
		content = m.styles.EmptyState.Title.Render("Unable to load projects")
	} else if content == "" {
		content = strings.Join([]string{
			m.styles.EmptyState.Title.Render("No projects"),
			m.styles.EmptyState.Hint.Render("Press n to register one"),
		}, "\n")
	}
	return components.PanelWithSize(m.styles, content, m.focused, m.width, m.height)
}

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.List(context.Background(), service.ListProjectsRequest{})
		return shared.ProjectsLoadedMsg{Projects: result.Projects, Err: err}
	}
}

func (m Model) projectTreeNodes() []components.TreeNode[projectNode] {
	projects := append([]domain.Project(nil), m.projects...)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	nodes := make([]components.TreeNode[projectNode], len(projects))
	for i, p := range projects {
		nodes[i] = components.TreeNode[projectNode]{
			ID: "project:" + p.ID.String(),
			Item: projectNode{
				projectID: p.ID.String(),
				label:     p.Name,
				path:      p.Path,
			},
		}
	}
	return nodes
}

func firstProjectID(projects []domain.Project) string {
	if len(projects) == 0 {
		return ""
	}
	return projects[0].ID.String()
}

func renderProjectNode(s *styles.Styles) components.TreeRenderFunc[projectNode] {
	return func(item projectNode, index, _ int, _, _, focused bool) string {
		number := fmt.Sprintf("%02d. ", index+1)
		if focused {
			return s.ListRow.NumberSelected.Render(number) + s.ListRow.Selected.Render(item.label)
		}
		return s.ListRow.Number.Render(number) + s.ListRow.Normal.Render(item.label)
	}
}
