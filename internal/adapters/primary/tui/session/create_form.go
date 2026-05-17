package session

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

type CreateFormModel struct {
	nameInput    textinput.Model
	projectInput textinput.Model
	focusIndex   int
	errMsg       string
	svc          *service.SessionService
	styles       *styles.Styles
}

func NewCreateForm(s *styles.Styles, svc *service.SessionService) CreateFormModel {
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

	return CreateFormModel{
		nameInput:    nameInput,
		projectInput: projectInput,
		focusIndex:   0,
		svc:          svc,
		styles:       s,
	}
}

func (m CreateFormModel) Init() tea.Cmd { return textinput.Blink }

type createErrMsg struct{ err error }

func (m CreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelFormMsg{} }
		case "tab", "shift+tab":
			m.focusIndex = (m.focusIndex + 1) % 2
			if m.focusIndex == 0 {
				m.nameInput.Focus()
				m.projectInput.Blur()
				return m, nil
			}
			m.projectInput.Focus()
			m.nameInput.Blur()
			return m, nil
		case "enter", "ctrl+j":
			return m.submit()
		}
	case createErrMsg:
		m.errMsg = msg.err.Error()
		return m, nil
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.projectInput, cmd = m.projectInput.Update(msg)
	}
	return m, cmd
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
	svc := m.svc
	req := service.CreateSessionRequest{Name: name, ProjectName: project}
	return m, func() tea.Msg {
		resp, err := svc.Create(context.Background(), req)
		if err != nil {
			return createErrMsg{err: err}
		}
		return SessionCreatedMsg{Session: resp.Session}
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
