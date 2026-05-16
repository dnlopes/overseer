package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type BorderStyles struct {
	Focused lipgloss.Style
	Blurred lipgloss.Style
}

type PaneStyles struct {
	Sessions lipgloss.Style
	Status   lipgloss.Style
	Preview  lipgloss.Style
}

type GroupStyles struct {
	Header lipgloss.Style
}

type SessionItemStyles struct {
	Normal   lipgloss.Style
	Selected lipgloss.Style
}

type SessionStyles struct {
	Item SessionItemStyles
}

type StatusStyles struct {
	Label     lipgloss.Style
	Value     lipgloss.Style
	Separator lipgloss.Style
}

type FormFieldStyles struct {
	Label lipgloss.Style
	Input lipgloss.Style
	Error lipgloss.Style
}

type FormStyles struct {
	Field     FormFieldStyles
	Container lipgloss.Style
}

type HelpStyles struct {
	Key         lipgloss.Style
	Description lipgloss.Style
	Separator   lipgloss.Style
}

type EmptyStateStyles struct {
	Title lipgloss.Style
	Hint  lipgloss.Style
}

type TooSmallStyles struct {
	Message lipgloss.Style
}

type Styles struct {
	Border     BorderStyles
	TitleBar   struct {
		Base     lipgloss.Style
		Branding lipgloss.Style
		Subtext  lipgloss.Style
	}
	Pane       PaneStyles
	ListRow    struct {
		Normal   lipgloss.Style
		Selected lipgloss.Style
	}
	Group      GroupStyles
	Session    SessionStyles
	Status     StatusStyles
	StatusSegment struct {
		Default   lipgloss.Style
		Highlight lipgloss.Style
	}
	Form       FormStyles
	Modal      struct {
		Box          lipgloss.Style
		Overlay      color.Color
		OverlayStyle lipgloss.Style
	}
	Badge      struct {
		Key   lipgloss.Style
		Label lipgloss.Style
	}
	Divider    struct {
		Horizontal lipgloss.Style
	}
	Help       HelpStyles
	EmptyState EmptyStateStyles
	TooSmall   TooSmallStyles
}

func New() *Styles {
	theme := LoadTheme("dark")

	helpKeyStyle := lipgloss.NewStyle().Foreground(theme.Text).Background(theme.HelpKeyBg).Padding(0, 1)

	return &Styles{
		Border: BorderStyles{
			Focused: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderFocus),
			Blurred: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Border),
		},
		TitleBar: struct {
			Base     lipgloss.Style
			Branding lipgloss.Style
			Subtext  lipgloss.Style
		}{
			Base:     lipgloss.NewStyle().Background(theme.Primary).Foreground(theme.TitleText),
			Branding: lipgloss.NewStyle().Background(theme.Primary).Foreground(theme.TitleText).Bold(true).Padding(0, 1),
			Subtext:  lipgloss.NewStyle().Background(theme.Primary).Foreground(theme.TitleSubtext).Padding(0, 1),
		},
		Pane: PaneStyles{
			Sessions: lipgloss.NewStyle().Padding(0, 1),
			Status:   lipgloss.NewStyle().Padding(0, 1),
			Preview:  lipgloss.NewStyle().Padding(0, 1),
		},
		ListRow: struct {
			Normal   lipgloss.Style
			Selected lipgloss.Style
		}{
			Normal:   lipgloss.NewStyle().Foreground(theme.Text),
			Selected: lipgloss.NewStyle().Foreground(theme.Text).Bold(true).Background(theme.SelectionBg),
		},
		Group: GroupStyles{
			Header: lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
		},
		Session: SessionStyles{
			Item: SessionItemStyles{
				Normal:   lipgloss.NewStyle().PaddingLeft(2).Foreground(theme.Text),
				Selected: lipgloss.NewStyle().PaddingLeft(2).Foreground(theme.Text).Bold(true).Background(theme.SelectionBg),
			},
		},
		Status: StatusStyles{
			Label:     lipgloss.NewStyle().Foreground(theme.Subtext),
			Value:     lipgloss.NewStyle().Foreground(theme.Text),
			Separator: lipgloss.NewStyle().Foreground(theme.Muted),
		},
		StatusSegment: struct {
			Default   lipgloss.Style
			Highlight lipgloss.Style
		}{
			Default:   lipgloss.NewStyle().Background(theme.HelpBg).Foreground(theme.Subtext).Padding(0, 1),
			Highlight: lipgloss.NewStyle().Background(theme.Primary).Foreground(theme.TitleText).Padding(0, 1).Bold(true),
		},
		Form: FormStyles{
			Container: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderFocus).
				Padding(1, 2),
			Field: FormFieldStyles{
				Label: lipgloss.NewStyle().Foreground(theme.Subtext),
				Input: lipgloss.NewStyle().Foreground(theme.Text),
				Error: lipgloss.NewStyle().Foreground(theme.Warning),
			},
		},
		Modal: struct {
			Box          lipgloss.Style
			Overlay      color.Color
			OverlayStyle lipgloss.Style
		}{
			Box: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderFocus).
				Background(theme.ModalBg).
				Foreground(theme.Text).
				Padding(1, 3),
			Overlay:      theme.OverlayBg,
			OverlayStyle: lipgloss.NewStyle().Background(theme.OverlayBg),
		},
		Badge: struct {
			Key   lipgloss.Style
			Label lipgloss.Style
		}{
			Key:   helpKeyStyle,
			Label: lipgloss.NewStyle().Foreground(theme.Subtext),
		},
		Divider: struct {
			Horizontal lipgloss.Style
		}{
			Horizontal: lipgloss.NewStyle().Foreground(theme.Border),
		},
		Help: HelpStyles{
			Key:         helpKeyStyle,
			Description: lipgloss.NewStyle().Foreground(theme.Subtext),
			Separator:   lipgloss.NewStyle().Foreground(theme.Muted),
		},
		EmptyState: EmptyStateStyles{
			Title: lipgloss.NewStyle().Foreground(theme.Text).Bold(true),
			Hint:  lipgloss.NewStyle().Foreground(theme.Subtext),
		},
		TooSmall: TooSmallStyles{
			Message: lipgloss.NewStyle().Foreground(theme.Warning).Bold(true),
		},
	}
}
