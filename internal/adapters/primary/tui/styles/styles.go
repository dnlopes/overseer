package styles

import "github.com/charmbracelet/lipgloss"

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
	Field FormFieldStyles
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
	Pane       PaneStyles
	Group      GroupStyles
	Session    SessionStyles
	Status     StatusStyles
	Form       FormStyles
	Help       HelpStyles
	EmptyState EmptyStateStyles
	TooSmall   TooSmallStyles
}

func New() *Styles {
	focusColor := lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accentColor := lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	subtleColor := lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	errorColor := lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF6B6B"}

	return &Styles{
		Border: BorderStyles{
			Focused: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(focusColor),
			Blurred: lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()),
		},
		Pane: PaneStyles{
			Sessions: lipgloss.NewStyle().Padding(0, 1),
			Status:   lipgloss.NewStyle().Padding(0, 1),
			Preview:  lipgloss.NewStyle().Padding(0, 1),
		},
		Group: GroupStyles{
			Header: lipgloss.NewStyle().Bold(true).Foreground(accentColor),
		},
		Session: SessionStyles{
			Item: SessionItemStyles{
				Normal:   lipgloss.NewStyle().PaddingLeft(2),
				Selected: lipgloss.NewStyle().PaddingLeft(2).Bold(true).Foreground(focusColor),
			},
		},
		Status: StatusStyles{
			Label:     lipgloss.NewStyle().Foreground(subtleColor),
			Value:     lipgloss.NewStyle().Bold(true),
			Separator: lipgloss.NewStyle().Foreground(subtleColor).SetString(" | "),
		},
		Form: FormStyles{
			Field: FormFieldStyles{
				Label: lipgloss.NewStyle().Bold(true),
				Input: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(subtleColor),
				Error: lipgloss.NewStyle().Foreground(errorColor),
			},
		},
		Help: HelpStyles{
			Key:         lipgloss.NewStyle().Foreground(subtleColor),
			Description: lipgloss.NewStyle().Foreground(subtleColor),
			Separator:   lipgloss.NewStyle().Foreground(subtleColor),
		},
		EmptyState: EmptyStateStyles{
			Title: lipgloss.NewStyle().Bold(true),
			Hint:  lipgloss.NewStyle().Foreground(subtleColor),
		},
		TooSmall: TooSmallStyles{
			Message: lipgloss.NewStyle().Bold(true).Foreground(errorColor),
		},
	}
}
