package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewLabel_CreatesLabel(t *testing.T) {
	l, err := NewLabel("WIP", "#F59E0B", "🚧")

	if err != nil {
		t.Fatalf("NewLabel() error = %v", err)
	}
	if l.Code != "WIP" {
		t.Errorf("NewLabel() Code = %q, want %q", l.Code, "WIP")
	}
	if l.Color != "#F59E0B" {
		t.Errorf("NewLabel() Color = %q, want %q", l.Color, "#F59E0B")
	}
	if l.Glyph != "🚧" {
		t.Errorf("NewLabel() Glyph = %q, want %q", l.Glyph, "🚧")
	}
}

func TestNewLabel_EmptyGlyphIsAllowed(t *testing.T) {
	l, err := NewLabel("plain", "#000000", "")

	if err != nil {
		t.Fatalf("NewLabel() error = %v, want nil for empty glyph", err)
	}
	if l.Glyph != "" {
		t.Errorf("NewLabel() Glyph = %q, want empty", l.Glyph)
	}
}

func TestNewLabel_TrimsWhitespace(t *testing.T) {
	l, err := NewLabel("  ready  ", "  #60A5FA  ", "  📦  ")

	if err != nil {
		t.Fatalf("NewLabel() error = %v", err)
	}
	if l.Code != "ready" {
		t.Errorf("NewLabel() Code = %q, want trimmed %q", l.Code, "ready")
	}
	if l.Color != "#60A5FA" {
		t.Errorf("NewLabel() Color = %q, want trimmed %q", l.Color, "#60A5FA")
	}
	if l.Glyph != "📦" {
		t.Errorf("NewLabel() Glyph = %q, want trimmed %q", l.Glyph, "📦")
	}
}

func TestNewLabel_RejectsEmptyCode(t *testing.T) {
	_, err := NewLabel("   ", "#000000", "")

	if !errors.Is(err, ErrLabelEmptyCode) {
		t.Errorf("NewLabel() error = %v, want %v", err, ErrLabelEmptyCode)
	}
}

func TestNewLabel_RejectsTooLongCode(t *testing.T) {
	code := strings.Repeat("a", 51)

	_, err := NewLabel(code, "#000000", "")

	if !errors.Is(err, ErrLabelCodeTooLong) {
		t.Errorf("NewLabel() error = %v, want %v", err, ErrLabelCodeTooLong)
	}
}

func TestNewLabel_RejectsEmptyColor(t *testing.T) {
	_, err := NewLabel("WIP", "   ", "")

	if !errors.Is(err, ErrLabelEmptyColor) {
		t.Errorf("NewLabel() error = %v, want %v", err, ErrLabelEmptyColor)
	}
}

func TestNewLabel_RejectsTooLongGlyph(t *testing.T) {
	glyph := strings.Repeat("x", 21)

	_, err := NewLabel("ok", "#000000", glyph)

	if !errors.Is(err, ErrLabelGlyphTooLong) {
		t.Errorf("NewLabel() error = %v, want %v", err, ErrLabelGlyphTooLong)
	}
}

func TestDefaultLabels_FiveBuiltInsInCanonicalOrder(t *testing.T) {
	wantCodes := []string{"WIP", "reviewing", "ready", "done", "draft"}

	if len(DefaultLabels) != len(wantCodes) {
		t.Fatalf("DefaultLabels length = %d, want %d", len(DefaultLabels), len(wantCodes))
	}
	for i, want := range wantCodes {
		if DefaultLabels[i].Code != want {
			t.Errorf("DefaultLabels[%d].Code = %q, want %q", i, DefaultLabels[i].Code, want)
		}
		if DefaultLabels[i].Color == "" {
			t.Errorf("DefaultLabels[%d].Color is empty, want non-empty hex color", i)
		}
		if DefaultLabels[i].Glyph == "" {
			t.Errorf("DefaultLabels[%d].Glyph is empty, want non-empty default glyph", i)
		}
	}
}

func TestDefaultLabels_AllValidViaNewLabel(t *testing.T) {
	for i, l := range DefaultLabels {
		if _, err := NewLabel(l.Code, l.Color, l.Glyph); err != nil {
			t.Errorf("DefaultLabels[%d] = %+v failed NewLabel validation: %v", i, l, err)
		}
	}
}

func TestNextLabelCode_EmptyCurrent_ReturnsFirst(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}, {Code: "b", Color: "#000"}}

	got := NextLabelCode("", labels)

	if got != "a" {
		t.Errorf("NextLabelCode(\"\", ...) = %q, want %q", got, "a")
	}
}

func TestNextLabelCode_MiddleCurrent_ReturnsNext(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}, {Code: "b", Color: "#000"}, {Code: "c", Color: "#aaa"}}

	got := NextLabelCode("b", labels)

	if got != "c" {
		t.Errorf("NextLabelCode(\"b\", ...) = %q, want %q", got, "c")
	}
}

func TestNextLabelCode_LastCurrent_ReturnsEmpty(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}, {Code: "b", Color: "#000"}}

	got := NextLabelCode("b", labels)

	if got != "" {
		t.Errorf("NextLabelCode(\"b\", ...) = %q, want empty string (clear)", got)
	}
}

func TestNextLabelCode_StaleCurrent_ReturnsFirst(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}, {Code: "b", Color: "#000"}}

	got := NextLabelCode("removed-from-config", labels)

	if got != "a" {
		t.Errorf("NextLabelCode(\"removed-from-config\", ...) = %q, want %q (reset to first when stale)", got, "a")
	}
}

func TestNextLabelCode_EmptyLabelsList_ReturnsEmpty(t *testing.T) {
	got := NextLabelCode("anything", nil)

	if got != "" {
		t.Errorf("NextLabelCode(_, nil) = %q, want empty string", got)
	}
}

func TestFindLabel_Match_ReturnsLabel(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}, {Code: "b", Color: "#000"}}

	got, ok := FindLabel("b", labels)

	if !ok {
		t.Fatal("FindLabel() ok = false, want true")
	}
	if got.Color != "#000" {
		t.Errorf("FindLabel() Color = %q, want %q", got.Color, "#000")
	}
}

func TestFindLabel_Miss_ReturnsZero(t *testing.T) {
	labels := []Label{{Code: "a", Color: "#fff"}}

	_, ok := FindLabel("missing", labels)

	if ok {
		t.Error("FindLabel() ok = true, want false")
	}
}
