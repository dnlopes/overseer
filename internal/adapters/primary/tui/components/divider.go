package components

import (
	"strings"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// HorizontalDivider produces "─" repeated `width` times, styled with s.Divider.Horizontal.
// PURE function.
func HorizontalDivider(s *styles.Styles, width int) string {
	if width <= 0 {
		return ""
	}
	return s.Divider.Horizontal.Render(strings.Repeat("─", width))
}
