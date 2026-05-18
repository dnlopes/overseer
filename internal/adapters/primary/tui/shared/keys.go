package shared

import "charm.land/bubbles/v2/key"

type SessionsKeyMap struct {
	Up, Down, NewSession key.Binding
}

var (
	NewSessionKey     = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session"))
	HelpMenuKey       = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help menu"))
	QuitKey           = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"), key.WithHelp("ctrl+c", "quit"))
	PopupCloseKey     = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close popup"))
	PopupConfirmKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	PopupNextFieldKey = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field"))
	PopupPrevFieldKey = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field"))
)

var SessionsListKeyMap = SessionsKeyMap{
	Up:         key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
	Down:       key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
	NewSession: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
}

// ShortHelp satisfies bubbles/help.KeyMap so the help bar can render these
func (k SessionsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NewSession}
}
func (k SessionsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down}, {k.NewSession}}
}
