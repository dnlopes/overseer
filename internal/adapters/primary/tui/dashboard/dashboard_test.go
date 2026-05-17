package dashboard

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	xgolden "github.com/charmbracelet/x/exp/golden"
	teatestv2 "github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newDashboard() Model {
	return New(styles.New(), nil, NewHelpRegistry())
}

func sizedDashboard(t *testing.T, width, height int) Model {
	t.Helper()
	m := newDashboard()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(Model)
}

func viewString(m Model) string {
	return testutil.StripANSI(m.viewString(m.View()))
}

func keyMsg(text string) tea.KeyPressMsg {
	switch text {
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	case "shift+tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	case "ctrl+c":
		return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
	default:
		return tea.KeyPressMsg(tea.Key{Code: []rune(text)[0], Text: text})
	}
}

func TestDashboard_Default80x24(t *testing.T) {
	testutil.Setup(t)
	m := sizedDashboard(t, 80, 24)
	out := viewString(m)

	if m.activePane != PaneSessions {
		t.Fatalf("activePane: want PaneSessions, got %d", m.activePane)
	}
	if !strings.Contains(out, "No sessions yet") {
		t.Fatalf("expected sessions empty state, got:\n%s", out)
	}
	if !strings.Contains(out, "Stub mode: preview not available.") {
		t.Fatalf("expected preview pane, got:\n%s", out)
	}
	xgolden.RequireEqual(t, []byte(out))
}

func TestDashboard_SessionsFocused(t *testing.T) {
	testutil.Setup(t)
	m := sizedDashboard(t, 80, 24)
	xgolden.RequireEqual(t, []byte(viewString(m)))
}

func TestDashboard_PreviewFocused(t *testing.T) {
	testutil.Setup(t)
	m := sizedDashboard(t, 80, 24)
	updated, _ := m.Update(keyMsg("tab"))
	m = updated.(Model)

	if m.activePane != PanePreview {
		t.Fatalf("activePane after Tab: want PanePreview, got %d", m.activePane)
	}
	xgolden.RequireEqual(t, []byte(viewString(m)))
}

func TestDashboard_TabCyclesFocus(t *testing.T) {
	m := sizedDashboard(t, 80, 24)

	updated, cmd := m.Update(keyMsg("tab"))
	if cmd != nil {
		t.Fatal("Tab should not produce a command")
	}
	m = updated.(Model)
	if m.activePane != PanePreview {
		t.Fatalf("after Tab: want PanePreview, got %d", m.activePane)
	}

	updated, _ = m.Update(keyMsg("tab"))
	m = updated.(Model)
	if m.activePane != PaneSessions {
		t.Fatalf("after second Tab: want PaneSessions, got %d", m.activePane)
	}
}

func TestDashboard_OpenCreate(t *testing.T) {
	testutil.Setup(t)
	m := sizedDashboard(t, 80, 24)

	updated, _ := m.Update(keyMsg("n"))
	m = updated.(Model)
	if m.createForm == nil {
		t.Fatal("expected createForm to be opened")
	}
	out := viewString(m)
	if !strings.Contains(out, "Project") || !strings.Contains(out, "Esc: cancel") {
		t.Fatalf("expected create form content, got:\n%s", out)
	}
	xgolden.RequireEqual(t, []byte(out))
}

func TestDashboard_TooSmall(t *testing.T) {
	testutil.Setup(t)
	m := sizedDashboard(t, 40, 10)
	out := viewString(m)

	if !m.tooSmall {
		t.Fatal("expected tooSmall flag")
	}
	if !strings.Contains(out, "Terminal too small") {
		t.Fatalf("expected too-small message, got:\n%s", out)
	}
	xgolden.RequireEqual(t, []byte(out))
}

func TestDashboard_Quit(t *testing.T) {
	m := sizedDashboard(t, 80, 24)

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should return tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q command: want tea.QuitMsg, got %T", cmd())
	}

	_, cmd = m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c should return tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c command: want tea.QuitMsg, got %T", cmd())
	}
}

func TestDashboard_1JumpToSessions(t *testing.T) {
	m := sizedDashboard(t, 80, 24)
	m.focus(PanePreview)

	updated, _ := m.Update(keyMsg("1"))
	m = updated.(Model)
	if m.activePane != PaneSessions {
		t.Fatalf("after 1: want PaneSessions, got %d", m.activePane)
	}
}

func TestDashboard_2JumpToPreview(t *testing.T) {
	m := sizedDashboard(t, 80, 24)

	updated, _ := m.Update(keyMsg("2"))
	m = updated.(Model)
	if m.activePane != PanePreview {
		t.Fatalf("after 2: want PanePreview, got %d", m.activePane)
	}
}

func TestDashboard_RWithNoSession(t *testing.T) {
	m := sizedDashboard(t, 80, 24)

	updated, _ := m.Update(keyMsg("r"))
	m = updated.(Model)
	if m.renameForm != nil {
		t.Fatal("expected renameForm to remain nil with no selected session")
	}
}

func TestDashboard_DefaultViaHarness(t *testing.T) {
	m := newDashboard()
	tm := testutil.NewHarness(t, m, 80, 24)

	teatestv2.WaitFor(t, tm.Output(), func(b []byte) bool {
		return strings.Contains(string(b), "j move down")
	}, teatestv2.WithDuration(time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatestv2.WithFinalTimeout(time.Second))
}
