package testutil

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	teatest "github.com/charmbracelet/x/exp/teatest/v2"
)

// NewHarness creates a teatest.TestModel with fixed terminal size and golden setup.
func NewHarness(t *testing.T, model tea.Model, width, height int) *teatest.TestModel {
	t.Helper()
	Setup(t)
	return teatest.NewTestModel(t, model, teatest.WithInitialTermSize(width, height))
}
