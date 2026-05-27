package session

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	kind        sessionNodeKind
	sessionID   string
	projectID   uuid.UUID
	label       string
	statusCode  string
	updatedAt   time.Time
	agentStatus domain.AgentStatusKind
}

type Model struct {
	sessions       []domain.Session
	projectNames   map[uuid.UUID]string
	labels         []domain.Label
	groupingMode   sessionGroupingMode
	styles         *styles.Styles
	service        service.SessionService
	tree           components.TreeModel[sessionNode]
	focused        bool
	width          int
	height         int
	err            error
	agentStatuses  map[uuid.UUID]domain.AgentStatus
}

func New(s *styles.Styles, service service.SessionService, labels []domain.Label) Model {
	tree := components.NewTree(renderSessionNode(s, labels))
	return Model{
		styles:        s,
		service:       service,
		labels:        labels,
		tree:          tree,
		groupingMode:  sessionGroupingProject,
		projectNames:  map[uuid.UUID]string{},
		agentStatuses: map[uuid.UUID]domain.AgentStatus{},
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
	case shared.SessionLabelCycledMsg:
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
	case shared.SessionRenamedMsg:
		return m, m.loadSessions()
	case shared.AgentStatusesUpdatedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.agentStatuses = msg.Statuses
		m.rebuildTree()
		return m, nil
	case components.TreeSelectMsg[sessionNode]:
		if msg.Item.kind == sessionNodeSession {
			if sess, ok := m.findSession(msg.Item.sessionID); ok {
				return m, shared.Emit(shared.SessionSelectedMsg{Session: sess})
			}
		}
		return m, shared.Emit(shared.SessionSelectionClearedMsg{})
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
	case key.Matches(msg, GoToNextGroupKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveToNext(isGroupNode)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, GoToPrevGroupKeyBinding):
		var cmd tea.Cmd
		m.tree, cmd = m.tree.MoveToPrev(isGroupNode)
		return m.translateTreeSelection(cmd), true
	case key.Matches(msg, ReorderSessionUpKeyBinding):
		return m.reorderSelected(-1), true
	case key.Matches(msg, ReorderSessionDownKeyBinding):
		return m.reorderSelected(1), true
	case key.Matches(msg, DeleteSessionKeyBinding):
		if cmd := m.requestDeleteSelected(); cmd != nil {
			return cmd, true
		}
		return nil, true
	case key.Matches(msg, RenameKeyBinding):
		if cmd := m.requestRenameSelected(); cmd != nil {
			return cmd, true
		}
		return nil, true
	case key.Matches(msg, CycleLabelKeyBinding):
		if cmd := m.cycleLabelSelected(); cmd != nil {
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

func (m Model) requestRenameSelected() tea.Cmd {
	cur, ok := m.tree.Selected()
	if !ok {
		return nil
	}
	switch cur.kind {
	case sessionNodeSession:
		sess, ok := m.findSession(cur.sessionID)
		if !ok {
			return nil
		}
		return shared.Emit(shared.SessionRenameRequestedMsg{Session: sess})
	case sessionNodeGroup:
		if cur.projectID == uuid.Nil {
			return nil
		}
		return shared.Emit(shared.ProjectRenameRequestedMsg{
			ProjectID:   cur.projectID,
			CurrentName: cur.label,
		})
	}
	return nil
}

func (m Model) cycleLabelSelected() tea.Cmd {
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return nil
	}
	sessID, err := uuid.Parse(cur.sessionID)
	if err != nil {
		return nil
	}
	svc := m.service
	labels := m.labels
	focusID := cur.sessionID
	return func() tea.Msg {
		if _, err := svc.CycleLabel(context.Background(), service.CycleSessionLabelRequest{
			ID:     sessID,
			Labels: labels,
		}); err != nil {
			return shared.SessionLabelCycledMsg{Err: err}
		}
		listResp, listErr := svc.List(context.Background(), service.ListSessionsRequest{})
		return shared.SessionLabelCycledMsg{
			Sessions: listResp.Sessions,
			FocusID:  focusID,
			Err:      listErr,
		}
	}
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
	if ok && cur.kind == sessionNodeSession {
		if sess, ok := m.findSession(cur.sessionID); ok {
			return shared.Emit(shared.SessionSelectedMsg{Session: sess})
		}
	}
	return shared.Emit(shared.SessionSelectionClearedMsg{})
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
	innerW, innerH := components.TitledPanelInnerSize(m.styles, m.focused, width, height)
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

// SelectedSession returns the session the cursor is currently on, or
// (zero, false) when the cursor is on a group node or the list is empty.
func (m Model) SelectedSession() (domain.Session, bool) {
	cur, ok := m.tree.Selected()
	if !ok || cur.kind != sessionNodeSession {
		return domain.Session{}, false
	}
	return m.findSession(cur.sessionID)
}

func (m Model) View() tea.View {
	content := m.tree.View()
	innerW, innerH := components.TitledPanelInnerSize(m.styles, m.focused, m.width, m.height)
	if m.err != nil {
		content = components.CenteredContent(m.styles, m.styles.EmptyState.Title.Render("Unable to load sessions"), innerW, innerH)
	} else if content == "" {
		content = components.CenteredContent(m.styles, strings.Join([]string{
			m.styles.EmptyState.Title.Render("No sessions"),
			m.styles.EmptyState.Hint.Render("Press n to create one"),
		}, "\n"), innerW, innerH)
	}
	return components.PanelWithTitle(m.styles, content, agentStatusTitle(m.agentStatuses, m.styles.Glyphs), m.focused, m.width, m.height)
}

var agentStatusTitleOrder = []domain.AgentStatusKind{
	domain.AgentStatusRunning,
	domain.AgentStatusWaiting,
	domain.AgentStatusDead,
	domain.AgentStatusIdle,
	domain.AgentStatusUnknown,
}

func agentStatusLabel(kind domain.AgentStatusKind) string {
	switch kind {
	case domain.AgentStatusRunning:
		return "running"
	case domain.AgentStatusWaiting:
		return "waiting"
	case domain.AgentStatusIdle:
		return "idle"
	case domain.AgentStatusDead:
		return "dead"
	default:
		return "unknown"
	}
}

func agentStatusTitle(statuses map[uuid.UUID]domain.AgentStatus, glyphs styles.Glyphs) string {
	if len(statuses) == 0 {
		return "Sessions"
	}
	counts := map[domain.AgentStatusKind]int{}
	for _, st := range statuses {
		counts[st.Kind]++
	}
	segments := make([]string, 0, len(agentStatusTitleOrder))
	for _, kind := range agentStatusTitleOrder {
		n := counts[kind]
		if n == 0 {
			continue
		}
		segments = append(segments, glyphs.AgentStatus(kind)+" "+strconv.Itoa(n)+" "+agentStatusLabel(kind))
	}
	if len(segments) == 0 {
		return "Sessions"
	}
	return "Sessions (" + strings.Join(segments, " ") + ")"
}

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.List(context.Background(), service.ListSessionsRequest{})
		return shared.SessionsLoadedMsg{Sessions: result.Sessions, Err: err}
	}
}

func (m Model) sessionTreeNodes() []components.TreeNode[sessionNode] {
	if m.groupingMode == sessionGroupingNone {
		return rawSessionNodes(m.sessions, m.styles.Glyphs, m.agentStatuses)
	}
	return projectSessionNodes(m.sessions, m.projectNames, m.styles.Glyphs, m.agentStatuses)
}

func rawSessionNodes(sessions []domain.Session, glyphs styles.Glyphs, statuses map[uuid.UUID]domain.AgentStatus) []components.TreeNode[sessionNode] {
	nodes := make([]components.TreeNode[sessionNode], len(sessions))
	for i, sess := range sessions {
		nodes[i] = sessionTreeNode(sess, glyphs, statusKindFor(sess.ID, statuses))
	}
	return nodes
}

func projectSessionNodes(sessions []domain.Session, projectNames map[uuid.UUID]string, glyphs styles.Glyphs, statuses map[uuid.UUID]domain.AgentStatus) []components.TreeNode[sessionNode] {
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
			children[j] = sessionTreeNode(sess, glyphs, statusKindFor(sess.ID, statuses))
		}
		label := projectLabel(id, projectNames)
		nodes[i] = components.TreeNode[sessionNode]{
			ID:       "project:" + id.String(),
			Item:     sessionNode{kind: sessionNodeGroup, projectID: id, label: label},
			Children: children,
		}
	}
	return nodes
}

func statusKindFor(id uuid.UUID, statuses map[uuid.UUID]domain.AgentStatus) domain.AgentStatusKind {
	if st, ok := statuses[id]; ok {
		return st.Kind
	}
	return domain.AgentStatusUnknown
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

func sessionTreeNode(sess domain.Session, glyphs styles.Glyphs, agentStatus domain.AgentStatusKind) components.TreeNode[sessionNode] {
	label := sess.Name
	if !sess.HasWorktree() {
		label = glyphs.ProjectMode + " " + sess.Name
	}
	return components.TreeNode[sessionNode]{
		ID: "session:" + sess.ID.String(),
		Item: sessionNode{
			kind:        sessionNodeSession,
			sessionID:   sess.ID.String(),
			label:       label,
			statusCode:  sess.Label,
			updatedAt:   sess.UpdatedAt,
			agentStatus: agentStatus,
		},
	}
}

const labelBadgeGap = 1

func renderSessionNode(s *styles.Styles, labels []domain.Label) components.TreeRenderFunc[sessionNode] {
	return func(item sessionNode, _, depth, width int, hasKids, expanded, focused bool) string {
		prefix := components.TreePrefix(depth, hasKids, expanded)
		if item.kind == sessionNodeGroup {
			style := s.Group.Header
			if focused {
				style = s.Group.HeaderSelected
			}
			return style.Render(prefix + item.label)
		}

		statusGlyph := s.Glyphs.AgentStatus(item.agentStatus)
		body := prefix + statusGlyph + " " + item.label
		badge := renderLabelBadge(s, item.statusCode, labels)
		aux := shared.FormatRelativeDuration(time.Since(item.updatedAt))

		bodyW := lipgloss.Width(body)
		badgeW := lipgloss.Width(badge)
		auxW := lipgloss.Width(aux)

		badgeGap := 0
		if badgeW > 0 {
			badgeGap = labelBadgeGap
		}
		auxGap := labelBadgeGap

		filler := width - bodyW - badgeGap - badgeW - auxGap - auxW
		if filler < 0 && badgeW > 0 {
			filler = width - bodyW - auxGap - auxW
			badge, badgeW, badgeGap = "", 0, 0
		}
		if filler < 0 {
			if focused {
				return s.ListRow.Selected.Render(body)
			}
			return statusRowStyle(s).Render(body)
		}

		fillerSpaces := strings.Repeat(" ", filler)
		badgeGapSpaces := strings.Repeat(" ", badgeGap)
		auxGapSpaces := strings.Repeat(" ", auxGap)

		if focused {
			left := s.ListRow.Selected.Render(body + fillerSpaces + badgeGapSpaces + badge)
			right := s.ListRow.AuxSelected.Render(auxGapSpaces + aux)
			return left + right
		}
		rowStyle := statusRowStyle(s)
		return rowStyle.Render(body) + fillerSpaces + badgeGapSpaces + badge + auxGapSpaces + s.ListRow.Aux.Render(aux)
	}
}

func statusRowStyle(s *styles.Styles) lipgloss.Style {
	return s.ListRow.Normal
}

var staleLabelColor = lipgloss.Color("#9CA3AF")

func renderLabelBadge(s *styles.Styles, code string, labels []domain.Label) string {
	if code == "" {
		return ""
	}
	color := staleLabelColor
	text := code
	if l, ok := domain.FindLabel(code, labels); ok {
		color = lipgloss.Color(l.Color)
		glyph := l.Glyph
		if glyph == "" {
			glyph = s.Glyphs.LabelGlyph(code)
		}
		if glyph != "" {
			text = glyph + " " + code
		}
	}
	return s.SessionLabel.Foreground(color).Render(text)
}
