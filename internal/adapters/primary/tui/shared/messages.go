package shared

import (
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/core/domain"
)

type SessionCreatedMsg struct{ Session domain.Session }

type SessionSelectedMsg struct{ Session domain.Session }

type SessionSelectionClearedMsg struct{}

type SessionsLoadedMsg struct {
	Sessions []domain.Session
	Err      error
}

type SessionReorderedMsg struct {
	Sessions []domain.Session
	FocusID  string
	Err      error
}

type SessionLabelCycledMsg struct {
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

type PreviewSessionKilledMsg struct{ Err error }

type KillPreviewPopupCloseMsg struct{}

type SessionDeleteRequestedMsg struct{ Session domain.Session }

type SessionDeletedMsg struct{}

type SessionDeleteErrMsg struct{ Err error }

type NewSessionDeletePopupCloseMsg struct{}

type SessionRenameRequestedMsg struct{ Session domain.Session }

type ProjectRenameRequestedMsg struct {
	ProjectID   uuid.UUID
	CurrentName string
}

type SessionRenamedMsg struct{ Session domain.Session }

type SessionRenameErrMsg struct{ Err error }

type ProjectRenamedMsg struct{ Project domain.Project }

type ProjectRenameErrMsg struct{ Err error }

type RenamePopupCloseMsg struct{}

type HelpPopupCloseMsg struct{}

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

type AgentStatusesUpdatedMsg struct {
	Statuses map[uuid.UUID]domain.AgentStatus
	Err      error
}

type BranchesLoadedMsg struct {
	ProjectID     uuid.UUID
	Branches      []domain.BranchInfo
	DefaultBranch string
	LoadedAt      time.Time
	Err           error
}

type BranchCacheTickMsg struct{}

type ProjectCurrentBranchLoadedMsg struct {
	ProjectID uuid.UUID
	Branch    string
	Err       error
}
