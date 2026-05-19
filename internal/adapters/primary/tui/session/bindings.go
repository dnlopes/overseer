package session

import "charm.land/bubbles/v2/key"

var (
	popupNextFieldKeyBinding    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field"))
	popupPrevFieldKeyBinding    = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field"))
	popupSubmitFormKeyBinding   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "create session"))
	popupCloseKeyBinding        = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	popupSelectorNextKeyBinding = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next project"))
	popupSelectorPrevKeyBinding = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "previous project"))

	jumpToSessionKeyBinding = key.NewBinding(
		key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
		key.WithHelp("1-9", "jump to session"),
	)
)
