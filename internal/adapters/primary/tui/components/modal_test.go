package components_test

import (
	"strings"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestModal_CentersContent(t *testing.T) {
	s := styles.New()
	body := "hello modal"
	out := components.Modal(s, body, 80, 24)

	if !strings.Contains(out, body) {
		t.Errorf("Modal output missing body text %q, got: %q", body, out)
	}
}

func TestModal_HasRoundedBorder(t *testing.T) {
	s := styles.New()
	out := components.Modal(s, "content", 80, 24)

	roundedChars := "╭╮╰╯"
	found := false
	for _, ch := range roundedChars {
		if strings.ContainsRune(out, ch) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Modal output missing rounded border chars (╭╮╰╯), got: %q", out)
	}
}

func TestModal_OverlayBackground(t *testing.T) {
	s := styles.New()
	out := components.Modal(s, "content", 80, 24)

	if !strings.Contains(out, "\x1b[") {
		t.Errorf("Modal output missing ANSI escape sequences (expected overlay background color), got: %q", out)
	}
}
