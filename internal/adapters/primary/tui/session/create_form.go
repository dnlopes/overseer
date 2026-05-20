package session

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

const (
	FieldNameSelectedIndex int = iota
	FieldProjectSelectedIndex
	FieldLauncherSelectedIndex
	FieldEditorSelectedIndex
)

type CreateFormModel struct {
	nameInput       textinput.Model
	projectIdx      int
	projects        []domain.Project
	launchers       []domain.Launcher
	launcherIdx     int
	editors         []domain.Editor
	editorIdx       int
	focusIndex      shared.CircularInt
	errMsg          string
	sessionsService service.SessionService
	styles          *styles.Styles
}

// NewCreateForm builds the session-create form with the configured launcher and editor choices.
func NewCreateForm(s *styles.Styles, sessionsService service.SessionService, projects []domain.Project, launchers []domain.Launcher, editors []domain.Editor) CreateFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(36)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	return CreateFormModel{
		nameInput:       nameInput,
		projectIdx:      0,
		projects:        projects,
		launchers:       launchers,
		launcherIdx:     0,
		editors:         editors,
		editorIdx:       0,
		focusIndex:      shared.NewCircularInt(0, 3),
		sessionsService: sessionsService,
		styles:          s,
	}
}

func (m CreateFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, popupCloseKeyBinding) {
			return m, shared.Emit(shared.NewSessionPopupCloseMsg{})
		}
		if key.Matches(msg, popupSubmitFormKeyBinding) {
			return m.submit()
		}
		if key.Matches(msg, popupNextFieldKeyBinding) {
			m.focusIndex.Increment()
			m.updateFocusAndBlurs()
			return m, nil
		}
		if key.Matches(msg, popupPrevFieldKeyBinding) {
			m.focusIndex.Decrement()
			m.updateFocusAndBlurs()
			return m, nil
		}
		if m.focusIndex.Value() == FieldProjectSelectedIndex {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleProject(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleProject(-1)
				return m, nil
			}
		}
		if m.focusIndex.Value() == FieldLauncherSelectedIndex {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleLauncher(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleLauncher(-1)
				return m, nil
			}
		}
		if m.focusIndex.Value() == FieldEditorSelectedIndex {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleEditor(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleEditor(-1)
				return m, nil
			}
		}
	}

	if msg, ok := msg.(shared.SessionCreateErrMsg); ok {
		m.errMsg = msg.Err.Error()
		return m, nil
	}

	if _, ok := msg.(shared.SessionCreatedMsg); ok {
		return m, shared.Emit(shared.NewSessionPopupCloseMsg{})
	}

	if m.focusIndex.Value() == FieldNameSelectedIndex {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *CreateFormModel) cycleProject(direction int) {
	choices := len(m.projects) + 1
	if choices <= 0 {
		return
	}
	m.projectIdx = ((m.projectIdx+direction)%choices + choices) % choices
}

func (m *CreateFormModel) cycleLauncher(direction int) {
	choices := len(m.launchers)
	if choices == 0 {
		return
	}
	m.launcherIdx = ((m.launcherIdx+direction)%choices + choices) % choices
}

func (m *CreateFormModel) cycleEditor(direction int) {
	choices := len(m.editors)
	if choices == 0 {
		return
	}
	m.editorIdx = ((m.editorIdx+direction)%choices + choices) % choices
}

func (m CreateFormModel) resolvedAgentCommand() string {
	if len(m.launchers) == 0 {
		return ""
	}
	return m.launchers[m.launcherIdx].Command
}

func (m CreateFormModel) resolvedEditorCommand() string {
	if len(m.editors) == 0 {
		return ""
	}
	return m.editors[m.editorIdx].Command
}

func (m CreateFormModel) selectedProjectID() uuid.UUID {
	if m.projectIdx == 0 {
		return uuid.Nil
	}
	return m.projects[m.projectIdx-1].ID
}

func (m CreateFormModel) selectedProjectLabel() string {
	if m.projectIdx == 0 || len(m.projects) == 0 {
		return "(none)"
	}
	return m.projects[m.projectIdx-1].Name
}

func (m CreateFormModel) submit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.errMsg = "session name is required"
		return m, nil
	}

	m.errMsg = ""
	req := service.CreateSessionRequest{
		Name:          name,
		ProjectID:     m.selectedProjectID(),
		AgentCommand:  m.resolvedAgentCommand(),
		EditorCommand: m.resolvedEditorCommand(),
	}
	return m, func() tea.Msg {
		resp, err := m.sessionsService.Create(context.Background(), req)
		if err != nil {
			return shared.SessionCreateErrMsg{Err: err}
		}
		return shared.SessionCreatedMsg{Session: resp.Session}
	}
}

func (m *CreateFormModel) updateFocusAndBlurs() {
	if m.focusIndex.Value() == FieldNameSelectedIndex {
		m.nameInput.Focus()
		return
	}
	m.nameInput.Blur()
}

func (m CreateFormModel) View() tea.View {
	s := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(m.labelStyle(FieldNameSelectedIndex).Render("Name"))
	b.WriteByte('\n')
	b.WriteString(m.nameInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(FieldProjectSelectedIndex).Render("Project"))
	b.WriteByte('\n')
	b.WriteString(m.projectSelectorView())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(FieldLauncherSelectedIndex).Render("Launcher"))
	b.WriteByte('\n')
	b.WriteString(m.launcherSelectorView())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(FieldEditorSelectedIndex).Render("Editor"))
	b.WriteByte('\n')
	b.WriteString(m.editorSelectorView())
	b.WriteByte('\n')
	b.WriteString(s.Error.Render(m.errMsg))
	b.WriteByte('\n')
	if m.errMsg != "" {
		b.WriteByte('\n')
	}
	b.WriteString(m.styles.Help.Description.Render("Tab: next field  ←/→: cycle  Enter: submit  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
}

func (m CreateFormModel) labelStyle(field int) lipgloss.Style {
	if m.focusIndex.Value() == field {
		return m.styles.Form.Field.LabelFocused
	}
	return m.styles.Form.Field.Label
}

func (m CreateFormModel) projectSelectorView() string {
	label := m.selectedProjectLabel()
	if m.focusIndex.Value() == FieldProjectSelectedIndex {
		return m.styles.ListRow.Selected.Render("< " + label + " >")
	}
	return m.styles.ListRow.Normal.Render("  " + label + "  ")
}

func (m CreateFormModel) launcherSelectorView() string {
	parts := make([]string, 0, len(m.launchers))
	for i, l := range m.launchers {
		if i == m.launcherIdx {
			parts = append(parts, m.styles.ListRow.Selected.Render("[ "+l.DisplayName+" ]"))
			continue
		}
		parts = append(parts, m.styles.ListRow.Normal.Render("  "+l.DisplayName+"  "))
	}
	return strings.Join(parts, " ")
}

func (m CreateFormModel) editorSelectorView() string {
	parts := make([]string, 0, len(m.editors))
	for i, e := range m.editors {
		if i == m.editorIdx {
			parts = append(parts, m.styles.ListRow.Selected.Render("[ "+e.DisplayName+" ]"))
			continue
		}
		parts = append(parts, m.styles.ListRow.Normal.Render("  "+e.DisplayName+"  "))
	}
	return strings.Join(parts, " ")
}

func (m CreateFormModel) KeyBindings() []key.Binding {
	return []key.Binding{popupNextFieldKeyBinding, popupPrevFieldKeyBinding, popupSelectorNextKeyBinding, popupSelectorPrevKeyBinding, popupSubmitFormKeyBinding, popupCloseKeyBinding}
}
