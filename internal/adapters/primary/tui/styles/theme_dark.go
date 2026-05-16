package styles

import "charm.land/lipgloss/v2"

func DarkTheme() Theme {
	return Theme{
		Primary:      lipgloss.Color("#7C3AED"),
		Accent:       lipgloss.Color("#10B981"),
		Warning:      lipgloss.Color("#F59E0B"),
		Muted:        lipgloss.Color("#6B7280"),
		Text:         lipgloss.Color("#F9FAFB"),
		Subtext:      lipgloss.Color("#9CA3AF"),
		Border:       lipgloss.Color("#374151"),
		BorderFocus:  lipgloss.Color("#7C3AED"),
		SelectionBg:  lipgloss.Color("#3730A3"),
		TitleText:    lipgloss.Color("#F9FAFB"),
		TitleSubtext: lipgloss.Color("#E0E7FF"),
		HelpBg:       lipgloss.Color("#111827"),
		HelpKeyBg:    lipgloss.Color("#1F2937"),
		ModalBg:      lipgloss.Color("#1F2937"),
		OverlayBg:    lipgloss.Color("#111827"),
	}
}
