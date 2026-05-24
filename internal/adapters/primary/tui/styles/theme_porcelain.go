package styles

import "charm.land/lipgloss/v2"

func PorcelainTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#3F72AF"),
		Accent:          lipgloss.Color("#14B8A6"),
		Warning:         lipgloss.Color("#D97706"),
		Danger:          lipgloss.Color("#B91C1C"),
		Muted:           lipgloss.Color("#94A3B8"),
		Text:            lipgloss.Color("#112D4E"),
		Subtext:         lipgloss.Color("#3F72AF"),
		Border:          lipgloss.Color("#DBE2EF"),
		BorderFocus:     lipgloss.Color("#3F72AF"),
		SelectionBg:     lipgloss.Color("#DBE2EF"),
		TitleText:       lipgloss.Color("#F9F7F7"),
		TitleSubtext:    lipgloss.Color("#DBE2EF"),
		HelpBg:          lipgloss.Color("#F9F7F7"),
		HelpBarBg:       lipgloss.Color("#DBE2EF"),
		HelpKeyBg:       lipgloss.Color("#FFFFFF"),
		ModalBg:         lipgloss.Color("#FFFFFF"),
		OverlayBg:       lipgloss.Color("#F9F7F7"),
		StatusRunningFg: lipgloss.Color("#047857"),
		StatusRunningBg: lipgloss.Color("#D1FAE5"),
		StatusWaitingFg: lipgloss.Color("#B45309"),
		StatusWaitingBg: lipgloss.Color("#FEF3C7"),
		StatusIdleFg:    lipgloss.Color("#94A3B8"),
		StatusDeadFg:    lipgloss.Color("#B91C1C"),
		StatusDeadBg:    lipgloss.Color("#FECACA"),
		StatusUnknownFg: lipgloss.Color("#3F72AF"),
	}
}
