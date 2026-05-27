package styles

import "charm.land/lipgloss/v2"

func GitHubDarkTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#1F6FEB"),
		Accent:          lipgloss.Color("#3FB950"),
		Warning:         lipgloss.Color("#D29922"),
		Danger:          lipgloss.Color("#F85149"),
		Muted:           lipgloss.Color("#6E7681"),
		Text:            lipgloss.Color("#C9D1D9"),
		Subtext:         lipgloss.Color("#8B949E"),
		Border:          lipgloss.Color("#58A6FF"),
		BorderFocus:     lipgloss.Color("#58A6FF"),
		SelectionBg:     lipgloss.Color("#1F2D3D"),
		TitleText:       lipgloss.Color("#FFFFFF"),
		TitleSubtext:    lipgloss.Color("#C8E1FF"),
		HelpBg:          lipgloss.Color("#161B22"),
		HelpBarBg:       lipgloss.Color("#0D419D"),
		HelpKeyBg:       lipgloss.Color("#21262D"),
		ModalBg:         lipgloss.Color("#161B22"),
		OverlayBg:       lipgloss.Color("#010409"),
		StatusRunningFg: lipgloss.Color("#3FB950"),
		StatusRunningBg: lipgloss.Color("#0E2818"),
		StatusWaitingFg: lipgloss.Color("#D29922"),
		StatusWaitingBg: lipgloss.Color("#2A1F0B"),
		StatusIdleFg:    lipgloss.Color("#6E7681"),
		StatusDeadFg:    lipgloss.Color("#F85149"),
		StatusDeadBg:    lipgloss.Color("#2E1212"),
		StatusUnknownFg: lipgloss.Color("#8B949E"),
	}
}
