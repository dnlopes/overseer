package styles

import "charm.land/lipgloss/v2"

func NordTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#5E81AC"),
		Accent:          lipgloss.Color("#A3BE8C"),
		Warning:         lipgloss.Color("#EBCB8B"),
		Danger:          lipgloss.Color("#BF616A"),
		Muted:           lipgloss.Color("#4C566A"),
		Text:            lipgloss.Color("#ECEFF4"),
		Subtext:         lipgloss.Color("#D8DEE9"),
		Border:          lipgloss.Color("#434C5E"),
		BorderFocus:     lipgloss.Color("#88C0D0"),
		SelectionBg:     lipgloss.Color("#434C5E"),
		TitleText:       lipgloss.Color("#ECEFF4"),
		TitleSubtext:    lipgloss.Color("#D8DEE9"),
		HelpBg:          lipgloss.Color("#242933"),
		HelpBarBg:       lipgloss.Color("#3B4252"),
		HelpKeyBg:       lipgloss.Color("#434C5E"),
		ModalBg:         lipgloss.Color("#3B4252"),
		OverlayBg:       lipgloss.Color("#242933"),
		StatusRunningFg: lipgloss.Color("#A3BE8C"),
		StatusRunningBg: lipgloss.Color("#2A3A2A"),
		StatusWaitingFg: lipgloss.Color("#EBCB8B"),
		StatusWaitingBg: lipgloss.Color("#3A3024"),
		StatusIdleFg:    lipgloss.Color("#4C566A"),
		StatusDeadFg:    lipgloss.Color("#BF616A"),
		StatusDeadBg:    lipgloss.Color("#3A2228"),
		StatusUnknownFg: lipgloss.Color("#D8DEE9"),
	}
}
