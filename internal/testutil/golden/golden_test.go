package golden

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSetupStripsANSI(t *testing.T) {
	Setup(t)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	got := style.Render("hello")
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI codes stripped, got %q", got)
	}
	if got != "hello" {
		t.Fatalf("expected plain text, got %q", got)
	}
}

func TestReadBts(t *testing.T) {
	got := ReadBts(t, strings.NewReader("abc"))
	if string(got) != "abc" {
		t.Fatalf("expected %q, got %q", "abc", string(got))
	}
}
