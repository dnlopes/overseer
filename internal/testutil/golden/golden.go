package golden

import (
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Setup configures lipgloss to use ASCII color profile, stripping ANSI codes.
// Call this at the start of every golden file test.
func Setup(t *testing.T) {
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
}

// ReadBts reads all bytes from r, failing the test on error.
func ReadBts(tb testing.TB, r io.Reader) []byte {
	tb.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		tb.Fatalf("ReadBts: %v", err)
	}
	return data
}
