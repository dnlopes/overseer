package inspector

import "github.com/google/uuid"

type viewKind int

const (
	viewKindAgent viewKind = iota
	viewKindShell
)

// previewCapturedMsg carries the result of a single tmux capture-pane call.
// kind identifies which view produced the message so inactive views ignore
// messages that arrive after a view switch. sessionID lets a view drop
// messages that arrived after the selected session changed, breaking the
// stale polling chain.
type previewCapturedMsg struct {
	kind         viewKind
	sessionID    uuid.UUID
	content      string
	sessionReady bool
	err          error
}
