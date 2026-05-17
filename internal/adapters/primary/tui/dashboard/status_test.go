package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func TestStatus_Default(t *testing.T) {
	testutil.Setup(t)

	s := styles.New()
	m := newStatus(s)
	m.workdir = "/home/user/projects"

	output := testutil.StripANSI(m.View().Content)

	goldenPath := filepath.Join("testdata", "TestStatus_Default.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, []byte(output), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (re-run with UPDATE_GOLDEN=1 to generate): %v", err)
	}
	if string(data) != output {
		t.Errorf("golden mismatch\ngot:  %q\nwant: %q", output, string(data))
	}
}

func TestStatus_Truncate(t *testing.T) {
	testutil.Setup(t)

	s := styles.New()
	m := newStatus(s)
	m.workdir = "/home/user/projects/very-long-project-name-that-exceeds-the-width"
	m.width = 40

	output := testutil.StripANSI(m.View().Content)

	if !strings.Contains(output, "...") {
		t.Errorf("expected ellipsis in truncated output: %q", output)
	}

	got := lipgloss.Width(output)
	if got > 40 {
		t.Errorf("output display width %d exceeds terminal width 40: %q", got, output)
	}
}

func TestStatus_WindowResize(t *testing.T) {
	s := styles.New()
	m := newStatus(s)

	if m.width != 80 {
		t.Fatalf("expected initial width 80, got %d", m.width)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	resized := updated.(StatusModel)

	if resized.width != 120 {
		t.Errorf("expected width 120 after resize, got %d", resized.width)
	}
}
