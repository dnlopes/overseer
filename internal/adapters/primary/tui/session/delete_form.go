package session

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

const (
	deletePopupWidth = 80
)

type DeleteFormModel struct {
	sessionID       uuid.UUID
	sessionName     string
	hasWorktree     bool
	errMsg          string
	sessionsService service.SessionService
	styles          *styles.Styles
}

func NewDeleteForm(s *styles.Styles, sessionsService service.SessionService, sess domain.Session) DeleteFormModel {
	return DeleteFormModel{
		sessionID:       sess.ID,
		sessionName:     sess.Name,
		hasWorktree:     sess.HasWorktree(),
		sessionsService: sessionsService,
		styles:          s,
	}
}

func (m DeleteFormModel) Init() tea.Cmd { return nil }

func (m DeleteFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, popupCloseKeyBinding) || key.Matches(msg, deleteCancelKeyBinding) {
			return m, shared.Emit(shared.NewSessionDeletePopupCloseMsg{})
		}
		if key.Matches(msg, deleteConfirmKeyBinding) {
			return m.submit()
		}
	case shared.SessionDeleteErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil
	}
	return m, nil
}

func (m DeleteFormModel) submit() (tea.Model, tea.Cmd) {
	id := m.sessionID
	svc := m.sessionsService
	return m, func() tea.Msg {
		_, err := svc.Delete(context.Background(), service.DeleteSessionRequest{ID: id})
		if err != nil {
			return shared.SessionDeleteErrMsg{Err: err}
		}
		return shared.SessionDeletedMsg{}
	}
}

func (m DeleteFormModel) View() tea.View {
	field := m.styles.Form.Field
	danger := m.styles.Danger

	var b strings.Builder
	b.WriteString(danger.Title.Render("Delete session"))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(field.Label.Render("Session: "))
	b.WriteString(field.LabelFocused.Render(m.sessionName))
	b.WriteByte('\n')
	b.WriteString(m.consequencesHint())
	b.WriteByte('\n')
	b.WriteString(field.Error.Render(m.errMsg))
	b.WriteByte('\n')
	if m.errMsg != "" {
		b.WriteByte('\n')
	}
	b.WriteString(m.styles.Help.Description.Render("y/enter: confirm delete  n/esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), deletePopupWidth, 0))
}

func (m DeleteFormModel) consequencesHint() string {
	if m.hasWorktree {
		return m.styles.Danger.Body.Render("This will kill the tmux session, remove the git worktree (uncommitted changes lost), and delete the session record. This cannot be undone.")
	}
	return m.styles.Danger.Body.Render("This will kill the tmux session and delete the session record. This cannot be undone.")
}
