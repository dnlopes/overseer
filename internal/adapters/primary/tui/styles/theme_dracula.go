package styles

import "charm.land/lipgloss/v2"

func DraculaTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#BD93F9"),
		Accent:          lipgloss.Color("#50FA7B"),
		Warning:         lipgloss.Color("#FFB86C"),
		Danger:          lipgloss.Color("#FF5555"),
		Muted:           lipgloss.Color("#6272A4"),
		Text:            lipgloss.Color("#F8F8F2"),
		Subtext:         lipgloss.Color("#BFBFBF"),
		Border:          lipgloss.Color("#44475A"),
		BorderFocus:     lipgloss.Color("#BD93F9"),
		SelectionBg:     lipgloss.Color("#44475A"),
		TitleText:       lipgloss.Color("#21222C"),
		TitleSubtext:    lipgloss.Color("#44475A"),
		HelpBg:          lipgloss.Color("#21222C"),
		HelpBarBg:       lipgloss.Color("#383A59"),
		HelpKeyBg:       lipgloss.Color("#44475A"),
		ModalBg:         lipgloss.Color("#44475A"),
		OverlayBg:       lipgloss.Color("#21222C"),
		StatusRunningFg: lipgloss.Color("#50FA7B"),
		StatusRunningBg: lipgloss.Color("#1A3526"),
		StatusWaitingFg: lipgloss.Color("#FFB86C"),
		StatusWaitingBg: lipgloss.Color("#3A2E18"),
		StatusIdleFg:    lipgloss.Color("#6272A4"),
		StatusDeadFg:    lipgloss.Color("#FF5555"),
		StatusDeadBg:    lipgloss.Color("#3A1A1A"),
		StatusUnknownFg: lipgloss.Color("#BFBFBF"),
	}
}
