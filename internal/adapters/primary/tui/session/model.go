package session

import (
	"context"
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
}

type Model struct {
	sessions     []domain.Session
	projectNames map[uuid.UUID]string
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
	m.tree = m.tree.SetNodes(m.sessionTreeNodes()).ExpandAll()
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
		if len(m.sessions) == 0 {
			return m, nil
		}
		first := m.sessions[0]
		m.tree = m.tree.SelectID("session:" + first.ID.String())
		return m, shared.Emit(shared.SessionSelectedMsg{Session: first})
	case shared.SessionReorderedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.sessions = msg.Sessions
		m.rebuildTree()
		if msg.FocusID != "" {
			if sess, ok := m.findSession(msg.FocusID); ok {
				m.tree = m.tree.SelectID("session:" + msg.FocusID)
				return m, shared.Emit(shared.SessionSelectedMsg{Session: sess})
			}
		}
		return m, nil
	case shared.SessionCreatedMsg:
		return m, m.loadSessions()
	case shared.SessionDeletedMsg:
		return m, m.loadSessions()
	case components.TreeSelectMsg[sessionNode]:
		if msg.Item.kind == sessionNodeSession {
			if sess, ok := m.findSession(msg.Item.sessionID); ok {
				return m, shared.Emit(shared.SessionSelectedMsg{Session: sess})
			}
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.focused {
			if cmd, handled := m.handleNavigationKey(msg); handled {
				return m, cmd
			}
		}
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, m.translateTreeSelection(cmd)
}

func (m *Model) handleNavigationKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, jumpUpKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveCursor(-jumpRowDelta)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, jumpDownKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveCursor(jumpRowDelta)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, nextGroupKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveToNext(isGroupNode)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, prevGroupKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveToPrev(isGroupNode)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, reorderUpKeyBinding):
		return m.reorderSelected(-1), true
	case key.Matches(msg, reorderDownKeyBinding):
		return m.reorderSelected(1), true
	case key.Matches(msg, deleteSessionKeyBinding):
		if cmd := m.requestDeleteSelected(); cmd != nil {
			return cmd, true
		}
		return nil, true
	}
	return nil, false
}

func (m Model) requestDeleteSelected() tea.Cmd {
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return nil
	}
	id, err := uuid.Parse(cur.sessionID)
	if err != nil {
		return nil
	}
	for _, sess := range m.sessions {
		if sess.ID == id {
			return shared.Emit(shared.SessionDeleteRequestedMsg{Session: sess})
		}
	}
	return nil
}

func (m Model) reorderSelected(direction int) tea.Cmd {
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return nil
	}
	sessID, err := uuid.Parse(cur.sessionID)
	if err != nil {
		return nil
	}
	svc := m.service
	return func() tea.Msg {
		_, err := svc.Reorder(context.Background(), service.ReorderSessionRequest{
			ID:        sessID,
			Direction: direction,
		})
		if err != nil {
			return shared.SessionReorderedMsg{Err: err}
		}
		listResp, listErr := svc.List(context.Background(), service.ListSessionsRequest{})
		return shared.SessionReorderedMsg{
			Sessions: listResp.Sessions,
			FocusID:  cur.sessionID,
			Err:      listErr,
		}
	}
}

func isGroupNode(n sessionNode) bool {
	return n.kind == sessionNodeGroup
}

func (m Model) translateTreeSelection(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return nil
	}
	sess, ok := m.findSession(cur.sessionID)
	if !ok {
		return nil
	}
	return shared.Emit(shared.SessionSelectedMsg{Session: sess})
}

func (m Model) findSession(id string) (domain.Session, bool) {
	for _, sess := range m.sessions {
		if sess.ID.String() == id {
			return sess, true
		}
	}
	return domain.Session{}, false
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

func (m Model) SelectedSessionID() string {
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return ""
	}
	return cur.sessionID
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

func renderSessionNode(s *styles.Styles) components.TreeRenderFunc[sessionNode] {
	return func(item sessionNode, _, depth int, hasKids, expanded, focused bool) string {
		prefix := components.TreePrefix(depth, hasKids, expanded)
		if item.kind == sessionNodeGroup {
			style := s.Group.Header
			if focused {
				style = s.Group.HeaderSelected
			}
			return style.Render(prefix + item.label)
		}
		if focused {
			return s.ListRow.Selected.Render(prefix + item.label)
		}
		return s.ListRow.Normal.Render(prefix + item.label)
	}
}
