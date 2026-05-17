package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	xgolden "github.com/charmbracelet/x/exp/golden"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func newSizedPreview(t *testing.T, width, height int) PreviewModel {
	t.Helper()
	m := newPreview(styles.New())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(PreviewModel)
}

func TestPreview_Default(t *testing.T) {
	m := newSizedPreview(t, 80, 20)
	out := testutil.StripANSI(m.View().Content)
	if !strings.Contains(out, "Stub mode: preview not available.") {
		t.Fatalf("expected stub message in view, got:\n%s", out)
	}
	xgolden.RequireEqual(t, []byte(out))
}

func TestPreview_FocusedBorder(t *testing.T) {
	m := newSizedPreview(t, 80, 20)

	blurredOut := m.View().Content

	m.SetFocus(true)
	focusedOut := m.View().Content

	if !strings.ContainsAny(blurredOut, "╭╮╰╯") {
		t.Error("blurred view should show rounded border (not hidden)")
	}
	if !strings.ContainsAny(focusedOut, "╭╮╰╯") {
		t.Error("focused view should show rounded border")
	}
}

func TestPreview_SetFocus(t *testing.T) {
	m := newPreview(styles.New())

	if m.Focused() {
		t.Fatal("new model should not be focused")
	}

	m.SetFocus(true)
	if !m.Focused() {
		t.Fatal("model should be focused after SetFocus(true)")
	}

	m.SetFocus(false)
	if m.Focused() {
		t.Fatal("model should not be focused after SetFocus(false)")
	}
}
