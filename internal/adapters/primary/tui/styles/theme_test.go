package styles_test

import (
	"image/color"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestDarkTheme_NoZeroColors(t *testing.T) {
	theme := styles.DarkTheme()
	fields := []struct {
		name  string
		color color.Color
	}{
		{"Primary", theme.Primary},
		{"Accent", theme.Accent},
		{"Warning", theme.Warning},
		{"Muted", theme.Muted},
		{"Text", theme.Text},
		{"Subtext", theme.Subtext},
		{"Border", theme.Border},
		{"BorderFocus", theme.BorderFocus},
		{"SelectionBg", theme.SelectionBg},
		{"TitleText", theme.TitleText},
		{"TitleSubtext", theme.TitleSubtext},
		{"HelpBg", theme.HelpBg},
		{"HelpKeyBg", theme.HelpKeyBg},
		{"ModalBg", theme.ModalBg},
		{"OverlayBg", theme.OverlayBg},
	}
	if len(fields) != 15 {
		t.Fatalf("expected 15 color fields, got %d", len(fields))
	}
	for _, f := range fields {
		if f.color == nil {
			t.Errorf("Theme.%s is nil", f.name)
		}
	}
}

func TestLoadTheme_DefaultsToDark(t *testing.T) {
	got := styles.LoadTheme("unknown-theme")
	want := styles.DarkTheme()
	if got != want {
		t.Errorf("LoadTheme(\"unknown-theme\") did not fall back to DarkTheme()\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestLoadTheme_Dark(t *testing.T) {
	got := styles.LoadTheme("dark")
	want := styles.DarkTheme()
	if got != want {
		t.Errorf("LoadTheme(\"dark\") != DarkTheme()\ngot:  %+v\nwant: %+v", got, want)
	}
}
