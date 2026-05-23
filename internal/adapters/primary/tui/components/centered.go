package components

import (
	"charm.land/lipgloss/v2"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// CenteredContent renders content centered both vertically and horizontally
// within a box of the given dimensions. Used for empty-state panels.
func CenteredContent(s *styles.Styles, content string, width, height int) string {
	return s.Layout.Box.
		Width(width).
		Height(height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
}
