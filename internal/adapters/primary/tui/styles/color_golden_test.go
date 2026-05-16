package styles_test

import (
	"flag"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	golden "github.com/dnlopes/overseer/internal/testutil/golden"
)

func init() {
	if flag.Lookup("update") == nil {
		flag.Bool("update", false, "update .golden files")
	}
}

func TestColorGolden_TitleBarBranding(t *testing.T) {
	s := styles.New()
	out := s.TitleBar.Branding.Render("Overseer")
	golden.RequireEqualColor(t, "titlebar-branding", out)
}

func TestColorGolden_ListRowSelected(t *testing.T) {
	s := styles.New()
	out := s.ListRow.Selected.Render("my-session")
	golden.RequireEqualColor(t, "list-row-selected", out)
}

func TestColorGolden_StatusSegmentHighlight(t *testing.T) {
	s := styles.New()
	out := s.StatusSegment.Highlight.Render("idle")
	golden.RequireEqualColor(t, "status-segment-highlight", out)
}

func TestColorGolden_HelpKey(t *testing.T) {
	s := styles.New()
	out := s.Help.Key.Render("q")
	golden.RequireEqualColor(t, "help-key", out)
}
