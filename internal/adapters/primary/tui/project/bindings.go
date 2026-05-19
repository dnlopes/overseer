package project

import "charm.land/bubbles/v2/key"

var (
	popupNextFieldKeyBinding  = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field"))
	popupPrevFieldKeyBinding  = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field"))
	popupSubmitFormKeyBinding = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "register project"))
	popupCloseKeyBinding      = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))

	jumpToProjectKeyBinding = key.NewBinding(
		key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
		key.WithHelp("1-9", "jump to project"),
	)
)
