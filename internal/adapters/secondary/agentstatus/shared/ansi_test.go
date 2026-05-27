package shared

import (
	"strings"
	"testing"
)

func TestStripANSI_RemovesCSISequences(t *testing.T) {
	input := "\x1b[38;2;201;209;217mSisyphus\x1b[0m"
	got := StripANSI(input)
	want := "Sisyphus"
	if got != want {
		t.Fatalf("StripANSI() = %q, want %q", got, want)
	}
}

func TestStripANSI_RemovesAllCSIFormsInRealFixture(t *testing.T) {
	// Fragment lifted verbatim from a real `tmux capture-pane -p -e`
	// output produced by Claude Code in the Running state.
	input := "\x1b[38;2;201;209;217mesc \x1b[38;2;139;148;158minterrupt\x1b[38;2;255;255;255m"
	got := StripANSI(input)
	want := "esc interrupt"
	if got != want {
		t.Fatalf("StripANSI() = %q, want %q", got, want)
	}
}

func TestStripANSI_RemovesOSCHyperlinks(t *testing.T) {
	// tmux capture-pane -e preserves OSC 8 hyperlinks; the sequence is
	// ESC ] 8 ; <params> ; <uri> ESC \ <text> ESC ] 8 ; ; ESC \ .
	input := "PR \x1b]8;id=foo;https://github.com/x/y/pull/11\x1b\\#11\x1b]8;;\x1b\\ done"
	got := StripANSI(input)
	want := "PR #11 done"
	if got != want {
		t.Fatalf("StripANSI() = %q, want %q", got, want)
	}
}

func TestStripANSI_PreservesPlainText(t *testing.T) {
	input := "Sisyphus - Ultraworker · Claude Opus 4.7"
	got := StripANSI(input)
	if got != input {
		t.Fatalf("StripANSI() = %q, want unchanged %q", got, input)
	}
}

func TestStripANSI_HandlesMultilineInput(t *testing.T) {
	input := "\x1b[31mline1\x1b[0m\n\x1b[32mline2\x1b[0m\n"
	got := StripANSI(input)
	want := "line1\nline2\n"
	if got != want {
		t.Fatalf("StripANSI() = %q, want %q", got, want)
	}
}

func TestStripANSI_EmptyString(t *testing.T) {
	if got := StripANSI(""); got != "" {
		t.Fatalf("StripANSI(\"\") = %q, want empty", got)
	}
}

func TestStripANSI_OutputIsAllPrintable(t *testing.T) {
	// Comprehensive sanity: stripped output should contain no ESC.
	input := "\x1b[38;2;0;206;209mSisyphus\x1b[0m\nmore"
	got := StripANSI(input)
	if strings.ContainsRune(got, '\x1b') {
		t.Fatalf("StripANSI() left ESC bytes in output: %q", got)
	}
}
