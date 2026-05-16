package components_test

import (
	"strings"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestKeyBadge_RendersBothParts(t *testing.T) {
	s := styles.New()
	out := components.KeyBadge(s, "q", "quit")

	if !strings.Contains(out, "q") {
		t.Errorf("KeyBadge output missing key %q, got: %q", "q", out)
	}
	if !strings.Contains(out, "quit") {
		t.Errorf("KeyBadge output missing label %q, got: %q", "quit", out)
	}
}

func TestKeyBadge_KeyHasBackground(t *testing.T) {
	s := styles.New()
	out := components.KeyBadge(s, "q", "quit")

	// Badge.Key has Background(HelpKeyBg) — rendered output must contain ANSI escape sequences
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("KeyBadge output missing ANSI escape sequences (no background color applied), got: %q", out)
	}
}
