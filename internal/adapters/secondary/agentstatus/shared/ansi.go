// Package shared holds detector helpers reused across per-agent
// implementations under internal/adapters/secondary/agentstatus.
//
// Helpers in this package are intentionally tiny and pure — they have no
// dependencies on the tmux adapter, the registry, or the service. Per-agent
// detectors call them to normalize raw pane captures before pattern
// matching.
package shared

import "regexp"

// ansiCSI matches the CSI (Control Sequence Introducer) family of ANSI
// escape sequences: ESC '[' <params> <final>. tmux capture-pane -e emits
// these to preserve foreground / background colors; detectors must strip
// them before pattern matching so signals like "esc to interrupt" match
// regardless of styling.
var ansiCSI = regexp.MustCompile(`\x1b\[[0-9;:?]*[ -/]*[@-~]`)

// ansiOSCHyperlink matches OSC 8 hyperlink wrappers: ESC ']' '8' ';'
// <params> ';' <uri> ST <text> ESC ']' '8' ';' ';' ST. tmux preserves these
// when `-e` is passed; stripping the wrapper leaves the link text in place
// so signals embedded inside hyperlinks ("PR #11", etc.) still match.
var ansiOSCHyperlink = regexp.MustCompile(`\x1b\]8;[^\x1b]*\x1b\\`)

// StripANSI returns s with all ANSI escape sequences removed. It strips
// both CSI sequences (color / cursor codes) and OSC 8 hyperlink wrappers,
// leaving plain text suitable for substring matching.
func StripANSI(s string) string {
	s = ansiOSCHyperlink.ReplaceAllString(s, "")
	s = ansiCSI.ReplaceAllString(s, "")
	return s
}
