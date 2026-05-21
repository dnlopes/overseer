package shared

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

type SessionCreatedMsg struct{ Session domain.Session }

type SessionSelectedMsg struct{ Session domain.Session }

type SessionsLoadedMsg struct {
	Sessions []domain.Session
	Err      error
}

type SessionReorderedMsg struct {
	Sessions []domain.Session
	FocusID  string
	Err      error
}

type NewSessionPopupCloseMsg struct{}

type SessionCreateErrMsg struct{ Err error }

type SessionAttachReadyMsg struct {
	Command *exec.Cmd
	Err     error
}

type SessionAttachedMsg struct{ Err error }

type SessionEditorLaunchedMsg struct{ Err error }

type SessionDeleteRequestedMsg struct{ Session domain.Session }

type SessionDeletedMsg struct{}

type SessionDeleteErrMsg struct{ Err error }

type NewSessionDeletePopupCloseMsg struct{}

type ProjectsLoadedMsg struct {
	Projects []domain.Project
	Err      error
}

type ProjectRegisteredMsg struct{ Project domain.Project }

type ProjectRegisterErrMsg struct{ Err error }

type JobsTickMsg struct{ JobID string }

type JobsBatchMsg struct{ Cmds []tea.Cmd }

type PRStatusUpdatedMsg struct {
	SessionID uuid.UUID
	PR        domain.PullRequest
	Err       error
}
