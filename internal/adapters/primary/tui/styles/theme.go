package styles

import (
	"image/color"
)

type Theme struct {
	Primary     color.Color
	Accent      color.Color
	Warning     color.Color
	Muted       color.Color
	Text        color.Color
	Subtext     color.Color
	Border      color.Color
	BorderFocus color.Color
	SelectionBg color.Color
	TitleText   color.Color
	TitleSubtext color.Color
	HelpBg      color.Color
	HelpKeyBg   color.Color
	ModalBg     color.Color
	OverlayBg   color.Color
}

func LoadTheme(name string) Theme {
	switch name {
	case "dark":
		return DarkTheme()
	default:
		return DarkTheme()
	}
}
