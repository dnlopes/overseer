package styles

import "charm.land/lipgloss/v2"

func SunsetTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#F08A5D"),
		Accent:          lipgloss.Color("#ADC178"),
		Warning:         lipgloss.Color("#F9ED69"),
		Danger:          lipgloss.Color("#B83B5E"),
		Muted:           lipgloss.Color("#806870"),
		Text:            lipgloss.Color("#F5E1D2"),
		Subtext:         lipgloss.Color("#C4A0A8"),
		Border:          lipgloss.Color("#4A2A4E"),
		BorderFocus:     lipgloss.Color("#F08A5D"),
		SelectionBg:     lipgloss.Color("#6A2C70"),
		TitleText:       lipgloss.Color("#1F0D20"),
		TitleSubtext:    lipgloss.Color("#6A2C70"),
		HelpBg:          lipgloss.Color("#1F0D20"),
		HelpBarBg:       lipgloss.Color("#6A2C70"),
		HelpKeyBg:       lipgloss.Color("#3D1F40"),
		ModalBg:         lipgloss.Color("#3D1F40"),
		OverlayBg:       lipgloss.Color("#1F0D20"),
		StatusRunningFg: lipgloss.Color("#ADC178"),
		StatusRunningBg: lipgloss.Color("#1F2A14"),
		StatusWaitingFg: lipgloss.Color("#F9ED69"),
		StatusWaitingBg: lipgloss.Color("#3A2818"),
		StatusIdleFg:    lipgloss.Color("#806870"),
		StatusDeadFg:    lipgloss.Color("#B83B5E"),
		StatusDeadBg:    lipgloss.Color("#2E1820"),
		StatusUnknownFg: lipgloss.Color("#C4A0A8"),
	}
}
