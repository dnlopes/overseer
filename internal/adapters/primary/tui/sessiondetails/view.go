package sessiondetails

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
)

const (
	glyphFeatureBranch = "⎇"
	glyphBaseBranch    = "↳"
	glyphWorktree      = "~"
	glyphPRStateDot    = "●"
	glyphCheckPass     = "✓"
	glyphCheckFail     = "✗"
	glyphCheckPending  = "◷"
)

func (m Model) renderContent(width int) string {
	if m.session == nil {
		return m.styles.SessionDetails.Hint.Render("Select a session")
	}

	var sections [][]string
	if m.session.HasWorktree() {
		sections = append(sections, m.renderPRSection(width))
		sections = append(sections, m.renderWorktreeSection(width))
	}

	return joinSections(sections)
}

func joinSections(sections [][]string) string {
	parts := make([]string, 0, len(sections)*2)
	for i, sec := range sections {
		if len(sec) == 0 {
			continue
		}
		if i > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, sec...)
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderPRSection(width int) []string {
	pr, ok := m.prCache[m.session.ID]
	if !ok {
		return []string{m.styles.SessionDetails.Hint.Render("◌ fetching PR…")}
	}
	if pr.PR.Number == 0 {
		return []string{m.styles.SessionDetails.Hint.Render("● no PR yet")}
	}
	s := &m.styles.SessionDetails
	header := prStateStyle(s, pr.PR.State).Render(glyphPRStateDot+" "+string(pr.PR.State)) +
		"  " + s.Glyph.Render(fmt.Sprintf("#%d", pr.PR.Number))

	stats := s.Good.Render(fmt.Sprintf("+%d", pr.PR.Stats.Additions)) +
		"  " + s.Bad.Render(fmt.Sprintf("-%d", pr.PR.Stats.Deletions)) +
		"  " + s.Glyph.Render(fmt.Sprintf("%d files", pr.PR.Stats.ChangedFiles))

	lines := []string{truncate(header, width), truncate(stats, width)}
	if checks := renderChecksLine(s, pr.PR.Checks); checks != "" {
		lines = append(lines, truncate(checks, width))
	}
	return lines
}

func renderChecksLine(s *styles.SessionDetailsStyles, c domain.PRChecks) string {
	if c.Total == 0 {
		return ""
	}
	var parts []string
	if c.Passing > 0 {
		parts = append(parts, s.Good.Render(fmt.Sprintf("%s %d", glyphCheckPass, c.Passing)))
	}
	if c.Failing > 0 {
		parts = append(parts, s.Bad.Render(fmt.Sprintf("%s %d", glyphCheckFail, c.Failing)))
	}
	if c.Pending > 0 {
		parts = append(parts, s.Warn.Render(fmt.Sprintf("%s %d", glyphCheckPending, c.Pending)))
	}
	return strings.Join(parts, "   ")
}

func prStateStyle(s *styles.SessionDetailsStyles, state domain.PRState) lipgloss.Style {
	switch state {
	case domain.PRStateOpen:
		return s.Good
	case domain.PRStateMerged:
		return s.Special
	case domain.PRStateClosed:
		return s.Bad
	}
	return s.Glyph
}

func (m Model) renderWorktreeSection(width int) []string {
	s := &m.styles.SessionDetails
	rows := make([]string, 0, 3)
	rows = append(rows, glyphLine(s, glyphFeatureBranch, m.session.FeatureBranch, width))
	rows = append(rows, glyphLine(s, glyphBaseBranch, m.session.BaseBranch, width))
	rows = append(rows, pathLine(s, glyphWorktree, m.session.WorktreePath, width))
	return rows
}

func glyphLine(s *styles.SessionDetailsStyles, glyph, value string, width int) string {
	prefix := s.Glyph.Render(glyph + "  ")
	avail := width - lipgloss.Width(prefix)
	return prefix + s.Value.Render(truncate(value, avail))
}

func pathLine(s *styles.SessionDetailsStyles, glyph, path string, width int) string {
	prefix := s.Glyph.Render(glyph + "  ")
	avail := width - lipgloss.Width(prefix)
	return prefix + s.Value.Render(truncatePath(path, avail))
}

// truncate clips s to maxWidth, replacing the trailing characters with "…"
// when truncation occurs. maxWidth ≤ 0 returns empty; maxWidth < 2 returns "…".
func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth < 2 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// truncatePath clips path from the LEFT (keeping the deepest component
// visible), prefixing with "…" when truncation occurs. Useful for
// long worktree paths where the trailing directory is the meaningful part.
func truncatePath(path string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(path) <= maxWidth {
		return path
	}
	if maxWidth < 2 {
		return "…"
	}
	runes := []rune(path)
	keep := maxWidth - 1
	if keep >= len(runes) {
		return path
	}
	return "…" + string(runes[len(runes)-keep:])
}
