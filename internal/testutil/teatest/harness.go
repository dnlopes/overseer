package teatest

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	teatest "github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/dnlopes/overseer/internal/testutil/golden"
)

// NewHarness creates a teatest.TestModel with fixed terminal size and ANSI stripping.
func NewHarness(t *testing.T, model tea.Model, width, height int) *teatest.TestModel {
	t.Helper()
	golden.Setup(t)
	return teatest.NewTestModel(t, model, teatest.WithInitialTermSize(width, height))
}
