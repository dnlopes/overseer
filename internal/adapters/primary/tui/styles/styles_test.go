package styles_test

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
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

func TestAllStyles_NonEmptyRender(t *testing.T) {
	s := styles.New()
	cases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"Border.Focused", s.Border.Focused},
		{"Border.Blurred", s.Border.Blurred},
		{"Pane.Sessions", s.Pane.Sessions},
		{"Pane.Status", s.Pane.Status},
		{"Pane.Preview", s.Pane.Preview},
		{"Group.Header", s.Group.Header},
		{"Session.Item.Normal", s.Session.Item.Normal},
		{"Session.Item.Selected", s.Session.Item.Selected},
		{"Status.Label", s.Status.Label},
		{"Status.Value", s.Status.Value},
		{"Status.Separator", s.Status.Separator},
		{"Form.Field.Label", s.Form.Field.Label},
		{"Form.Field.Input", s.Form.Field.Input},
		{"Form.Field.Error", s.Form.Field.Error},
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
