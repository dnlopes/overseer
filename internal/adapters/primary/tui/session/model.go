package session

import (
	"context"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

type sessionGroupingMode int

type sessionNodeKind int

const (
	sessionGroupingProject sessionGroupingMode = iota
	sessionGroupingNone
)

const (
	sessionNodeGroup sessionNodeKind = iota
	sessionNodeSession
)

type sessionNode struct {
	kind      sessionNodeKind
	sessionID string
	label     string
}

type Model struct {
	sessions     []domain.Session
	groupingMode sessionGroupingMode
	styles       *styles.Styles
	service      service.SessionService
	tree         components.TreeModel[sessionNode]
	focused      bool
	width        int
	height       int
	err          error
}

func New(s *styles.Styles, service service.SessionService) Model {
	tree := components.NewTree(renderSessionNode(s))
	return Model{styles: s, service: service, tree: tree, groupingMode: sessionGroupingProject}
}

func (m Model) Init() tea.Cmd {
	return m.loadSessions()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.SessionsLoadedMsg:
		m.err = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.sessions = msg.Sessions
		m.tree = m.tree.SetNodes(m.sessionTreeNodes()).ExpandAll()
		firstSessionID := firstSessionID(m.sessions)
		if firstSessionID == "" {
			return m, nil
		}
		m.tree = m.tree.SelectID("session:" + firstSessionID)
		return m, shared.Emit(shared.SessionSelectedMsg{ID: firstSessionID})
	case components.TreeSelectMsg[sessionNode]:
		if msg.Item.kind == sessionNodeSession {
			return m, shared.Emit(shared.SessionSelectedMsg{ID: msg.Item.sessionID})
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, m.translateTreeSelection(cmd)
}

func (m Model) translateTreeSelection(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return nil
	}
	return shared.Emit(shared.SessionSelectedMsg{ID: cur.sessionID})
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
		content = m.styles.EmptyState.Title.Render("Unable to load sessions")
	} else if content == "" {
		content = strings.Join([]string{
			m.styles.EmptyState.Title.Render("No sessions"),
			m.styles.EmptyState.Hint.Render("Press n to create one"),
		}, "\n")
	}
	return components.PanelWithSize(m.styles, content, m.focused, m.width, m.height)
}

// The Cmd: a function that does the work and returns a Msg
func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.List(context.Background(), service.ListSessionsRequest{})
		return shared.SessionsLoadedMsg{Sessions: result.Sessions, Err: err}
	}
}

func (m Model) sessionTreeNodes() []components.TreeNode[sessionNode] {
	if m.groupingMode == sessionGroupingNone {
		return rawSessionNodes(m.sessions)
	}
	return projectSessionNodes(m.sessions)
}

func rawSessionNodes(sessions []domain.Session) []components.TreeNode[sessionNode] {
	nodes := make([]components.TreeNode[sessionNode], len(sessions))
	for i, sess := range sessions {
		nodes[i] = sessionTreeNode(sess)
	}
	return nodes
}

func projectSessionNodes(sessions []domain.Session) []components.TreeNode[sessionNode] {
	projectSessions := make(map[string][]domain.Session)
	projectNames := make([]string, 0)
	for _, sess := range sessions {
		if _, ok := projectSessions[sess.ProjectName]; !ok {
			projectNames = append(projectNames, sess.ProjectName)
		}
		projectSessions[sess.ProjectName] = append(projectSessions[sess.ProjectName], sess)
	}
	sort.Strings(projectNames)

	nodes := make([]components.TreeNode[sessionNode], len(projectNames))
	for i, projectName := range projectNames {
		sessions := projectSessions[projectName]
		children := make([]components.TreeNode[sessionNode], len(sessions))
		for j, sess := range sessions {
			children[j] = sessionTreeNode(sess)
		}
		nodes[i] = components.TreeNode[sessionNode]{
			ID:       "project:" + projectName,
			Item:     sessionNode{kind: sessionNodeGroup, label: projectName},
			Children: children,
		}
	}
	return nodes
}

func sessionTreeNode(sess domain.Session) components.TreeNode[sessionNode] {
	return components.TreeNode[sessionNode]{
		ID: "session:" + sess.ID.String(),
		Item: sessionNode{
			kind:      sessionNodeSession,
			sessionID: sess.ID.String(),
			label:     sess.Name,
		},
	}
}

func firstSessionID(sessions []domain.Session) string {
	if len(sessions) == 0 {
		return ""
	}
	return sessions[0].ID.String()
}

func renderSessionNode(s *styles.Styles) components.TreeRenderFunc[sessionNode] {
	return func(item sessionNode, depth int, hasKids, expanded, focused bool) string {
		prefix := treePrefix(depth, hasKids, expanded)
		row := prefix + item.label
		if item.kind == sessionNodeGroup {
			row = s.Group.Header.Render(row)
		} else if focused {
			row = s.Session.Item.Selected.Render(row)
		} else {
			row = s.Session.Item.Normal.Render(row)
		}
		return row
	}
}

func treePrefix(depth int, hasKids, expanded bool) string {
	indicator := "  "
	if hasKids && expanded {
		indicator = "▾ "
	} else if hasKids {
		indicator = "▸ "
	}
	return strings.Repeat("  ", depth) + indicator
}
