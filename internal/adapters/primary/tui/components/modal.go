package components

import (
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

func Modal(s *styles.Styles, body string, width, height int) string {
	return s.Modal.Box.Render(body)
}
