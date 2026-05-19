package project

import (
	"context"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/components"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

const (
	FieldPathSelectedIndex int = iota
	FieldNameSelectedIndex
)

type RegisterFormModel struct {
	pathInput      textinput.Model
	nameInput      textinput.Model
	focusIndex     shared.CircularInt
	errMsg         string
	projectService service.ProjectService
	styles         *styles.Styles
	nameTouched    bool
}

func NewRegisterForm(s *styles.Styles, projectService service.ProjectService) RegisterFormModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "/absolute/path/to/repo"
	pathInput.CharLimit = 4096
	pathInput.SetWidth(48)
	pathInput.SetStyles(s.Form.Input)
	pathInput.Focus()

	nameInput := textinput.New()
	nameInput.Placeholder = "Project name (auto-filled)"
	nameInput.CharLimit = 100
	nameInput.SetWidth(48)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Blur()

	return RegisterFormModel{
		pathInput:      pathInput,
		nameInput:      nameInput,
		focusIndex:     shared.NewCircularInt(0, 1),
		projectService: projectService,
		styles:         s,
	}
}

func (m RegisterFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RegisterFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, popupCloseKeyBinding) {
			return m, shared.Emit(shared.NewProjectPopupCloseMsg{})
		}
		if key.Matches(msg, popupSubmitFormKeyBinding) {
			return m.submit()
		}
		if key.Matches(msg, popupNextFieldKeyBinding) {
			m.autofillNameFromPath()
			m.focusIndex.Increment()
			m.updateFocusAndBlurs()
			return m, nil
		}
		if key.Matches(msg, popupPrevFieldKeyBinding) {
			m.focusIndex.Decrement()
			m.updateFocusAndBlurs()
			return m, nil
		}
	}

	if msg, ok := msg.(shared.ProjectRegisterErrMsg); ok {
		m.errMsg = msg.Err.Error()
		return m, nil
	}

	if _, ok := msg.(shared.ProjectRegisteredMsg); ok {
		return m, shared.Emit(shared.NewProjectPopupCloseMsg{})
	}

	switch m.focusIndex.Value() {
	case FieldPathSelectedIndex:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	case FieldNameSelectedIndex:
		previousValue := m.nameInput.Value()
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		if m.nameInput.Value() != previousValue {
			m.nameTouched = true
		}
		return m, cmd
	}

	return m, nil
}

func (m *RegisterFormModel) autofillNameFromPath() {
	if m.focusIndex.Value() != FieldPathSelectedIndex {
		return
	}
	if m.nameTouched {
		return
	}
	path := strings.TrimSpace(m.pathInput.Value())
	if path == "" {
		return
	}
	m.nameInput.SetValue(filepath.Base(path))
}

func (m RegisterFormModel) submit() (tea.Model, tea.Cmd) {
	path := strings.TrimSpace(m.pathInput.Value())
	name := strings.TrimSpace(m.nameInput.Value())
	if path == "" {
		m.errMsg = "project path is required"
		return m, nil
	}

	m.errMsg = ""
	req := service.RegisterProjectRequest{Path: path, Name: name}
	return m, func() tea.Msg {
		resp, err := m.projectService.Register(context.Background(), req)
		if err != nil {
			return shared.ProjectRegisterErrMsg{Err: err}
		}
		return shared.ProjectRegisteredMsg{Project: resp.Project}
	}
}

func (m *RegisterFormModel) updateFocusAndBlurs() {
	switch m.focusIndex.Value() {
	case FieldPathSelectedIndex:
		m.pathInput.Focus()
		m.nameInput.Blur()
	case FieldNameSelectedIndex:
		m.nameInput.Focus()
		m.pathInput.Blur()
	}
}

func (m RegisterFormModel) View() tea.View {
	s := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(m.labelStyle(FieldPathSelectedIndex).Render("Path"))
	b.WriteByte('\n')
	b.WriteString(m.pathInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(FieldNameSelectedIndex).Render("Name"))
	b.WriteByte('\n')
	b.WriteString(m.nameInput.View())
	b.WriteByte('\n')
	b.WriteString(s.Error.Render(m.errMsg))
	b.WriteByte('\n')
	if m.errMsg != "" {
		b.WriteByte('\n')
	}
	b.WriteString(m.styles.Help.Description.Render("Tab: next field  Enter: submit  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
}

func (m RegisterFormModel) labelStyle(field int) lipgloss.Style {
	if m.focusIndex.Value() == field {
		return m.styles.Form.Field.LabelFocused
	}
	return m.styles.Form.Field.Label
}

func (m RegisterFormModel) KeyBindings() []key.Binding {
	return []key.Binding{popupNextFieldKeyBinding, popupPrevFieldKeyBinding, popupSubmitFormKeyBinding, popupCloseKeyBinding}
}
