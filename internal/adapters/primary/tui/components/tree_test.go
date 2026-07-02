package components_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
)

func TestTreeModel_RendersExpandedTree(t *testing.T) {
	tree := newTestTree().ExpandAll()

	want := "▾ project\n  · alpha\n  · beta"
	if got := tree.View(); got != want {
		t.Fatalf("View() = %q, want %q", got, want)
	}
}

func TestTreeModel_ToggleCollapsesFocusedNode(t *testing.T) {
	tree := newTestTree().ExpandAll().Focus()

	updated, _ := tree.Update(keyPress("enter"))

	want := "▸ project"
	if got := updated.View(); got != want {
		t.Fatalf("View() after toggle = %q, want %q", got, want)
	}
}

func TestTreeModel_SelectIDMovesCursor(t *testing.T) {
	tree := newTestTree().ExpandAll().SelectID("session-beta")

	if got := tree.SelectedID(); got != "session-beta" {
		t.Fatalf("SelectedID() = %q, want %q", got, "session-beta")
	}
}

func TestTreeModel_NavigationEmitsSelectedNode(t *testing.T) {
	tree := newTestTree().ExpandAll().Focus()

	updated, cmd := tree.Update(keyPress("j"))
	msg, ok := cmd().(components.TreeSelectMsg[string])
	if !ok {
		t.Fatalf("selection msg type = %T, want components.TreeSelectMsg[string]", cmd())
	}
	if msg.ID != "session-alpha" || msg.Item != "alpha" {
		t.Fatalf("selection msg = %+v, want session-alpha/alpha", msg)
	}
	if got := updated.SelectedID(); got != "session-alpha" {
		t.Fatalf("SelectedID() = %q, want %q", got, "session-alpha")
	}
}

func TestTreeModel_HeightLimitsVisibleRows(t *testing.T) {
	tree := newTestTree().ExpandAll().SetSize(20, 2).Focus()
	tree, _ = tree.Update(keyPress("j"))
	tree, _ = tree.Update(keyPress("j"))

	want := "  · alpha\n  · beta"
	if got := tree.View(); got != want {
		t.Fatalf("View() = %q, want %q", got, want)
	}
}

func TestTreeModel_MoveCursorClampsToLastRowWhenOverflowing(t *testing.T) {
	tree := newTestTree().ExpandAll().Focus()

	tree, cmd := tree.MoveCursor(99)

	if got := tree.SelectedID(); got != "session-beta" {
		t.Fatalf("SelectedID() = %q, want %q", got, "session-beta")
	}
	if cmd == nil {
		t.Fatalf("MoveCursor() cmd = nil, want selection emit")
	}
}

func TestTreeModel_MoveCursorClampsToFirstRowWhenUnderflowing(t *testing.T) {
	tree := newTestTree().ExpandAll().SelectID("session-beta").Focus()

	tree, cmd := tree.MoveCursor(-99)

	if got := tree.SelectedID(); got != "project" {
		t.Fatalf("SelectedID() = %q, want %q", got, "project")
	}
	if cmd == nil {
		t.Fatalf("MoveCursor() cmd = nil, want selection emit")
	}
}

func TestTreeModel_MoveCursorReturnsNilCmdWhenNoMovement(t *testing.T) {
	tree := newTestTree().ExpandAll().Focus()

	_, cmd := tree.MoveCursor(-1)

	if cmd != nil {
		t.Fatalf("MoveCursor(-1) at top: cmd = %#v, want nil", cmd)
	}
}

func TestTreeModel_MoveToNextFindsMatchingItem(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().Focus()

	tree, cmd := tree.MoveToNext(func(item string) bool { return item == "second" })

	if got := tree.SelectedID(); got != "group-second" {
		t.Fatalf("SelectedID() = %q, want %q", got, "group-second")
	}
	if cmd == nil {
		t.Fatalf("MoveToNext() cmd = nil, want selection emit")
	}
}

func TestTreeModel_MoveToNextNoMatchKeepsCursor(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().Focus()

	tree, cmd := tree.MoveToNext(func(item string) bool { return item == "nonexistent" })

	if got := tree.SelectedID(); got != "group-first" {
		t.Fatalf("SelectedID() = %q, want %q (unchanged)", got, "group-first")
	}
	if cmd != nil {
		t.Fatalf("MoveToNext() cmd = %#v, want nil when no match", cmd)
	}
}

func TestTreeModel_MoveToPrevFindsMatchingItem(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().SelectID("group-second").Focus()

	tree, cmd := tree.MoveToPrev(func(item string) bool { return item == "first" })

	if got := tree.SelectedID(); got != "group-first" {
		t.Fatalf("SelectedID() = %q, want %q", got, "group-first")
	}
	if cmd == nil {
		t.Fatalf("MoveToPrev() cmd = nil, want selection emit")
	}
}

func TestTreeModel_SelectNthFindsMatchingItem(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().Focus()

	tree, cmd := tree.SelectNth(func(item string) bool { return item == "second" }, 1)

	if got := tree.SelectedID(); got != "group-second" {
		t.Fatalf("SelectedID() = %q, want %q", got, "group-second")
	}
	if cmd == nil {
		t.Fatalf("SelectNth() cmd = nil, want selection emit")
	}
}

func TestTreeModel_SelectNthNoMatchKeepsCursor(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().Focus()

	tree, cmd := tree.SelectNth(func(item string) bool { return item == "nonexistent" }, 1)

	if got := tree.SelectedID(); got != "group-first" {
		t.Fatalf("SelectedID() = %q, want %q (unchanged)", got, "group-first")
	}
	if cmd != nil {
		t.Fatalf("SelectNth() cmd = %#v, want nil when no match", cmd)
	}
}

func TestTreeModel_SelectNthSameCursorReturnsNilCmd(t *testing.T) {
	tree := newTreeWithTwoGroups().ExpandAll().Focus()

	_, cmd := tree.SelectNth(func(item string) bool { return item == "first" }, 1)

	if cmd != nil {
		t.Fatalf("SelectNth() to current cursor cmd = %#v, want nil", cmd)
	}
}

func newTreeWithTwoGroups() components.TreeModel[string] {
	tree := components.NewTree(func(item string, _, _, _ int, _, _, _ bool) string {
		return item
	})
	return tree.SetNodes([]components.TreeNode[string]{
		{
			ID:   "group-first",
			Item: "first",
			Children: []components.TreeNode[string]{
				{ID: "child-a", Item: "a"},
			},
		},
		{
			ID:   "group-second",
			Item: "second",
			Children: []components.TreeNode[string]{
				{ID: "child-b", Item: "b"},
			},
		},
	})
}

func newTestTree() components.TreeModel[string] {
	tree := components.NewTree(func(item string, _, depth, _ int, hasKids, expanded, _ bool) string {
		prefix := "· "
		if hasKids && expanded {
			prefix = "▾ "
		} else if hasKids {
			prefix = "▸ "
		}
		if depth > 0 {
			prefix = "  " + prefix
		}
		return prefix + item
	})
	return tree.SetNodes([]components.TreeNode[string]{
		{
			ID:   "project",
			Item: "project",
			Children: []components.TreeNode[string]{
				{ID: "session-alpha", Item: "alpha"},
				{ID: "session-beta", Item: "beta"},
			},
		},
	})
}

func keyPress(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: value, Code: []rune(value)[0]}
}
