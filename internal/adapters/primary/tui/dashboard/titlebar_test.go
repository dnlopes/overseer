package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func TestTitlebar_RendersBranding(t *testing.T) {
	s := styles.New()
	m := newTitlebar(s, "overseer")

	output := testutil.StripANSI(m.View().Content)

	if !strings.Contains(output, "overseer") {
		t.Errorf("branding not found in output: %q", output)
	}
}

func TestTitlebar_RendersActivePane(t *testing.T) {
	s := styles.New()
	m := newTitlebar(s, "overseer")

	updated, _ := m.Update(TitlebarSetActivePaneMsg{Label: "sessions"})
	m = updated.(TitlebarModel)

	output := testutil.StripANSI(m.View().Content)

	if !strings.Contains(output, "sessions") {
		t.Errorf("active pane label not found in output: %q", output)
	}
}

func TestTitlebar_WidthHonored(t *testing.T) {
	s := styles.New()
	m := newTitlebar(s, "overseer")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(TitlebarModel)

	got := lipgloss.Width(m.View().Content)
	if got != 80 {
		t.Errorf("expected rendered width 80, got %d", got)
	}
}
