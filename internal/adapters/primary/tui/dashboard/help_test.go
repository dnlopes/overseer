package dashboard

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/testutil"
)

func TestHelpRegistry_RegisterAndRetrieve(t *testing.T) {
	reg := NewHelpRegistry()
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down")),
		key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "up")),
	}
	reg.RegisterPane("sessions", bindings)

	got := reg.BindingsFor("sessions")
	if len(got) < 2 {
		t.Fatalf("expected at least 2 bindings, got %d", len(got))
	}

	foundJ := false
	for _, b := range got {
		if b.Help().Key == "j" {
			foundJ = true
			break
		}
	}
	if !foundJ {
		t.Error("expected 'j' binding in BindingsFor(sessions)")
	}
}

func TestHelpRegistry_GlobalsAlwaysPresent(t *testing.T) {
	reg := NewHelpRegistry()

	got := reg.BindingsFor("nonexistent-pane")
	if len(got) == 0 {
		t.Fatal("expected global bindings for unknown pane, got none")
	}

	foundQuit := false
	for _, b := range got {
		if b.Help().Key == "q" {
			foundQuit = true
			break
		}
	}
	if !foundQuit {
		t.Error("expected 'q' quit binding in globals")
	}
}

func TestHelp_ActivePaneOnly(t *testing.T) {
	reg := NewHelpRegistry()
	reg.RegisterPane("sessions", []key.Binding{
		key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down")),
	})
	reg.RegisterPane("preview", []key.Binding{
		key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "scroll up")),
	})

	bar := newHelpBar(reg, nil)
	bar.SetActivePane("sessions")

	out := testutil.StripANSI(bar.View().Content)

	if !strings.Contains(out, "j") {
		t.Errorf("expected 'j' in view for sessions pane, got:\n%s", out)
	}
	if strings.Contains(out, "pgup") {
		t.Errorf("expected 'pgup' absent when preview pane is inactive, got:\n%s", out)
	}
}

func TestHelp_Toggle(t *testing.T) {
	reg := NewHelpRegistry()
	reg.RegisterPane("sessions", []key.Binding{
		key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "down")),
	})

	bar := newHelpBar(reg, nil)
	bar.SetActivePane("sessions")

	shortOut := testutil.StripANSI(bar.View().Content)

	updated, _ := bar.Update(tea.KeyPressMsg(tea.Key{Code: '?', Text: "?"}))
	bar = updated.(HelpModel)

	fullOut := testutil.StripANSI(bar.View().Content)

	if shortOut == fullOut {
		t.Errorf("expected view to change after toggling help\nshort: %q\nfull:  %q", shortOut, fullOut)
	}
}
