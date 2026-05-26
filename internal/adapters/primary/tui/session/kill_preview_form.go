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
	"github.com/dnlopes/overseer/internal/core/service"
)

const killPreviewPopupWidth = 80

type KillPreviewFormModel struct {
	sessionID       uuid.UUID
	sessionName     string
	previewKind     string
	sessionsService service.SessionService
	styles          *styles.Styles
}

func NewKillPreviewForm(s *styles.Styles, sessionsService service.SessionService, sessionID uuid.UUID, sessionName string, previewKind string) KillPreviewFormModel {
	return KillPreviewFormModel{
		sessionID:       sessionID,
		sessionName:     sessionName,
		previewKind:     previewKind,
		sessionsService: sessionsService,
		styles:          s,
	}
}

func (m KillPreviewFormModel) Init() tea.Cmd { return nil }

func (m KillPreviewFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, popupCloseKeyBinding) || key.Matches(msg, deleteCancelKeyBinding) {
			return m, shared.Emit(shared.KillPreviewPopupCloseMsg{})
		}
		if key.Matches(msg, deleteConfirmKeyBinding) {
			return m.submit()
		}
	case shared.PreviewSessionKilledMsg:
		if msg.Err != nil {
			return m, shared.Emit(shared.KillPreviewPopupCloseMsg{})
		}
		return m, shared.Emit(shared.KillPreviewPopupCloseMsg{})
	}
	return m, nil
}

func (m KillPreviewFormModel) submit() (tea.Model, tea.Cmd) {
	id := m.sessionID
	kind := service.PreviewKindShell
	if m.previewKind == "Agent" {
		kind = service.PreviewKindAgent
	}
	svc := m.sessionsService
	return m, func() tea.Msg {
		_, err := svc.KillPreviewSession(context.Background(), service.KillPreviewSessionRequest{
			ID:   id,
			Kind: kind,
		})
		return shared.PreviewSessionKilledMsg{Err: err}
	}
}

func (m KillPreviewFormModel) View() tea.View {
	danger := m.styles.Danger
	field := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(danger.Title.Render("Kill preview session"))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(field.Label.Render("Session: "))
	b.WriteString(field.LabelFocused.Render(m.sessionName))
	b.WriteByte('\n')
	b.WriteString(field.Label.Render("Preview: "))
	b.WriteString(field.LabelFocused.Render(m.previewKind))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(m.styles.Help.Description.Render("y/enter: confirm kill  n/esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), killPreviewPopupWidth, 0))
}
