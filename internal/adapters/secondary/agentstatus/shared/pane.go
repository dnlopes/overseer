package shared

import "strings"

// LastNonEmptyLines returns the last n lines of s that contain at least one
// non-whitespace character, in original (top-down) order. n ≤ 0 or empty
// input yields a nil-equivalent zero-length slice. Detectors call this to
// scan the freshest pane content without dragging in the scrollback of
// blank rows tmux pads each pane with.
func LastNonEmptyLines(s string, n int) []string {
	if n <= 0 || s == "" {
		return nil
	}
	all := strings.Split(s, "\n")
	out := make([]string, 0, n)
	for i := len(all) - 1; i >= 0 && len(out) < n; i-- {
		if strings.TrimSpace(all[i]) == "" {
			continue
		}
		out = append(out, all[i])
	}
	reverseInPlace(out)
	return out
}

func reverseInPlace(xs []string) {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
}
