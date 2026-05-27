package styles

import "charm.land/lipgloss/v2"

func MonokaiTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#F92672"),
		Accent:          lipgloss.Color("#A6E22E"),
		Warning:         lipgloss.Color("#FD971F"),
		Danger:          lipgloss.Color("#F92672"),
		Muted:           lipgloss.Color("#75715E"),
		Text:            lipgloss.Color("#F8F8F2"),
		Subtext:         lipgloss.Color("#C5C5C0"),
		Border:          lipgloss.Color("#49483E"),
		BorderFocus:     lipgloss.Color("#F92672"),
		SelectionBg:     lipgloss.Color("#49483E"),
		TitleText:       lipgloss.Color("#272822"),
		TitleSubtext:    lipgloss.Color("#49483E"),
		HelpBg:          lipgloss.Color("#1E1F1A"),
		HelpBarBg:       lipgloss.Color("#49483E"),
		HelpKeyBg:       lipgloss.Color("#3E3D32"),
		ModalBg:         lipgloss.Color("#3E3D32"),
		OverlayBg:       lipgloss.Color("#1E1F1A"),
		StatusRunningFg: lipgloss.Color("#A6E22E"),
		StatusRunningBg: lipgloss.Color("#2A361A"),
		StatusWaitingFg: lipgloss.Color("#FD971F"),
		StatusWaitingBg: lipgloss.Color("#3A2A14"),
		StatusIdleFg:    lipgloss.Color("#75715E"),
		StatusDeadFg:    lipgloss.Color("#F92672"),
		StatusDeadBg:    lipgloss.Color("#3A1A24"),
		StatusUnknownFg: lipgloss.Color("#C5C5C0"),
	}
}
