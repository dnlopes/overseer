package domain

import (
	"errors"
	"strings"
)

const (
	labelCodeMaxLen  = 50
	labelGlyphMaxLen = 20
)

// Label is a configured status indicator that can be attached to a Session.
// Code is the persisted identifier (e.g. "WIP", "reviewing"); Color is a
// lipgloss-compatible color string (hex like "#F59E0B" or a named color
// like "yellow") used to render the badge in the sessions list. Glyph is
// an optional leading icon (emoji or unicode glyph) rendered before the
// code in the badge — empty Glyph renders just the code.
//
// Labels are configured globally (see config.Config.Labels) and selected
// per-session through the l keybinding via Service.CycleLabel. The
// in-memory list is the cycle order — empty -> first -> second -> ... ->
// last -> empty (clears the label). See NextLabelCode.
type Label struct {
	Code  string
	Color string
	Glyph string
}

// DefaultLabels is the ordered cycle used when the user has not customised
// labels in config. Codes are short status verbs; colors are hex strings
// chosen for readable contrast on dark backgrounds and reuse the existing
// theme palette where possible.
//
// Glyphs are deliberately monochrome BMP geometric symbols (Unicode 1.1,
// blocks U+25xx / U+27xx) rather than color emojis: they render in every
// modern terminal font without requiring an emoji font, and they are
// single-cell-wide in non-CJK contexts so alignment is reliable across
// terminals (iTerm2, Alacritty, Kitty, Wezterm, Terminal.app, gnome-
// terminal, Windows Terminal, tmux/ssh). Users who want emojis can
// override per-label in config — alignment still works because the
// renderer uses lipgloss.Width which handles wide characters.
//
// When the user defines labels in their config file, this list is fully
// replaced (not appended-to) — see config.Load. The intent is that custom
// labels supersede the defaults entirely.
var DefaultLabels = []Label{
	{Code: "WIP", Color: "#F59E0B", Glyph: "◐"},
	{Code: "reviewing", Color: "#22D3EE", Glyph: "◉"},
	{Code: "ready", Color: "#60A5FA", Glyph: "◆"},
	{Code: "done", Color: "#10B981", Glyph: "✓"},
	{Code: "draft", Color: "#9CA3AF", Glyph: "◌"},
}

// NewLabel constructs a validated Label. Code, Color and Glyph are trimmed.
// Empty code (after trim) is rejected; codes over 50 characters are
// rejected; empty color is rejected. Glyph is optional — empty is allowed
// and renders as a plain text badge — but glyphs over 20 characters are
// rejected (a sanity cap above multi-codepoint compound emojis).
// Color is not parsed here — invalid color strings only become apparent
// at render time when lipgloss treats them as transparent.
func NewLabel(code, color, glyph string) (Label, error) {
	code = strings.TrimSpace(code)
	color = strings.TrimSpace(color)
	glyph = strings.TrimSpace(glyph)

	if code == "" {
		return Label{}, ErrLabelEmptyCode
	}
	if len(code) > labelCodeMaxLen {
		return Label{}, ErrLabelCodeTooLong
	}
	if color == "" {
		return Label{}, ErrLabelEmptyColor
	}
	if len(glyph) > labelGlyphMaxLen {
		return Label{}, ErrLabelGlyphTooLong
	}

	return Label{Code: code, Color: color, Glyph: glyph}, nil
}

// NextLabelCode advances through the label cycle, returning the next
// code to assign. The cycle includes the empty state so users can clear
// a label by pressing L enough times:
//
//   - "" (no label)    → labels[0].Code  (assign first)
//   - labels[i].Code   → labels[i+1].Code (advance)
//   - labels[last].Code → ""              (clear)
//   - unknown current   → labels[0].Code  (stale config; reset to first)
//   - empty labels list → ""              (nothing to cycle)
//
// The function is pure — it never mutates its arguments.
func NextLabelCode(current string, labels []Label) string {
	if len(labels) == 0 {
		return ""
	}
	if current == "" {
		return labels[0].Code
	}
	for i, l := range labels {
		if l.Code == current {
			if i == len(labels)-1 {
				return ""
			}
			return labels[i+1].Code
		}
	}
	// Current code is not in the configured list — treat as stale and
	// reset to the first label on next cycle.
	return labels[0].Code
}

// FindLabel returns the label whose Code matches needle. The boolean
// reports whether a match was found; on miss the returned Label is the
// zero value. Used by the TUI to resolve a session's stored label code
// to the full Label (so the renderer can colour the badge).
func FindLabel(needle string, labels []Label) (Label, bool) {
	for _, l := range labels {
		if l.Code == needle {
			return l, true
		}
	}
	return Label{}, false
}

var (
	ErrLabelEmptyCode    = errors.New("label code cannot be empty")
	ErrLabelCodeTooLong  = errors.New("label code exceeds 50 characters")
	ErrLabelEmptyColor   = errors.New("label color cannot be empty")
	ErrLabelGlyphTooLong = errors.New("label glyph exceeds 20 characters")
)
