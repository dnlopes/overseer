package dashboard

import (
	"charm.land/bubbles/v2/key"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/inspector"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
)

var (
	newSessionKeyBinding  = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session"))
	helpMenuKeyBinding    = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help menu"))
	quitKeyBinding        = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quit"))
	attachKeyBinding = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "attach to preview"))
	openEditorKeyBinding  = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "open editor"))
	killPreviewSessionKeyBinding = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill preview session"))

	sessionsKeyBindings  = []key.Binding{newSessionKeyBinding, attachKeyBinding, openEditorKeyBinding, session.ReorderSessionUpKeyBinding, session.ReorderSessionDownKeyBinding, session.GoToNextGroupKeyBinding, session.GoToPrevGroupKeyBinding, session.DeleteSessionKeyBinding, session.RenameKeyBinding, session.CycleLabelKeyBinding}
	inspectorKeyBindings = []key.Binding{inspector.ToggleViewKeyBinding, killPreviewSessionKeyBinding}
	generalKeyBindings   = []key.Binding{helpMenuKeyBinding, quitKeyBinding}

	sessionsHelpGroups = []shared.HelpPopupGroup{
		{
			Title:    "Sessions",
			Bindings: sessionsKeyBindings,
		},
		{
			Title:    "Inspector",
			Bindings: inspectorKeyBindings,
		},
		{
			Title:    "General",
			Bindings: generalKeyBindings,
		},
	}
)
