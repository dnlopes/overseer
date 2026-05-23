package shared

import (
	"strings"
	"testing"
)

func TestLastNonEmptyLines_ReturnsLastN(t *testing.T) {
	in := "a\nb\nc\nd\ne\n"
	got := LastNonEmptyLines(in, 3)
	want := []string{"c", "d", "e"}
	assertEqualLines(t, got, want)
}

func TestLastNonEmptyLines_SkipsBlankAndWhitespaceLines(t *testing.T) {
	in := "a\n\n   \nb\n\t\nc\n\n"
	got := LastNonEmptyLines(in, 5)
	want := []string{"a", "b", "c"}
	assertEqualLines(t, got, want)
}

func TestLastNonEmptyLines_NSmallerThanContent_OnlyTail(t *testing.T) {
	in := "a\nb\nc\nd\ne\nf\n"
	got := LastNonEmptyLines(in, 2)
	want := []string{"e", "f"}
	assertEqualLines(t, got, want)
}

func TestLastNonEmptyLines_NLargerThanContent_ReturnsAll(t *testing.T) {
	in := "a\nb\n"
	got := LastNonEmptyLines(in, 10)
	want := []string{"a", "b"}
	assertEqualLines(t, got, want)
}

func TestLastNonEmptyLines_EmptyInput_ReturnsEmpty(t *testing.T) {
	if got := LastNonEmptyLines("", 5); len(got) != 0 {
		t.Fatalf("LastNonEmptyLines(\"\", 5) = %v, want empty", got)
	}
}

func TestLastNonEmptyLines_ZeroN_ReturnsEmpty(t *testing.T) {
	if got := LastNonEmptyLines("a\nb\n", 0); len(got) != 0 {
		t.Fatalf("LastNonEmptyLines(_, 0) = %v, want empty", got)
	}
}

func TestLastNonEmptyLines_NegativeN_ReturnsEmpty(t *testing.T) {
	if got := LastNonEmptyLines("a\nb\n", -1); len(got) != 0 {
		t.Fatalf("LastNonEmptyLines(_, -1) = %v, want empty", got)
	}
}

func TestLastNonEmptyLines_PreservesLineOrderInOutput(t *testing.T) {
	in := "first\nsecond\nthird\n"
	got := LastNonEmptyLines(in, 3)
	if strings.Join(got, "|") != "first|second|third" {
		t.Fatalf("LastNonEmptyLines preserved wrong order: %v", got)
	}
}

func assertEqualLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("lines len = %d, want %d (got=%v want=%v)", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
