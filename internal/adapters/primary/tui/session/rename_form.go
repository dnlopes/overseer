package session

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	domainsession "github.com/dnlopes/overseer/internal/core/domain/session"
	servicesession "github.com/dnlopes/overseer/internal/core/service/session"
)

type RenameFormModel struct {
	nameInput      textinput.Model
	errMsg         string
	currentSession domainsession.Session
	renameUC       *servicesession.RenameUseCase
	styles         *styles.Styles
}

func NewRenameForm(s *styles.Styles, renameUC *servicesession.RenameUseCase, current domainsession.Session) RenameFormModel {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.SetWidth(36)
	ti.SetStyles(textinput.Styles{})
	ti.SetVirtualCursor(false)
	ti.SetValue(current.Name)
	ti.Focus()

	return RenameFormModel{
		nameInput:      ti,
		currentSession: current,
		renameUC:       renameUC,
		styles:         s,
	}
}

func (m RenameFormModel) Init() tea.Cmd { return textinput.Blink }

func (m RenameFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelFormMsg{} }
		case "enter", "ctrl+j":
			name := strings.TrimSpace(m.nameInput.Value())
			if name == "" {
				m.errMsg = "name cannot be empty"
				return m, nil
			}
			resp, err := m.renameUC.Execute(context.Background(), servicesession.RenameRequest{
				ID:      m.currentSession.ID,
				NewName: name,
			})
			if err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return SessionRenamedMsg{Session: resp.Session} }
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m RenameFormModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.styles.Form.Field.Label.Render("Name") + "\n")
	b.WriteString(m.nameInput.View() + "\n")
	if m.errMsg != "" {
		b.WriteString(m.styles.Form.Field.Error.Render(m.errMsg) + "\n")
	}
	b.WriteString(m.styles.Help.Description.Render("Enter: submit  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
}
