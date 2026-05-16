package components_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestHorizontalDivider_CorrectWidth(t *testing.T) {
	s := styles.New()
	width := 10
	out := components.HorizontalDivider(s, width)
	stripped := ansi.Strip(out)
	count := strings.Count(stripped, "─")
	if count != width {
		t.Errorf("HorizontalDivider(%d) produced %d '─' chars, want %d; raw=%q stripped=%q",
			width, count, width, out, stripped)
	}
}

func TestHorizontalDivider_ZeroWidth(t *testing.T) {
	s := styles.New()
	out := components.HorizontalDivider(s, 0)
	if out != "" {
		t.Errorf("HorizontalDivider(0) = %q, want empty string", out)
	}
}

func TestHorizontalDivider_NegativeWidth(t *testing.T) {
	s := styles.New()
	out := components.HorizontalDivider(s, -1)
	if out != "" {
		t.Errorf("HorizontalDivider(-1) = %q, want empty string", out)
	}
}
