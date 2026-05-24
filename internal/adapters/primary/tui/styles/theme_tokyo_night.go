package styles

import "charm.land/lipgloss/v2"

func TokyoNightTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#7AA2F7"),
		Accent:          lipgloss.Color("#9ECE6A"),
		Warning:         lipgloss.Color("#E0AF68"),
		Danger:          lipgloss.Color("#F7768E"),
		Muted:           lipgloss.Color("#565F89"),
		Text:            lipgloss.Color("#C0CAF5"),
		Subtext:         lipgloss.Color("#A9B1D6"),
		Border:          lipgloss.Color("#3B4261"),
		BorderFocus:     lipgloss.Color("#7AA2F7"),
		SelectionBg:     lipgloss.Color("#292E42"),
		TitleText:       lipgloss.Color("#1A1B26"),
		TitleSubtext:    lipgloss.Color("#292E42"),
		HelpBg:          lipgloss.Color("#16161E"),
		HelpBarBg:       lipgloss.Color("#394B70"),
		HelpKeyBg:       lipgloss.Color("#292E42"),
		ModalBg:         lipgloss.Color("#292E42"),
		OverlayBg:       lipgloss.Color("#16161E"),
		StatusRunningFg: lipgloss.Color("#9ECE6A"),
		StatusRunningBg: lipgloss.Color("#1A2818"),
		StatusWaitingFg: lipgloss.Color("#E0AF68"),
		StatusWaitingBg: lipgloss.Color("#2A2014"),
		StatusIdleFg:    lipgloss.Color("#565F89"),
		StatusDeadFg:    lipgloss.Color("#F7768E"),
		StatusDeadBg:    lipgloss.Color("#2E1820"),
		StatusUnknownFg: lipgloss.Color("#A9B1D6"),
	}
}
