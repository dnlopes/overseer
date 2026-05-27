package styles

import "charm.land/lipgloss/v2"

func SolarizedDarkTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#268BD2"),
		Accent:          lipgloss.Color("#859900"),
		Warning:         lipgloss.Color("#B58900"),
		Danger:          lipgloss.Color("#DC322F"),
		Muted:           lipgloss.Color("#586E75"),
		Text:            lipgloss.Color("#93A1A1"),
		Subtext:         lipgloss.Color("#839496"),
		Border:          lipgloss.Color("#073642"),
		BorderFocus:     lipgloss.Color("#268BD2"),
		SelectionBg:     lipgloss.Color("#073642"),
		TitleText:       lipgloss.Color("#FDF6E3"),
		TitleSubtext:    lipgloss.Color("#EEE8D5"),
		HelpBg:          lipgloss.Color("#001F26"),
		HelpBarBg:       lipgloss.Color("#073642"),
		HelpKeyBg:       lipgloss.Color("#586E75"),
		ModalBg:         lipgloss.Color("#073642"),
		OverlayBg:       lipgloss.Color("#001F26"),
		StatusRunningFg: lipgloss.Color("#859900"),
		StatusRunningBg: lipgloss.Color("#0F2A12"),
		StatusWaitingFg: lipgloss.Color("#B58900"),
		StatusWaitingBg: lipgloss.Color("#2A2105"),
		StatusIdleFg:    lipgloss.Color("#586E75"),
		StatusDeadFg:    lipgloss.Color("#DC322F"),
		StatusDeadBg:    lipgloss.Color("#2A1010"),
		StatusUnknownFg: lipgloss.Color("#839496"),
	}
}
