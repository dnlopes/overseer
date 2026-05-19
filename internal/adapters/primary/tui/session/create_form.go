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
	sessionsService service.SessionService
	styles          *styles.Styles
}

func NewCreateForm(s *styles.Styles, sessionsService service.SessionService) CreateFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(36)
	nameInput.SetStyles(textinput.Styles{})
	nameInput.SetVirtualCursor(false)
	nameInput.Focus()

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name"
	projectInput.CharLimit = 100
	projectInput.SetWidth(36)
	projectInput.SetStyles(textinput.Styles{})
	projectInput.SetVirtualCursor(false)
	projectInput.Blur()

	return CreateFormModel{
		nameInput:       nameInput,
		projectInput:    projectInput,
		focusIndex:      shared.NewCircularInt(0, 1),
		sessionsService: sessionsService,
		styles:          s,
	}
}

func (m CreateFormModel) Init() tea.Cmd {
	return nil
}

func (m CreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, shared.PopupCloseKey) {
			return m, func() tea.Msg { return shared.NewSessionPopupCloseMsg{} }
		}
		if key.Matches(msg, shared.PopupConfirmKey) {
			return m.submit()
		}
		if key.Matches(msg, shared.PopupNextFieldKey) {
			m.focusIndex.Increment()
			m.updateFocusAndBlurs()
			return m, nil
		}
		if key.Matches(msg, shared.PopupPrevFieldKey) {
			m.focusIndex.Decrement()
			m.updateFocusAndBlurs()
			return m, nil
		}
	}

	if msg, ok := msg.(shared.SessionCreateErrMsg); ok {
		m.errMsg = msg.Err.Error()
		return m, nil
	}

	if msg, ok := msg.(shared.SessionCreatedMsg); ok {
		return m, tea.Batch(
			shared.Emit(msg),
			shared.Emit(shared.NewSessionPopupCloseMsg{}),
		)
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
			return shared.SessionCreateErrMsg{Err: err}
		}
		return shared.SessionCreatedMsg{Session: resp.Session}
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
