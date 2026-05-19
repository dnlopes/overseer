package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// TreePrefix returns the leading string for a tree row at the given depth,
// composed of (1) a depth-based indent using [styles.ListIndentUnit] and
// (2) a collapse indicator (▾/▸) when the row has children. Use this from
// any [TreeRenderFunc] that wants the standard tree-view look. Flat lists
// (depth 0, no children) get a 2-space gutter that keeps labels aligned
// with collapsible rows above/below.
func TreePrefix(depth int, hasKids, expanded bool) string {
	indicator := "  "
	if hasKids && expanded {
		indicator = "▾ "
	} else if hasKids {
		indicator = "▸ "
	}
	return strings.Repeat(" ", depth*styles.ListIndentUnit) + indicator
}

// TreeNode is what callers supply: a generic tree of items. The component
// doesn't care what T is - it only navigates, expands, and renders.
type TreeNode[T any] struct {
	ID       string
	Item     T
	Children []TreeNode[T]
}

// TreeRenderFunc lets callers control how each item is displayed.
// Receives the item, its zero-based row index (across visible rows),
// indentation depth, whether it has children, whether it's currently
// expanded, and whether the cursor is on it.
type TreeRenderFunc[T any] func(item T, index, depth int, hasKids, expanded, focused bool) string

// TreeSelectMsg is emitted whenever the cursor lands on a different node,
// either by navigation or after a refresh.
type TreeSelectMsg[T any] struct {
	ID   string
	Item T
}

type TreeKeyMap struct {
	Up, Down, Toggle, ExpandAll, CollapseAll key.Binding
}

func DefaultTreeKeyMap() TreeKeyMap {
	return TreeKeyMap{
		Up:          key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:        key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Toggle:      key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("⏎/space", "toggle")),
		ExpandAll:   key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "expand all")),
		CollapseAll: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "collapse all")),
	}
}

// ShortHelp/FullHelp satisfy bubbles/help.KeyMap.
func (k TreeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle}
}
func (k TreeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Toggle, k.ExpandAll, k.CollapseAll},
	}
}

// row is a flattened view of one visible item.
type row[T any] struct {
	id       string
	item     T
	depth    int
	hasKids  bool
	expanded bool
}

type TreeModel[T any] struct {
	nodes    []TreeNode[T]   // source of truth (caller-supplied)
	rows     []row[T]        // flattened, filtered view
	expanded map[string]bool // node ID → expanded?
	cursor   int             // index into rows[]
	focused  bool

	w, h   int
	keys   TreeKeyMap
	render TreeRenderFunc[T]
}

func NewTree[T any](render TreeRenderFunc[T]) TreeModel[T] {
	return TreeModel[T]{
		expanded: make(map[string]bool),
		keys:     DefaultTreeKeyMap(),
		render:   render,
	}
}

func (m TreeModel[T]) WithKeyMap(k TreeKeyMap) TreeModel[T] {
	m.keys = k
	return m
}

// SetNodes replaces the tree's source data and re-flattens. The cursor
// is preserved by ID when possible; if the previously-focused node is
// gone, the cursor stays at the same index (clamped to bounds).
func (m TreeModel[T]) SetNodes(nodes []TreeNode[T]) TreeModel[T] {
	var prevID string
	if len(m.rows) > 0 && m.cursor < len(m.rows) {
		prevID = m.rows[m.cursor].id
	}

	m.nodes = nodes
	m.rows = flatten(nodes, m.expanded)

	// Try to restore cursor by ID
	if prevID != "" {
		for i, r := range m.rows {
			if r.id == prevID {
				m.cursor = i
				return m
			}
		}
	}
	m.cursor = clamp(m.cursor, 0, len(m.rows)-1)
	return m
}

// ExpandAll/CollapseAll toggle expansion state across every group.
func (m TreeModel[T]) ExpandAll() TreeModel[T] {
	walk(m.nodes, func(n TreeNode[T]) {
		if len(n.Children) > 0 {
			m.expanded[n.ID] = true
		}
	})
	return m.reflatten()
}

func (m TreeModel[T]) CollapseAll() TreeModel[T] {
	m.expanded = make(map[string]bool)
	return m.reflatten()
}

// Selected returns the currently-focused item, or zero T and false if empty.
func (m TreeModel[T]) Selected() (T, bool) {
	var zero T
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return zero, false
	}
	return m.rows[m.cursor].item, true
}

// SelectedID returns the ID of the focused node, or "" if empty.
func (m TreeModel[T]) SelectedID() string {
	if len(m.rows) == 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return ""
	}
	return m.rows[m.cursor].id
}

func (m TreeModel[T]) SelectID(id string) TreeModel[T] {
	for i, r := range m.rows {
		if r.id == id {
			m.cursor = i
			return m
		}
	}
	return m
}

// SelectIndex moves the cursor to the row at the given zero-based index
// among currently visible rows. Out-of-range indices are clamped — values
// below 0 are ignored, values past the last row also ignored. Use
// [TreeModel.RowCount] to check bounds before calling.
func (m TreeModel[T]) SelectIndex(idx int) TreeModel[T] {
	if idx < 0 || idx >= len(m.rows) {
		return m
	}
	m.cursor = idx
	return m
}

// RowCount returns the number of currently visible (post-flatten) rows.
func (m TreeModel[T]) RowCount() int {
	return len(m.rows)
}

// EmitSelection returns a tea.Cmd that emits a [TreeSelectMsg] for the
// currently-selected row. Use this when changing the cursor from outside
// the standard key handlers (e.g., number-prefix jump in the projects tab).
func (m TreeModel[T]) EmitSelection() tea.Cmd {
	return m.emitSelection()
}

func (m TreeModel[T]) Init() tea.Cmd { return nil }

func (m TreeModel[T]) Update(msg tea.Msg) (TreeModel[T], tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			return m, m.emitSelection()
		}
	case key.Matches(keyMsg, m.keys.Down):
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			return m, m.emitSelection()
		}
	case key.Matches(keyMsg, m.keys.Toggle):
		return m.toggleCurrent(), nil
	case key.Matches(keyMsg, m.keys.ExpandAll):
		return m.ExpandAll(), nil
	case key.Matches(keyMsg, m.keys.CollapseAll):
		return m.CollapseAll(), nil
	}
	return m, nil
}

func (m TreeModel[T]) View() string {
	if len(m.rows) == 0 {
		return ""
	}
	var b strings.Builder
	visible := m.visibleWindow()
	for i := visible.top; i <= visible.bottom; i++ {
		r := m.rows[i]
		b.WriteString(m.render(r.item, i, r.depth, r.hasKids, r.expanded, i == m.cursor))
		if i < visible.bottom {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m TreeModel[T]) SetSize(w, h int) TreeModel[T] {
	m.w, m.h = w, h
	return m
}

func (m TreeModel[T]) Focus() TreeModel[T] {
	m.focused = true
	return m
}

func (m TreeModel[T]) Blur() TreeModel[T] {
	m.focused = false
	return m
}

func (m TreeModel[T]) KeyMap() TreeKeyMap { return m.keys }

func (m TreeModel[T]) toggleCurrent() TreeModel[T] {
	if len(m.rows) == 0 {
		return m
	}
	cur := m.rows[m.cursor]
	if !cur.hasKids {
		return m
	}
	m.expanded[cur.id] = !m.expanded[cur.id]
	return m.reflatten()
}

func (m TreeModel[T]) reflatten() TreeModel[T] {
	prevID := m.SelectedID()
	m.rows = flatten(m.nodes, m.expanded)
	if prevID != "" {
		for i, r := range m.rows {
			if r.id == prevID {
				m.cursor = i
				return m
			}
		}
	}
	m.cursor = clamp(m.cursor, 0, len(m.rows)-1)
	return m
}

func (m TreeModel[T]) emitSelection() tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	cur := m.rows[m.cursor]
	return func() tea.Msg { return TreeSelectMsg[T]{ID: cur.id, Item: cur.item} }
}

// visibleWindow computes the slice of rows currently in view, keeping
// the cursor visible. Simple scroll behavior - the cursor sits roughly
// in the middle when scrolling, at the edges when near the top/bottom.
type window struct{ top, bottom int }

func (m TreeModel[T]) visibleWindow() window {
	if m.h <= 0 || len(m.rows) <= m.h {
		return window{top: 0, bottom: len(m.rows) - 1}
	}
	// Keep cursor in view, scrolling as needed
	half := m.h / 2
	top := m.cursor - half
	top = max(top, 0)
	bottom := top + m.h - 1
	if bottom >= len(m.rows) {
		bottom = len(m.rows) - 1
		top = bottom - m.h + 1
	}
	return window{top: top, bottom: bottom}
}

func flatten[T any](nodes []TreeNode[T], expanded map[string]bool) []row[T] {
	var rows []row[T]
	var walk func(n TreeNode[T], depth int)
	walk = func(n TreeNode[T], depth int) {
		r := row[T]{
			id:       n.ID,
			item:     n.Item,
			depth:    depth,
			hasKids:  len(n.Children) > 0,
			expanded: expanded[n.ID],
		}
		rows = append(rows, r)
		if r.hasKids && r.expanded {
			for _, c := range n.Children {
				walk(c, depth+1)
			}
		}
	}
	for _, root := range nodes {
		walk(root, 0)
	}
	return rows
}

func walk[T any](nodes []TreeNode[T], fn func(TreeNode[T])) {
	for _, n := range nodes {
		fn(n)
		walk(n.Children, fn)
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
