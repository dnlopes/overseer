package styles

import "charm.land/lipgloss/v2"

func OneDarkTheme() Theme {
	return Theme{
		Primary:         lipgloss.Color("#61AFEF"),
		Accent:          lipgloss.Color("#98C379"),
		Warning:         lipgloss.Color("#E5C07B"),
		Danger:          lipgloss.Color("#E06C75"),
		Muted:           lipgloss.Color("#5C6370"),
		Text:            lipgloss.Color("#ABB2BF"),
		Subtext:         lipgloss.Color("#828997"),
		Border:          lipgloss.Color("#3E4451"),
		BorderFocus:     lipgloss.Color("#61AFEF"),
		SelectionBg:     lipgloss.Color("#3E4451"),
		TitleText:       lipgloss.Color("#282C34"),
		TitleSubtext:    lipgloss.Color("#3E4451"),
		HelpBg:          lipgloss.Color("#21252B"),
		HelpBarBg:       lipgloss.Color("#2C313A"),
		HelpKeyBg:       lipgloss.Color("#3E4451"),
		ModalBg:         lipgloss.Color("#2C313A"),
		OverlayBg:       lipgloss.Color("#21252B"),
		StatusRunningFg: lipgloss.Color("#98C379"),
		StatusRunningBg: lipgloss.Color("#1F2D1F"),
		StatusWaitingFg: lipgloss.Color("#E5C07B"),
		StatusWaitingBg: lipgloss.Color("#2E2814"),
		StatusIdleFg:    lipgloss.Color("#5C6370"),
		StatusDeadFg:    lipgloss.Color("#E06C75"),
		StatusDeadBg:    lipgloss.Color("#2E1820"),
		StatusUnknownFg: lipgloss.Color("#828997"),
	}
}
