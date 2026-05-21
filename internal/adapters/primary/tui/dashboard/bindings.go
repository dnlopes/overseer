package dashboard

import (
	"charm.land/bubbles/v2/key"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/inspector"
)

var (
	newSessionKeyBinding    = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session"))
	checkoutBranchKeyBinding    = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "checkout branch"))
	helpMenuKeyBinding      = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help menu"))
	quitKeyBinding          = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"), key.WithHelp("ctrl+c", "quit"))
	attachShellKeyBinding   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "attach shell"))
	attachAgentKeyBinding   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "attach agent"))
	openEditorKeyBinding    = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "open editor"))
	deleteSessionKeyBinding = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete session"))

	sessionsKeyBindings = []key.Binding{newSessionKeyBinding, checkoutBranchKeyBinding, attachShellKeyBinding, attachAgentKeyBinding, openEditorKeyBinding, deleteSessionKeyBinding, inspector.ToggleViewKeyBinding, helpMenuKeyBinding, quitKeyBinding}
)
