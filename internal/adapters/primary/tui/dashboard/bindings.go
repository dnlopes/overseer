package dashboard

import (
	"charm.land/bubbles/v2/key"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/inspector"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/leftpane"
)

var (
	newSessionKeyBinding    = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session"))
	newProjectKeyBinding    = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new project"))
	nextPaneKeyBinding      = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane"))
	helpMenuKeyBinding      = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help menu"))
	quitKeyBinding          = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"), key.WithHelp("ctrl+c", "quit"))
	attachShellKeyBinding   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "attach shell"))
	attachAgentKeyBinding   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "attach agent"))
	openEditorKeyBinding    = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open editor"))
	deleteSessionKeyBinding = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete session"))

	sessionsTabKeyBindings  = []key.Binding{newSessionKeyBinding, attachShellKeyBinding, attachAgentKeyBinding, openEditorKeyBinding, deleteSessionKeyBinding, inspector.NextViewKeyBinding, inspector.PrevViewKeyBinding, leftpane.SessionsTabBinding(), leftpane.ProjectsTabBinding(), nextPaneKeyBinding, helpMenuKeyBinding, quitKeyBinding}
	projectsTabKeyBindings  = []key.Binding{newProjectKeyBinding, leftpane.SessionsTabBinding(), leftpane.ProjectsTabBinding(), nextPaneKeyBinding, helpMenuKeyBinding, quitKeyBinding}
	detailsPanelKeyBindings = []key.Binding{inspector.NextViewKeyBinding, inspector.PrevViewKeyBinding, nextPaneKeyBinding, helpMenuKeyBinding, quitKeyBinding}
)
