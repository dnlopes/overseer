package testutil

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

type dummyModel struct{}

func (dummyModel) Init() tea.Cmd { return nil }

func (m dummyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (dummyModel) View() tea.View { return tea.NewView("") }

func TestNewHarness(t *testing.T) {
	h := NewHarness(t, dummyModel{}, 80, 24)
	if h == nil {
		t.Fatal("expected harness")
	}
}
