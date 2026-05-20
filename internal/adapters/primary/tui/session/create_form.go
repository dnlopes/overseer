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
	fieldName int = iota
	fieldRepository
	fieldBaseBranch
	fieldLauncher
	fieldEditor
)

const totalCreateFields = 5

type CreateFormModel struct {
	nameInput       textinput.Model
	repoPicker      repoPicker
	baseBranchInput textinput.Model
	launchers       []domain.Launcher
	launcherIdx     int
	editors         []domain.Editor
	editorIdx       int
	focusIndex      shared.CircularInt
	errMsg          string
	sessionsService service.SessionService
	projectsService service.ProjectService
	styles          *styles.Styles
}

// NewCreateForm builds the session-create form. The supplied projects seed
// the repo picker's "recent" list (ordered by UpdatedAt server-side).
func NewCreateForm(
	s *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	projects []domain.Project,
	launchers []domain.Launcher,
	editors []domain.Editor,
) CreateFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(36)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	baseBranchInput := textinput.New()
	baseBranchInput.Placeholder = "main"
	baseBranchInput.CharLimit = 200
	baseBranchInput.SetWidth(36)
	baseBranchInput.SetStyles(s.Form.Input)

	return CreateFormModel{
		nameInput:       nameInput,
		repoPicker:      newRepoPicker(s, projects),
		baseBranchInput: baseBranchInput,
		launchers:       launchers,
		launcherIdx:     0,
		editors:         editors,
		editorIdx:       0,
		focusIndex:      shared.NewCircularInt(0, totalCreateFields-1),
		sessionsService: sessionsService,
		projectsService: projectsService,
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

		// Enter in paste-mode confirms the pasted path (registers the project)
		// rather than submitting the whole form.
		if m.focusIndex.Value() == fieldRepository && m.repoPicker.isPasteMode() && key.Matches(msg, popupSubmitFormKeyBinding) {
			return m.confirmPastedPath()
		}

		if key.Matches(msg, popupSubmitFormKeyBinding) {
			return m.submit()
		}
		if key.Matches(msg, popupNextFieldKeyBinding) {
			return m.moveFocus(1)
		}
		if key.Matches(msg, popupPrevFieldKeyBinding) {
			return m.moveFocus(-1)
		}

		if m.focusIndex.Value() == fieldRepository {
			var cmd tea.Cmd
			m.repoPicker, cmd = m.repoPicker.update(msg)
			return m, cmd
		}

		if m.focusIndex.Value() == fieldLauncher {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleLauncher(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleLauncher(-1)
				return m, nil
			}
		}

		if m.focusIndex.Value() == fieldEditor {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleEditor(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleEditor(-1)
				return m, nil
			}
		}

	case shared.SessionCreateErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case shared.SessionCreatedMsg:
		return m, shared.Emit(shared.NewSessionPopupCloseMsg{})

	case shared.ProjectRegisteredMsg:
		m.repoPicker.adoptRegisteredProject(msg.Project)
		m.errMsg = ""
		return m, m.detectDefaultBranchCmd(msg.Project.ID)

	case shared.ProjectRegisterErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case defaultBranchDetectedMsg:
		if msg.Err == nil && strings.TrimSpace(m.baseBranchInput.Value()) == "" {
			m.baseBranchInput.SetValue(msg.Branch)
		}
		return m, nil
	}

	switch m.focusIndex.Value() {
	case fieldName:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	case fieldBaseBranch:
		var cmd tea.Cmd
		m.baseBranchInput, cmd = m.baseBranchInput.Update(msg)
		return m, cmd
	case fieldRepository:
		var cmd tea.Cmd
		m.repoPicker, cmd = m.repoPicker.update(msg)
		return m, cmd
	}

	return m, nil
}

// defaultBranchDetectedMsg carries the result of an async
// ProjectService.DetectDefaultBranch call so the form can populate (or leave
// empty) the base-branch field.
type defaultBranchDetectedMsg struct {
	Branch string
	Err    error
}

func (m CreateFormModel) detectDefaultBranchCmd(projectID uuid.UUID) tea.Cmd {
	svc := m.projectsService
	return func() tea.Msg {
		resp, err := svc.DetectDefaultBranch(context.Background(), service.DetectDefaultBranchRequest{ProjectID: projectID})
		return defaultBranchDetectedMsg{Branch: resp.Branch, Err: err}
	}
}

// moveFocus shifts focus by direction (+1 next, -1 previous) and triggers a
// default-branch auto-detect when the user enters an empty BaseBranch field
// with a known project selected.
func (m CreateFormModel) moveFocus(direction int) (tea.Model, tea.Cmd) {
	if direction > 0 {
		m.focusIndex.Increment()
	} else {
		m.focusIndex.Decrement()
	}
	m.updateFocusAndBlurs()

	if m.focusIndex.Value() == fieldBaseBranch && strings.TrimSpace(m.baseBranchInput.Value()) == "" {
		if proj := m.repoPicker.selectedProject(); proj != nil {
			return m, m.detectDefaultBranchCmd(proj.ID)
		}
	}
	return m, nil
}

func (m *CreateFormModel) updateFocusAndBlurs() {
	m.nameInput.Blur()
	m.baseBranchInput.Blur()
	m.repoPicker.blur()

	switch m.focusIndex.Value() {
	case fieldName:
		m.nameInput.Focus()
	case fieldBaseBranch:
		m.baseBranchInput.Focus()
	case fieldRepository:
		m.repoPicker.focus()
	}
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

// confirmPastedPath fires a Register cmd for the path the user just typed in
// paste mode. The picker stays in paste mode until ProjectRegisteredMsg
// arrives; on error the form surfaces the message and the user can edit.
func (m CreateFormModel) confirmPastedPath() (tea.Model, tea.Cmd) {
	path := m.repoPicker.pastedPath()
	if path == "" {
		m.errMsg = "repository path is required"
		return m, nil
	}
	m.errMsg = ""
	svc := m.projectsService
	return m, func() tea.Msg {
		resp, err := svc.Register(context.Background(), service.RegisterProjectRequest{Path: path})
		if err != nil {
			return shared.ProjectRegisterErrMsg{Err: err}
		}
		return shared.ProjectRegisteredMsg{Project: resp.Project}
	}
}

func (m CreateFormModel) submit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.errMsg = "session name is required"
		return m, nil
	}

	selection := m.repoPicker.resolve()
	if selection.IsZero() {
		m.errMsg = "select a repository"
		return m, nil
	}
	if selection.Project == nil {
		// User pasted a path but didn't confirm with Enter — refuse to silently
		// register on submit; force an explicit confirm so the user sees errors
		// inline before the whole submission attempt.
		m.errMsg = "press enter to confirm the pasted path"
		return m, nil
	}

	baseBranch := strings.TrimSpace(m.baseBranchInput.Value())
	if baseBranch == "" {
		m.errMsg = "base branch is required"
		return m, nil
	}

	m.errMsg = ""
	req := service.CreateSessionRequest{
		Name:          name,
		ProjectID:     selection.Project.ID,
		BaseBranch:    baseBranch,
		AgentCommand:  m.resolvedAgentCommand(),
		EditorCommand: m.resolvedEditorCommand(),
	}
	svc := m.sessionsService
	return m, func() tea.Msg {
		resp, err := svc.Create(context.Background(), req)
		if err != nil {
			return shared.SessionCreateErrMsg{Err: err}
		}
		return shared.SessionCreatedMsg{Session: resp.Session}
	}
}

func (m CreateFormModel) View() tea.View {
	s := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(m.labelStyle(fieldName).Render("Name"))
	b.WriteByte('\n')
	b.WriteString(m.nameInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(fieldRepository).Render("Repository"))
	b.WriteByte('\n')
	b.WriteString(m.repoPicker.view())
	b.WriteByte('\n')
	b.WriteString(m.styles.Help.Description.Render(m.repoPickerHint()))
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(fieldBaseBranch).Render("Base branch"))
	b.WriteByte('\n')
	b.WriteString(m.baseBranchInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(fieldLauncher).Render("Launcher"))
	b.WriteByte('\n')
	b.WriteString(m.launcherSelectorView())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(fieldEditor).Render("Editor"))
	b.WriteByte('\n')
	b.WriteString(m.editorSelectorView())
	b.WriteByte('\n')
	b.WriteString(s.Error.Render(m.errMsg))
	if m.errMsg != "" {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(m.styles.Help.Description.Render("Tab: next field  Enter: submit  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
}

func (m CreateFormModel) labelStyle(field int) lipgloss.Style {
	if m.focusIndex.Value() == field {
		return m.styles.Form.Field.LabelFocused
	}
	return m.styles.Form.Field.Label
}

func (m CreateFormModel) launcherSelectorView() string {
	if len(m.launchers) == 0 {
		return m.styles.ListRow.Normal.Render("  (no launchers configured)  ")
	}
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
	if len(m.editors) == 0 {
		return m.styles.ListRow.Normal.Render("  (no editors configured)  ")
	}
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

func (m CreateFormModel) repoPickerHint() string {
	if m.repoPicker.isPasteMode() {
		return "Enter: confirm path  Ctrl+L: back to list"
	}
	return "←/→: cycle repos  Ctrl+P: paste new path"
}

func (m CreateFormModel) KeyBindings() []key.Binding {
	return []key.Binding{
		popupNextFieldKeyBinding,
		popupPrevFieldKeyBinding,
		popupSelectorNextKeyBinding,
		popupSelectorPrevKeyBinding,
		repoPickerEnterPasteKeyBinding,
		repoPickerExitPasteKeyBinding,
		popupSubmitFormKeyBinding,
		popupCloseKeyBinding,
	}
}
