package session

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

const unassignedProjectLabel = "(no project)"

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
	number    int
}

type Model struct {
	sessions           []domain.Session
	projectNames       map[uuid.UUID]string
	groupingMode       sessionGroupingMode
	styles             *styles.Styles
	service            service.SessionService
	tree               components.TreeModel[sessionNode]
	numberedSessionIDs []string
	focused            bool
	width              int
	height             int
	err                error
}

func New(s *styles.Styles, service service.SessionService) Model {
	tree := components.NewTree(renderSessionNode(s))
	return Model{
		styles:       s,
		service:      service,
		tree:         tree,
		groupingMode: sessionGroupingProject,
		projectNames: map[uuid.UUID]string{},
	}
}

func (m *Model) SetProjectNames(names map[uuid.UUID]string) {
	m.projectNames = names
	m.rebuildTree()
}

func (m *Model) rebuildTree() {
	nodes, ids := numberSessions(m.sessionTreeNodes())
	m.numberedSessionIDs = ids
	m.tree = m.tree.SetNodes(nodes).ExpandAll()
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
		m.rebuildTree()
		firstSessionID := firstSessionID(m.sessions)
		if firstSessionID == "" {
			return m, nil
		}
		m.tree = m.tree.SelectID("session:" + firstSessionID)
		return m, shared.Emit(shared.SessionSelectedMsg{ID: firstSessionID})
	case shared.SessionCreatedMsg:
		return m, m.loadSessions()
	case components.TreeSelectMsg[sessionNode]:
		if msg.Item.kind == sessionNodeSession {
			return m, shared.Emit(shared.SessionSelectedMsg{ID: msg.Item.sessionID})
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.focused && key.Matches(msg, jumpToSessionKeyBinding) {
			return m.jumpToSession(int(msg.String()[0] - '1'))
		}
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, m.translateTreeSelection(cmd)
}

func (m Model) jumpToSession(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.numberedSessionIDs) {
		return m, nil
	}
	sessID := m.numberedSessionIDs[idx]
	m.tree = m.tree.SelectID("session:" + sessID)
	return m, shared.Emit(shared.SessionSelectedMsg{ID: sessID})
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
	return projectSessionNodes(m.sessions, m.projectNames)
}

func rawSessionNodes(sessions []domain.Session) []components.TreeNode[sessionNode] {
	nodes := make([]components.TreeNode[sessionNode], len(sessions))
	for i, sess := range sessions {
		nodes[i] = sessionTreeNode(sess)
	}
	return nodes
}

func projectSessionNodes(sessions []domain.Session, projectNames map[uuid.UUID]string) []components.TreeNode[sessionNode] {
	grouped := make(map[uuid.UUID][]domain.Session)
	ids := make([]uuid.UUID, 0)
	for _, sess := range sessions {
		if _, ok := grouped[sess.ProjectID]; !ok {
			ids = append(ids, sess.ProjectID)
		}
		grouped[sess.ProjectID] = append(grouped[sess.ProjectID], sess)
	}
	sort.Slice(ids, func(i, j int) bool {
		return projectLabel(ids[i], projectNames) < projectLabel(ids[j], projectNames)
	})

	nodes := make([]components.TreeNode[sessionNode], len(ids))
	for i, id := range ids {
		groupSessions := grouped[id]
		children := make([]components.TreeNode[sessionNode], len(groupSessions))
		for j, sess := range groupSessions {
			children[j] = sessionTreeNode(sess)
		}
		label := projectLabel(id, projectNames)
		nodes[i] = components.TreeNode[sessionNode]{
			ID:       "project:" + id.String(),
			Item:     sessionNode{kind: sessionNodeGroup, label: label},
			Children: children,
		}
	}
	return nodes
}

func projectLabel(id uuid.UUID, names map[uuid.UUID]string) string {
	if id == uuid.Nil {
		return unassignedProjectLabel
	}
	if name, ok := names[id]; ok {
		return name
	}
	return id.String()
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
	return func(item sessionNode, _, depth int, hasKids, expanded, focused bool) string {
		if item.kind == sessionNodeGroup {
			return s.Group.Header.Render(components.TreePrefix(depth, hasKids, expanded) + item.label)
		}
		number := fmt.Sprintf("%s%02d. ", strings.Repeat(" ", depth*styles.ListIndentUnit), item.number)
		if focused {
			return s.ListRow.NumberSelected.Render(number) + s.ListRow.Selected.Render(item.label)
		}
		return s.ListRow.Number.Render(number) + s.ListRow.Normal.Render(item.label)
	}
}

// numberSessions walks tree nodes in visual (top-to-bottom) order and
// assigns a 1-based sequential number to every session-kind node. Group
// nodes are skipped — they have no number. Returns the numbered tree
// plus the list of session IDs indexed by their number-1 (so the Nth
// session ID is ids[N-1]), which the digit-jump handler uses for lookup.
func numberSessions(nodes []components.TreeNode[sessionNode]) ([]components.TreeNode[sessionNode], []string) {
	var ids []string
	var walk func(n components.TreeNode[sessionNode]) components.TreeNode[sessionNode]
	walk = func(n components.TreeNode[sessionNode]) components.TreeNode[sessionNode] {
		if n.Item.kind == sessionNodeSession {
			ids = append(ids, n.Item.sessionID)
			n.Item.number = len(ids)
		}
		for i, child := range n.Children {
			n.Children[i] = walk(child)
		}
		return n
	}
	numbered := make([]components.TreeNode[sessionNode], len(nodes))
	for i, n := range nodes {
		numbered[i] = walk(n)
	}
	return numbered, ids
}
