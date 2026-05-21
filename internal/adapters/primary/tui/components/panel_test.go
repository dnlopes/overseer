package components_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestPanel_FocusedRoundedBorder(t *testing.T) {
	s := styles.New()
	out := components.Panel(s, "hello", true)

	roundedChars := "╭╮╰╯"
	found := false
	for _, ch := range roundedChars {
		if strings.ContainsRune(out, ch) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Panel(focused=true) output missing rounded border chars (╭╮╰╯), got: %q", out)
	}
}

func TestPanel_BlurredRoundedBorder(t *testing.T) {
	s := styles.New()
	out := components.Panel(s, "hello", false)

	roundedChars := "╭╮╰╯"
	found := false
	for _, ch := range roundedChars {
		if strings.ContainsRune(out, ch) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Panel(focused=false) output missing rounded border chars (╭╮╰╯), got: %q", out)
	}
}

func TestPanel_ContentVisible(t *testing.T) {
	s := styles.New()
	content := "unique-content-xyz"
	out := components.Panel(s, content, true)

	if !strings.Contains(out, content) {
		t.Errorf("Panel output missing content %q, got: %q", content, out)
	}
}

func TestPanelWithSize_ClampsOverflowingContentToDeclaredHeight(t *testing.T) {
	s := styles.New()
	const width, height = 20, 6

	tallContent := strings.Repeat("row\n", height+5)
	out := components.PanelWithSize(s, tallContent, true, width, height).Content

	if got := lipgloss.Height(out); got != height {
		t.Errorf("PanelWithSize height = %d, want %d (panel must not grow past declared height even when content overflows)", got, height)
	}
}

func TestPanelWithSize_PreservesBottomBorderWhenContentOverflows(t *testing.T) {
	s := styles.New()
	const width, height = 20, 6

	tallContent := strings.Repeat("row\n", height+5)
	out := components.PanelWithSize(s, tallContent, true, width, height).Content

	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatalf("PanelWithSize produced no output")
	}
	last := lines[len(lines)-1]
	if !strings.ContainsRune(last, '╰') || !strings.ContainsRune(last, '╯') {
		t.Errorf("PanelWithSize last row missing bottom border corners (╰╯), got: %q\nfull output:\n%s", last, out)
	}
}

func TestPanelWithSize_TmuxLikeContentDoesNotOverflow(t *testing.T) {
	s := styles.New()
	const width, height = 20, 6

	containerOuterH := height - 2
	tmuxLike := strings.Repeat("line\n", containerOuterH)
	out := components.PanelWithSize(s, tmuxLike, true, width, height).Content

	if got := lipgloss.Height(out); got != height {
		t.Errorf("PanelWithSize height with tmux-style trailing-newline content = %d, want %d", got, height)
	}
}
