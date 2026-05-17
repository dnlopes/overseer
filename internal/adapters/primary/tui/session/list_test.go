package session

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	xgolden "github.com/charmbracelet/x/exp/golden"
	teatestv2 "github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/testutil"
)

func twoGroupsModel() Model {
	m := New(styles.New(), nil)
	m.groups = []SessionGroup{
		{
			ProjectName: "alpha",
			Sessions: []domain.Session{
				{Name: "session-one"},
			},
		},
		{
			ProjectName: "beta",
			Sessions: []domain.Session{
				{Name: "session-two"},
				{Name: "session-three"},
			},
		},
	}
	return m
}

func TestList_Empty(t *testing.T) {
	testutil.Setup(t)
	m := New(styles.New(), nil)
	xgolden.RequireEqual(t, []byte(testutil.StripANSI(m.render())))
}

func TestList_TwoGroups(t *testing.T) {
	testutil.Setup(t)
	m := twoGroupsModel()
	xgolden.RequireEqual(t, []byte(testutil.StripANSI(m.render())))
}

func TestList_CursorDown(t *testing.T) {
	m := twoGroupsModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	got := updated.(Model)

	if got.cursor != 1 {
		t.Fatalf("cursor after j: want 1, got %d", got.cursor)
	}
}

func TestList_CursorDownViaHarness(t *testing.T) {
	testutil.Setup(t)
	m := twoGroupsModel()

	tm := testutil.NewHarness(t, m, 80, 24)
	tm.Type("j")
	tm.Type("j")

	teatestv2.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return len(bts) > 0
		},
		teatestv2.WithDuration(time.Second),
	)

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatestv2.WithFinalTimeout(time.Second))

	fm, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatal("FinalModel is not a session.Model")
	}
	if fm.cursor != 2 {
		t.Fatalf("cursor after 2×j: want 2, got %d", fm.cursor)
	}
}

func TestList_CursorBoundary(t *testing.T) {
	m := twoGroupsModel()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	got := updated.(Model)

	if got.cursor != 2 {
		t.Fatalf("cursor past last item: want 2, got %d", got.cursor)
	}
}

func TestList_CursorUp(t *testing.T) {
	m := twoGroupsModel()
	m.cursor = 2

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	got := updated.(Model)

	if got.cursor != 1 {
		t.Fatalf("cursor after k: want 1, got %d", got.cursor)
	}
}

func TestList_FocusedBorder(t *testing.T) {
	testutil.Setup(t)
	m := twoGroupsModel()
	m.SetFocus(true)

	blurred := New(styles.New(), nil)
	blurred.groups = m.groups

	focusedOut := m.render()
	blurredOut := blurred.render()

	if focusedOut == blurredOut {
		t.Fatal("focused and blurred renders must differ")
	}
}

func TestList_SelectedSession(t *testing.T) {
	m := twoGroupsModel()
	m.cursor = 1

	s, ok := m.SelectedSession()
	if !ok {
		t.Fatal("expected a selected session")
	}
	if s.Name != "session-two" {
		t.Fatalf("selected session name: want session-two, got %q", s.Name)
	}
}

func TestList_Keybindings(t *testing.T) {
	m := New(styles.New(), nil)
	bindings := m.Keybindings()

	if len(bindings) < 4 {
		t.Fatalf("Keybindings: want ≥4, got %d", len(bindings))
	}

	keys := map[string]bool{}
	for _, b := range bindings {
		for _, k := range b.Keys() {
			keys[k] = true
		}
	}
	for _, required := range []string{"j", "k", "J", "K"} {
		if !keys[required] {
			t.Fatalf("missing keybinding: %q", required)
		}
	}
}
