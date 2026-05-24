package styles

import "charm.land/lipgloss/v2"

func DeepSeaTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#3282B8"),
		Accent:          lipgloss.Color("#82C8B5"),
		Warning:         lipgloss.Color("#F4A261"),
		Danger:          lipgloss.Color("#E76F51"),
		Muted:           lipgloss.Color("#5A7A91"),
		Text:            lipgloss.Color("#BBE1FA"),
		Subtext:         lipgloss.Color("#87B5D1"),
		Border:          lipgloss.Color("#1B262C"),
		BorderFocus:     lipgloss.Color("#3282B8"),
		SelectionBg:     lipgloss.Color("#0F4C75"),
		TitleText:       lipgloss.Color("#FFFFFF"),
		TitleSubtext:    lipgloss.Color("#BBE1FA"),
		HelpBg:          lipgloss.Color("#0E1A22"),
		HelpBarBg:       lipgloss.Color("#0F4C75"),
		HelpKeyBg:       lipgloss.Color("#1B262C"),
		ModalBg:         lipgloss.Color("#1B262C"),
		OverlayBg:       lipgloss.Color("#0A1419"),
		StatusRunningFg: lipgloss.Color("#82C8B5"),
		StatusRunningBg: lipgloss.Color("#0F2A26"),
		StatusWaitingFg: lipgloss.Color("#F4A261"),
		StatusWaitingBg: lipgloss.Color("#2E2418"),
		StatusIdleFg:    lipgloss.Color("#5A7A91"),
		StatusDeadFg:    lipgloss.Color("#E76F51"),
		StatusDeadBg:    lipgloss.Color("#2E1818"),
		StatusUnknownFg: lipgloss.Color("#87B5D1"),
	}
}
