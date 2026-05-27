package styles

import "charm.land/lipgloss/v2"

func CatppuccinMochaTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#CBA6F7"),
		Accent:          lipgloss.Color("#A6E3A1"),
		Warning:         lipgloss.Color("#F9E2AF"),
		Danger:          lipgloss.Color("#F38BA8"),
		Muted:           lipgloss.Color("#6C7086"),
		Text:            lipgloss.Color("#CDD6F4"),
		Subtext:         lipgloss.Color("#A6ADC8"),
		Border:          lipgloss.Color("#45475A"),
		BorderFocus:     lipgloss.Color("#CBA6F7"),
		SelectionBg:     lipgloss.Color("#45475A"),
		TitleText:       lipgloss.Color("#1E1E2E"),
		TitleSubtext:    lipgloss.Color("#313244"),
		HelpBg:          lipgloss.Color("#181825"),
		HelpBarBg:       lipgloss.Color("#313244"),
		HelpKeyBg:       lipgloss.Color("#45475A"),
		ModalBg:         lipgloss.Color("#313244"),
		OverlayBg:       lipgloss.Color("#11111B"),
		StatusRunningFg: lipgloss.Color("#A6E3A1"),
		StatusRunningBg: lipgloss.Color("#1F2E1F"),
		StatusWaitingFg: lipgloss.Color("#F9E2AF"),
		StatusWaitingBg: lipgloss.Color("#2E281A"),
		StatusIdleFg:    lipgloss.Color("#6C7086"),
		StatusDeadFg:    lipgloss.Color("#F38BA8"),
		StatusDeadBg:    lipgloss.Color("#2E1A22"),
		StatusUnknownFg: lipgloss.Color("#A6ADC8"),
	}
}
