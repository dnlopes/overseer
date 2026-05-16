package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireEqualColor_MatchesIdentical(t *testing.T) {
	withColorGolden(t, "matches_identical", "\x1b[31mhello\x1b[0m", func(name string) {
		RequireEqualColor(t, name, "\x1b[31mhello\x1b[0m")
	})
}

func TestRequireEqualColor_DetectsColorChange(t *testing.T) {
	withColorGolden(t, "detects_color_change", "\x1b[31mhello\x1b[0m", func(name string) {
		probe := &recordingTB{name: t.Name()}
		RequireEqualColor(probe, name, "\x1b[32mhello\x1b[0m")
		if !probe.failed {
			t.Fatal("expected color-only change to fail")
		}
	})
}

func TestRequireEqualColor_UpdateMode(t *testing.T) {
	name := "update_mode"
	path := colorGoldenPath(name)
	original := "\x1b[31mold\x1b[0m"
	updated := "\x1b[32mnew\x1b[0m"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	old := *update
	*update = true
	t.Cleanup(func() { *update = old })

	RequireEqualColor(t, name, updated)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != updated {
		t.Fatalf("expected updated golden %q, got %q", updated, string(got))
	}
}

func withColorGolden(t *testing.T, name, content string, run func(name string)) {
	t.Helper()
	path := colorGoldenPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	old := *update
	*update = false
	t.Cleanup(func() { *update = old })

	run(name)
}

type recordingTB struct {
	testing.TB
	name   string
	failed bool
}

func (tb *recordingTB) Helper() {}

func (tb *recordingTB) Name() string { return tb.name }

func (tb *recordingTB) Fatal(args ...any) { tb.failed = true }

func (tb *recordingTB) Fatalf(format string, args ...any) { tb.failed = true }
