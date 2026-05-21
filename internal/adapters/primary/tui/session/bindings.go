package session

import "charm.land/bubbles/v2/key"

const jumpRowDelta = 5

var (
	popupNextFieldKeyBinding    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field"))
	popupPrevFieldKeyBinding    = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field"))
	popupSubmitFormKeyBinding   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "create session"))
	popupCloseKeyBinding        = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	popupSelectorNextKeyBinding = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next option"))
	popupSelectorPrevKeyBinding = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "previous option"))

	repoPickerEnterPasteKeyBinding = key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "paste new path"))
	repoPickerExitPasteKeyBinding  = key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "back to list"))

	jumpUpKeyBinding      = key.NewBinding(key.WithKeys("ctrl+up"), key.WithHelp("ctrl+↑", "jump up"))
	jumpDownKeyBinding    = key.NewBinding(key.WithKeys("ctrl+down"), key.WithHelp("ctrl+↓", "jump down"))
	reorderUpKeyBinding   = key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "move row up"))
	reorderDownKeyBinding = key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "move row down"))
	nextGroupKeyBinding   = key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "next group"))
	prevGroupKeyBinding   = key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "previous group"))

	deleteSessionKeyBinding = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete session"))
	deleteConfirmKeyBinding = key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y/enter", "confirm delete"))
	deleteCancelKeyBinding  = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel"))

	cycleLabelKeyBinding = key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "cycle label"))
)
