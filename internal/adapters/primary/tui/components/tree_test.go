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

func newTestTree() components.TreeModel[string] {
	tree := components.NewTree(func(item string, _, depth int, hasKids, expanded, _ bool) string {
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
