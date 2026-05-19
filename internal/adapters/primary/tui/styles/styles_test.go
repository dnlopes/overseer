package styles_test

import (
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestNew_ReturnsNonNil(t *testing.T) {
	if styles.New() == nil {
		t.Fatal("New() returned nil")
	}
}

func TestBorderStyles_FocusedDiffersFromBlurred(t *testing.T) {
	s := styles.New()
	focused := s.Border.Focused.Render("x")
	blurred := s.Border.Blurred.Render("x")
	if focused == blurred {
		t.Errorf("Border.Focused and Border.Blurred produce identical output: %q", focused)
	}
}

// TestNew_NoHiddenBorder checks that Border.Blurred uses RoundedBorder (╭ present in output),
// not HiddenBorder — invisible whitespace that makes panes disappear completely.
func TestNew_NoHiddenBorder(t *testing.T) {
	s := styles.New()
	blurredOutput := s.Border.Blurred.Render("x")
	if !strings.Contains(blurredOutput, "╭") {
		t.Errorf("Border.Blurred must use RoundedBorder (output must contain '╭'), got: %q", blurredOutput)
	}
}

func TestAllStyles_NonEmptyRender(t *testing.T) {
	s := styles.New()
	cases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"Border.Focused", s.Border.Focused},
		{"Border.Blurred", s.Border.Blurred},
		{"TitleBar.Base", s.TitleBar.Base},
		{"TitleBar.Branding", s.TitleBar.Branding},
		{"TitleBar.Subtext", s.TitleBar.Subtext},
		{"Pane.Sessions", s.Pane.Sessions},
		{"Pane.Status", s.Pane.Status},
		{"Pane.Preview", s.Pane.Preview},
		{"ListRow.Normal", s.ListRow.Normal},
		{"ListRow.Selected", s.ListRow.Selected},
		{"Group.Header", s.Group.Header},
		{"Status.Label", s.Status.Label},
		{"Status.Value", s.Status.Value},
		{"Status.Separator", s.Status.Separator},
		{"StatusSegment.Default", s.StatusSegment.Default},
		{"StatusSegment.Highlight", s.StatusSegment.Highlight},
		{"Form.Field.Label", s.Form.Field.Label},
		{"Form.Field.LabelFocused", s.Form.Field.LabelFocused},
		{"Form.Field.Input", s.Form.Field.Input},
		{"Form.Field.Error", s.Form.Field.Error},
		{"Modal.Box", s.Modal.Box},
		{"Badge.Key", s.Badge.Key},
		{"Badge.Label", s.Badge.Label},
		{"Divider.Horizontal", s.Divider.Horizontal},
		{"Help.Key", s.Help.Key},
		{"Help.Description", s.Help.Description},
		{"Help.Separator", s.Help.Separator},
		{"EmptyState.Title", s.EmptyState.Title},
		{"EmptyState.Hint", s.EmptyState.Hint},
		{"TooSmall.Message", s.TooSmall.Message},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.style.Render("x")
			if out == "" {
				t.Errorf("%s: Render(\"x\") returned empty string", tc.name)
			}
		})
	}
}
