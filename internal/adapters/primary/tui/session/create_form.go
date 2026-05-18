package session

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

const (
	FieldNameSelectedIndex int = iota
	FieldProjectSelectedIndex
)

type CreateFormModel struct {
	nameInput       textinput.Model
	projectInput    textinput.Model
	focusIndex      shared.CircularInt
	errMsg          string
	sessionsService *service.SessionService
	styles          *styles.Styles
}

func NewCreateForm(s *styles.Styles, sessionsService *service.SessionService) CreateFormModel {
	return CreateFormModel{nameInput: textinput.New(), projectInput: textinput.New(), focusIndex: shared.NewCircularInt(0, 1), sessionsService: sessionsService, styles: s}
}

func (m CreateFormModel) Init() tea.Cmd {
	m.nameInput.Placeholder = "Session name"
	m.nameInput.CharLimit = 100
	m.nameInput.SetWidth(36)
	m.nameInput.SetStyles(textinput.Styles{})
	m.nameInput.SetVirtualCursor(false)
	m.nameInput.Focus()
	m.projectInput.Placeholder = "Project name"
	m.projectInput.CharLimit = 100
	m.projectInput.SetWidth(36)
	m.projectInput.SetStyles(textinput.Styles{})
	m.projectInput.SetVirtualCursor(false)
	m.projectInput.Blur()

	return nil
}

type createErrMsg struct{ err error }

func (m CreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, shared.PopupCloseKey) {
			return m, func() tea.Msg { return CancelFormMsg{} }
		}
		if key.Matches(msg, shared.PopupConfirmKey) {
			return m.submit()
		}
		if key.Matches(msg, shared.PopupNextFieldKey) {
			m.focusIndex.Increment()
			m.updateFocusAndBlurs()
		}
		if key.Matches(msg, shared.PopupPrevFieldKey) {
			m.focusIndex.Decrement()
			m.updateFocusAndBlurs()
		}
	}

	switch m.focusIndex.Value() {
	case FieldNameSelectedIndex:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	case FieldProjectSelectedIndex:
		var cmd tea.Cmd
		m.projectInput, cmd = m.projectInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CreateFormModel) submit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	project := strings.TrimSpace(m.projectInput.Value())

	if name == "" {
		m.errMsg = "session name is required"
		return m, nil
	}
	if project == "" {
		m.errMsg = "project name is required"
		return m, nil
	}

	m.errMsg = ""
	req := service.CreateSessionRequest{Name: name, ProjectName: project}
	return m, func() tea.Msg {
		resp, err := m.sessionsService.Create(context.Background(), req)
		if err != nil {
			return createErrMsg{err: err}
		}
		return SessionCreatedMsg{Session: resp.Session}
	}
}

func (m CreateFormModel) updateFocusAndBlurs() {
	switch m.focusIndex.Value() {
	case FieldNameSelectedIndex:
		m.nameInput.Focus()
		m.projectInput.Blur()
	case FieldProjectSelectedIndex:
		m.projectInput.Focus()
		m.nameInput.Blur()
	}
}

func (m CreateFormModel) View() tea.View {
	s := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(s.Label.Render("Name"))
	b.WriteByte('\n')
	b.WriteString(m.nameInput.View())
	b.WriteByte('\n')
	b.WriteString(s.Label.Render("Project"))
	b.WriteByte('\n')
	b.WriteString(m.projectInput.View())
	b.WriteByte('\n')
	if m.errMsg != "" {
		b.WriteString(s.Error.Render(m.errMsg))
		b.WriteByte('\n')
	}
	b.WriteString(m.styles.Help.Description.Render("Tab: next field  Enter: submit  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
}
