package styles_test

import (
	"flag"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func init() {
	if flag.Lookup("update") == nil {
		flag.Bool("update", false, "update .golden files")
	}
}

func TestColorGolden_TitleBarBranding(t *testing.T) {
	s := styles.New()
	out := s.TitleBar.Branding.Render("overseer")
	testutil.RequireEqualColor(t, "titlebar-branding", out)
}

func TestColorGolden_ListRowSelected(t *testing.T) {
	s := styles.New()
	out := s.ListRow.Selected.Render("my-session")
	testutil.RequireEqualColor(t, "list-row-selected", out)
}

func TestColorGolden_StatusSegmentHighlight(t *testing.T) {
	s := styles.New()
	out := s.StatusSegment.Highlight.Render("idle")
	testutil.RequireEqualColor(t, "status-segment-highlight", out)
}

func TestColorGolden_HelpKey(t *testing.T) {
	s := styles.New()
	out := s.Help.Key.Render("q")
	testutil.RequireEqualColor(t, "help-key", out)
}
