package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// Panel wraps content in a rounded border using styles.Border (Focused or Blurred).
// PURE function: consumes *styles.Styles, returns rendered output.
// MUST NOT create new lipgloss styles here — only consume *styles.Styles (C7).
func Panel(s *styles.Styles, content string, focused bool) string {
	if focused {
		return s.Border.Focused.Render(content)
	}
	return s.Border.Blurred.Render(content)
}

func PanelWithSize(s *styles.Styles, content string, focused bool, width, height int) tea.View {
	border := panelBorder(s, focused)
	borderW, borderH := border.GetFrameSize()

	// lipgloss v2 treats Style.Width(N) as the TOTAL box width including
	// padding. To make the container fill the border interior exactly
	// (no gap on the right), its width must be (width - borderW); the
	// content area inside the container is then PanelInnerSize.
	containerOuterW := max(width-borderW, 0)
	containerOuterH := max(height-borderH, 0)
	content = s.Pane.Container.Width(containerOuterW).Height(containerOuterH).Render(content)
	return tea.NewView(border.Width(width).Height(height).Render(content))
}

// PanelInnerSize returns the width and height available for content inside a
// PanelWithSize of the given total dimensions. Callers that wrap a sized
// child (e.g. a list) in PanelWithSize should size that child to the inner
// dimensions so it does not overflow the panel's frame.
func PanelInnerSize(s *styles.Styles, focused bool, width, height int) (innerW, innerH int) {
	borderW, borderH := panelBorder(s, focused).GetFrameSize()
	containerW, containerH := s.Pane.Container.GetFrameSize()
	return max(width-borderW-containerW, 0), max(height-borderH-containerH, 0)
}

// TitledPanelInnerSize is the PanelInnerSize equivalent for PanelWithTitle:
// width is identical, but height is reduced by the bottom padding the
// titled panel reserves below content (titledPanelPaddingBottom).
// Callers that pre-size sized children for a titled panel must use this.
func TitledPanelInnerSize(s *styles.Styles, focused bool, width, height int) (innerW, innerH int) {
	innerW, innerH = PanelInnerSize(s, focused, width, height)
	return innerW, max(innerH-titledPanelPaddingBottom, 0)
}

const titledPanelPaddingBottom = 1

func panelBorder(s *styles.Styles, focused bool) lipgloss.Style {
	if focused {
		return s.Border.Focused
	}
	return s.Border.Blurred
}

// PanelWithTitle renders a sized panel whose top border embeds a title:
//
//	╭─ <title> ──────╮
//	│   <content>    │
//	╰────────────────╯
//
// The top row is composed manually so its border characters (styled
// with s.Border.CharFocused/CharBlurred) and the title text (styled
// with s.Border.Title) can be coloured independently — lipgloss has no
// native title-in-border support. Sides and bottom are drawn by
// lipgloss with BorderTop disabled. Falls back to PanelWithSize (no
// title) when the width cannot fit corners + leading dash + spaces +
// title + at least one trailing dash.
func PanelWithTitle(s *styles.Styles, content, title string, focused bool, width, height int) tea.View {
	if width <= 0 || height <= 0 {
		return tea.NewView("")
	}
	borderStyle := panelBorder(s, focused)
	borderChar := s.Border.CharBlurred
	if focused {
		borderChar = s.Border.CharFocused
	}

	borderW, borderH := borderStyle.GetFrameSize()
	containerOuterW := max(width-borderW, 0)
	containerOuterH := max(height-borderH, 0)
	padded := s.Pane.Container.
		PaddingBottom(titledPanelPaddingBottom).
		Width(containerOuterW).Height(containerOuterH).
		Render(content)

	chars := lipgloss.RoundedBorder()
	titleRendered := s.Border.Title.Render(title)
	titleW := lipgloss.Width(titleRendered) + 2

	const leadDashes = 1
	const minTrailDashes = 1
	if width < 2+leadDashes+titleW+minTrailDashes {
		return PanelWithSize(s, content, focused, width, height)
	}
	trailDashes := width - 2 - leadDashes - titleW

	topRow := borderChar.Render(chars.TopLeft+strings.Repeat(chars.Top, leadDashes)) +
		" " + titleRendered + " " +
		borderChar.Render(strings.Repeat(chars.Top, trailDashes)+chars.TopRight)

	rest := borderStyle.BorderTop(false).
		Width(width).
		Height(max(height-1, 1)).
		Render(padded)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, topRow, rest))
}
