package styles

import (
	"image/color"
)

type Theme struct {
	Primary         color.Color
	Accent          color.Color
	Warning         color.Color
	Danger          color.Color
	Muted           color.Color
	Text            color.Color
	Subtext         color.Color
	Border          color.Color
	BorderFocus     color.Color
	SelectionBg     color.Color
	TitleText       color.Color
	TitleSubtext    color.Color
	HelpBg          color.Color
	HelpBarBg       color.Color
	HelpKeyBg       color.Color
	ModalBg         color.Color
	OverlayBg       color.Color
	StatusRunningFg color.Color
	StatusRunningBg color.Color
	StatusWaitingFg color.Color
	StatusWaitingBg color.Color
	StatusIdleFg    color.Color
	StatusDeadFg    color.Color
	StatusDeadBg    color.Color
	StatusUnknownFg color.Color
}

func LoadTheme(name string) Theme {
	switch name {
	case "dark":
		return DarkTheme()
	case "dracula":
		return DraculaTheme()
	case "github-dark":
		return GitHubDarkTheme()
	case "tokyo-night":
		return TokyoNightTheme()
	case "monokai":
		return MonokaiTheme()
	case "one-dark":
		return OneDarkTheme()
	case "solarized-dark":
		return SolarizedDarkTheme()
	case "nord":
		return NordTheme()
	case "catppuccin-mocha":
		return CatppuccinMochaTheme()
	case "porcelain":
		return PorcelainTheme()
	case "deep-sea":
		return DeepSeaTheme()
	case "sunset":
		return SunsetTheme()
	default:
		return DarkTheme()
	}
}
