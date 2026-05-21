package styles

import (
	"image/color"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ListIndentUnit is the canonical column count per nesting level for any
// list/tree view in the TUI. Compose it at the render site to indent rows:
//
//	indent := strings.Repeat(" ", depth * styles.ListIndentUnit)
//	row    := s.ListRow.Normal.Render(indent + label)
//
// Do NOT add Padding/Margin to ListRow itself, and do NOT create
// domain-named row styles that bake position into their definition
// (Session.Item.PaddingLeft was that mistake). Position is data —
// it belongs in the render function, computed from the tree's depth.
const ListIndentUnit = 2

type BorderStyles struct {
	Focused     lipgloss.Style
	Blurred     lipgloss.Style
	CharFocused lipgloss.Style
	CharBlurred lipgloss.Style
	Title       lipgloss.Style
}

type PaneStyles struct {
	Sessions  lipgloss.Style
	Status    lipgloss.Style
	Preview   lipgloss.Style
	Container lipgloss.Style
}

type GroupStyles struct {
	Header         lipgloss.Style
	HeaderSelected lipgloss.Style
}

type StatusStyles struct {
	Label     lipgloss.Style
	Value     lipgloss.Style
	Separator lipgloss.Style
}

type FormFieldStyles struct {
	Label        lipgloss.Style
	LabelFocused lipgloss.Style
	Input        lipgloss.Style
	Error        lipgloss.Style
}

type FormStyles struct {
	Field     FormFieldStyles
	Container lipgloss.Style
	// Title is the heading rendered at the top of each modal form body
	// (e.g. "New Session", "Open Branch"). Forms compose the title above
	// the field stack so users can tell the popups apart at a glance.
	Title lipgloss.Style
	// Input holds the bubbles textinput.Styles shared by every form field
	// (placeholder appearance, focused/blurred prompt color, cursor blink).
	// Configured once at theme load — see [New].
	Input textinput.Styles
}

type HelpStyles struct {
	Bar         lipgloss.Style
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

type LayoutStyles struct {
	Box lipgloss.Style
}

type TabStyles struct {
	Active   lipgloss.Style
	Inactive lipgloss.Style
	Bar      lipgloss.Style
}

type SessionDetailsStyles struct {
	Glyph   lipgloss.Style
	Value   lipgloss.Style
	Good    lipgloss.Style
	Bad     lipgloss.Style
	Warn    lipgloss.Style
	Special lipgloss.Style
	Hint    lipgloss.Style
}

type Styles struct {
	Border   BorderStyles
	TitleBar struct {
		Base     lipgloss.Style
		Branding lipgloss.Style
		Subtext  lipgloss.Style
	}
	Pane PaneStyles
	// ListRow describes the *appearance* of a row in any list/tree view —
	// foreground, weight, selection background. It deliberately carries no
	// padding or margin: spatial position (depth indent, prefixes) is the
	// render function's job, composed via [ListIndentUnit]. Never add
	// Padding to these styles or create per-depth row variants.
	//
	// Number / NumberSelected style the leading "NN. " prefix used by
	// digit-jump lists; they share the selection background of their
	// matching label state so the row reads as a single highlight.
	ListRow struct {
		Normal         lipgloss.Style
		Selected       lipgloss.Style
		Number         lipgloss.Style
		NumberSelected lipgloss.Style
	}
	Group         GroupStyles
	Status        StatusStyles
	StatusSegment struct {
		Default   lipgloss.Style
		Highlight lipgloss.Style
	}
	Form   FormStyles
	Danger struct {
		Title lipgloss.Style
		Body  lipgloss.Style
	}
	Modal struct {
		Box          lipgloss.Style
		Overlay      color.Color
		OverlayStyle lipgloss.Style
	}
	Badge struct {
		Key   lipgloss.Style
		Label lipgloss.Style
	}
	Divider struct {
		Horizontal lipgloss.Style
	}
	Help           HelpStyles
	EmptyState     EmptyStateStyles
	TooSmall       TooSmallStyles
	Layout         LayoutStyles
	Tab            TabStyles
	SessionDetails SessionDetailsStyles
}

func New() *Styles {
	theme := LoadTheme("dark")

	helpKeyStyle := lipgloss.NewStyle().Foreground(theme.Text).Background(theme.HelpBarBg).Bold(true)
	helpBarStyle := lipgloss.NewStyle().Background(theme.HelpBarBg).Padding(0, 1)

	return &Styles{
		Border: BorderStyles{
			Focused: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderFocus),
			Blurred: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Border),
			CharFocused: lipgloss.NewStyle().Foreground(theme.BorderFocus),
			CharBlurred: lipgloss.NewStyle().Foreground(theme.Border),
			Title:       lipgloss.NewStyle().Foreground(theme.Text).Bold(true),
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
			Sessions:  lipgloss.NewStyle().Padding(0, 1),
			Status:    lipgloss.NewStyle().Padding(0, 1),
			Preview:   lipgloss.NewStyle().Padding(0, 1),
			Container: lipgloss.NewStyle().Padding(0, 1),
		},
		ListRow: struct {
			Normal         lipgloss.Style
			Selected       lipgloss.Style
			Number         lipgloss.Style
			NumberSelected lipgloss.Style
		}{
			Normal:         lipgloss.NewStyle().Foreground(theme.Text),
			Selected:       lipgloss.NewStyle().Foreground(theme.Text).Bold(true).Background(theme.SelectionBg),
			Number:         lipgloss.NewStyle().Foreground(theme.Subtext),
			NumberSelected: lipgloss.NewStyle().Foreground(theme.Subtext).Bold(true).Background(theme.SelectionBg),
		},
		Group: GroupStyles{
			Header:         lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
			HeaderSelected: lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Background(theme.SelectionBg),
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
			Title: lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).MarginBottom(1),
			Field: FormFieldStyles{
				Label:        lipgloss.NewStyle().Foreground(theme.Subtext),
				LabelFocused: lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
				Input:        lipgloss.NewStyle().Foreground(theme.Text),
				Error:        lipgloss.NewStyle().Foreground(theme.Warning),
			},
			Input: textinput.Styles{
				Focused: textinput.StyleState{
					Placeholder: lipgloss.NewStyle().Foreground(theme.Muted).Italic(true),
					Prompt:      lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
					Text:        lipgloss.NewStyle().Foreground(theme.Text),
				},
				Blurred: textinput.StyleState{
					Placeholder: lipgloss.NewStyle().Foreground(theme.Muted).Italic(true),
					Prompt:      lipgloss.NewStyle().Foreground(theme.Muted),
					Text:        lipgloss.NewStyle().Foreground(theme.Subtext),
				},
				Cursor: textinput.CursorStyle{
					Color: theme.Accent,
					Shape: tea.CursorBlock,
					Blink: true,
				},
			},
		},
		Danger: struct {
			Title lipgloss.Style
			Body  lipgloss.Style
		}{
			Title: lipgloss.NewStyle().Foreground(theme.Danger).Bold(true),
			Body:  lipgloss.NewStyle().Foreground(theme.Danger),
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
			Bar:         helpBarStyle,
			Key:         helpKeyStyle,
			Description: lipgloss.NewStyle().Foreground(theme.Subtext).Background(theme.HelpBarBg),
			Separator:   lipgloss.NewStyle().Foreground(theme.Muted).Background(theme.HelpBarBg),
		},
		EmptyState: EmptyStateStyles{
			Title: lipgloss.NewStyle().Foreground(theme.Text).Bold(true),
			Hint:  lipgloss.NewStyle().Foreground(theme.Subtext),
		},
		TooSmall: TooSmallStyles{
			Message: lipgloss.NewStyle().Foreground(theme.Warning).Bold(true),
		},
		Layout: LayoutStyles{
			Box: lipgloss.NewStyle(),
		},
		Tab: TabStyles{
			Active:   lipgloss.NewStyle().Foreground(theme.TitleText).Background(theme.Primary).Bold(true).Padding(0, 2),
			Inactive: lipgloss.NewStyle().Foreground(theme.Subtext).Padding(0, 2),
			Bar:      lipgloss.NewStyle().Foreground(theme.Border),
		},
		SessionDetails: SessionDetailsStyles{
			Glyph:   lipgloss.NewStyle().Foreground(theme.Muted),
			Value:   lipgloss.NewStyle().Foreground(theme.Text),
			Good:    lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
			Bad:     lipgloss.NewStyle().Foreground(theme.Danger).Bold(true),
			Warn:    lipgloss.NewStyle().Foreground(theme.Warning).Bold(true),
			Special: lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
			Hint:    lipgloss.NewStyle().Foreground(theme.Subtext).Italic(true),
		},
	}
}
