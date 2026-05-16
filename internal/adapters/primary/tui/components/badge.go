package components

import "github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"

func KeyBadge(s *styles.Styles, key, label string) string {
	return s.Badge.Key.Render(key) + " " + s.Badge.Label.Render(label)
}
