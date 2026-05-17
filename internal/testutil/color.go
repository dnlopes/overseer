package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/aymanbagabas/go-udiff"

	"github.com/dnlopes/overseer/internal/shared/paths"
)

// RequireEqualColor asserts that actual matches the color golden file named by name.
// Unlike golden frameworks that strip ANSI, this preserves escape sequences for comparison.
func RequireEqualColor(t testing.TB, name string, actual string) {
	t.Helper()

	golden := colorGoldenPath(name)
	if updateGolden() {
		if err := os.MkdirAll(filepath.Dir(golden), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := paths.AtomicWrite(golden, []byte(actual)); err != nil {
			t.Fatal(err)
		}
	}

	goldenBts, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}

	goldenStr := normalizeWindowsLineBreaks(string(goldenBts))
	actualStr := string(actual)

	diff := udiff.Unified("golden", "run", visibleEscapes(goldenStr), visibleEscapes(actualStr))
	if diff != "" {
		t.Fatalf("output does not match, expected:\n\n%s\n\ngot:\n\n%s\n\ndiff:\n\n%s", visibleEscapes(goldenStr), visibleEscapes(actualStr), diff)
	}
}

func colorGoldenPath(name string) string {
	return filepath.Join("testdata", "golden", "color", name+".golden")
}

// updateGolden returns true if the -update flag was set.
func updateGolden() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}
	return f.Value.String() == "true"
}

func visibleEscapes(in string) string {
	lines := strings.Split(in, "\n")
	for i, line := range lines {
		quoted := strconv.Quote(line)
		quoted = strings.TrimPrefix(quoted, `"`)
		quoted = strings.TrimSuffix(quoted, `"`)
		lines[i] = strings.ReplaceAll(quoted, `\x1b`, `\e`)
	}
	return strings.Join(lines, "\n")
}

func normalizeWindowsLineBreaks(str string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(str, "\r\n", "\n")
	}
	return str
}
