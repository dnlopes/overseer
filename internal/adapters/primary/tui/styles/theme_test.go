package styles_test

import (
	"image/color"
	"testing"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func TestThemes_NoZeroColors(t *testing.T) {
	themes := []struct {
		name  string
		theme styles.Theme
	}{
		{"DarkTheme", styles.DarkTheme()},
		{"DraculaTheme", styles.DraculaTheme()},
		{"GitHubDarkTheme", styles.GitHubDarkTheme()},
		{"TokyoNightTheme", styles.TokyoNightTheme()},
		{"MonokaiTheme", styles.MonokaiTheme()},
		{"OneDarkTheme", styles.OneDarkTheme()},
		{"SolarizedDarkTheme", styles.SolarizedDarkTheme()},
		{"NordTheme", styles.NordTheme()},
		{"CatppuccinMochaTheme", styles.CatppuccinMochaTheme()},
		{"PorcelainTheme", styles.PorcelainTheme()},
		{"DeepSeaTheme", styles.DeepSeaTheme()},
		{"SunsetTheme", styles.SunsetTheme()},
	}
	for _, tt := range themes {
		t.Run(tt.name, func(t *testing.T) {
			assertAllColorsSet(t, tt.theme)
		})
	}
}

func TestLoadTheme_UnknownTheme_ReturnsDark(t *testing.T) {
	got := styles.LoadTheme("unknown-theme")
	want := styles.DarkTheme()
	if got != want {
		t.Errorf("LoadTheme(\"unknown-theme\") did not fall back to DarkTheme()\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestLoadTheme_KnownTheme_ReturnsRegisteredTheme(t *testing.T) {
	tests := []struct {
		key  string
		want styles.Theme
	}{
		{"dark", styles.DarkTheme()},
		{"dracula", styles.DraculaTheme()},
		{"github-dark", styles.GitHubDarkTheme()},
		{"tokyo-night", styles.TokyoNightTheme()},
		{"monokai", styles.MonokaiTheme()},
		{"one-dark", styles.OneDarkTheme()},
		{"solarized-dark", styles.SolarizedDarkTheme()},
		{"nord", styles.NordTheme()},
		{"catppuccin-mocha", styles.CatppuccinMochaTheme()},
		{"porcelain", styles.PorcelainTheme()},
		{"deep-sea", styles.DeepSeaTheme()},
		{"sunset", styles.SunsetTheme()},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := styles.LoadTheme(tt.key)
			if got != tt.want {
				t.Errorf("LoadTheme(%q) returned wrong theme\ngot:  %+v\nwant: %+v", tt.key, got, tt.want)
			}
		})
	}
}

func assertAllColorsSet(t *testing.T, theme styles.Theme) {
	t.Helper()
	fields := []struct {
		name  string
		color color.Color
	}{
		{"Primary", theme.Primary},
		{"Accent", theme.Accent},
		{"Warning", theme.Warning},
		{"Danger", theme.Danger},
		{"Muted", theme.Muted},
		{"Text", theme.Text},
		{"Subtext", theme.Subtext},
		{"Border", theme.Border},
		{"BorderFocus", theme.BorderFocus},
		{"SelectionBg", theme.SelectionBg},
		{"TitleText", theme.TitleText},
		{"TitleSubtext", theme.TitleSubtext},
		{"HelpBg", theme.HelpBg},
		{"HelpBarBg", theme.HelpBarBg},
		{"HelpKeyBg", theme.HelpKeyBg},
		{"ModalBg", theme.ModalBg},
		{"OverlayBg", theme.OverlayBg},
		{"StatusRunningFg", theme.StatusRunningFg},
		{"StatusRunningBg", theme.StatusRunningBg},
		{"StatusWaitingFg", theme.StatusWaitingFg},
		{"StatusWaitingBg", theme.StatusWaitingBg},
		{"StatusIdleFg", theme.StatusIdleFg},
		{"StatusDeadFg", theme.StatusDeadFg},
		{"StatusDeadBg", theme.StatusDeadBg},
		{"StatusUnknownFg", theme.StatusUnknownFg},
	}
	if len(fields) != 25 {
		t.Fatalf("expected 25 color fields, got %d", len(fields))
	}
	for _, f := range fields {
		if f.color == nil {
			t.Errorf("Theme.%s is nil", f.name)
		}
	}
}
