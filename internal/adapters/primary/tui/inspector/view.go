package inspector

import (
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"
)

// View is one tab inside the inspector. Each view owns its own polling
// loop; the inspector merely routes messages and key events to the active
// view, and re-inits a view's loop when it becomes active or when the
// selected session changes.
type View interface {
	Label() string
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	Body() string
	SetSize(width, height int)
	SetSession(sessionID uuid.UUID)
}
