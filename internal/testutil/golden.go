// Package testutil holds shared test helpers (fixtures, golden file helpers,
// teatest harness). Mocks live in the testutil/mocks subpackage.
package testutil

import (
	"io"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Setup is a placeholder hook for golden-test setup. Call at the start of every golden test.
func Setup(t *testing.T) {
	t.Helper()
}

// StripANSI removes ANSI escape codes from s.
func StripANSI(s string) string { return ansi.Strip(s) }

// ReadBts reads all bytes from r, failing the test on error.
func ReadBts(tb testing.TB, r io.Reader) []byte {
	tb.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		tb.Fatalf("ReadBts: %v", err)
	}
	return data
}
