package preview_test

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	xgolden "github.com/charmbracelet/x/exp/golden"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/preview"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	internalgolden "github.com/dnlopes/overseer/internal/testutil/golden"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newSizedModel(t *testing.T, width, height int) preview.Model {
	t.Helper()
	m := preview.New(styles.New())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(preview.Model)
}

func TestPreview_Default(t *testing.T) {
	m := newSizedModel(t, 80, 20)
	out := internalgolden.StripANSI(m.View().Content)
	if !strings.Contains(out, "Stub mode: preview not available.") {
		t.Fatalf("expected stub message in view, got:\n%s", out)
	}
	xgolden.RequireEqual(t, []byte(out))
}

func TestPreview_FocusedBorder(t *testing.T) {
	m := newSizedModel(t, 80, 20)

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
	m := preview.New(styles.New())

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
