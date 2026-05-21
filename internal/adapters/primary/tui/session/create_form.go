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
	fieldFeatureBranch
	fieldLauncher
	fieldEditor
)

const totalCreateFields = 6

type CreateFormModel struct {
	nameInput          textinput.Model
	repoPicker         repoPicker
	baseBranchInput    textinput.Model
	featureBranchInput textinput.Model
	launchers          []domain.Launcher
	launcherIdx        int
	editors            []domain.Editor
	editorIdx          int
	focusIndex         shared.CircularInt
	errMsg             string
	sessionsService    service.SessionService
	projectsService    service.ProjectService
	styles             *styles.Styles
	contentWidth       int
}

// NewCreateForm builds the session-create form. The supplied projects seed
// the repo picker's "recent" list (ordered by UpdatedAt server-side). When
// initialProjectID is non-Nil and present in projects, the repo picker
// starts positioned on that project; otherwise it starts on the most-recent.
// terminalWidth is the current terminal column count; the form clamps its
// modal box to [formMinBoxWidth, formMaxBoxWidth] and sizes inputs to fit.
func NewCreateForm(
	s *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	projects []domain.Project,
	initialProjectID uuid.UUID,
	launchers []domain.Launcher,
	editors []domain.Editor,
	terminalWidth int,
) CreateFormModel {
	contentWidth := formContentWidth(terminalWidth)
	inputWidth := formInputWidth(contentWidth)

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(inputWidth)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	baseBranchInput := textinput.New()
	baseBranchInput.Placeholder = "(repo default)"
	baseBranchInput.CharLimit = 200
	baseBranchInput.SetWidth(inputWidth)
	baseBranchInput.SetStyles(s.Form.Input)

	featureBranchInput := textinput.New()
	featureBranchInput.Placeholder = "(auto-generated if empty)"
	featureBranchInput.CharLimit = 200
	featureBranchInput.SetWidth(inputWidth)
	featureBranchInput.SetStyles(s.Form.Input)

	return CreateFormModel{
		nameInput:          nameInput,
		repoPicker:         newRepoPicker(s, projects, initialProjectID, inputWidth),
		baseBranchInput:    baseBranchInput,
		featureBranchInput: featureBranchInput,
		launchers:          launchers,
		launcherIdx:        0,
		editors:            editors,
		editorIdx:          0,
		focusIndex:         shared.NewCircularInt(0, totalCreateFields-1),
		sessionsService:    sessionsService,
		projectsService:    projectsService,
		styles:             s,
		contentWidth:       contentWidth,
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
		return m, nil

	case shared.ProjectRegisterErrMsg:
		m.errMsg = msg.Err.Error()
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
	case fieldFeatureBranch:
		var cmd tea.Cmd
		m.featureBranchInput, cmd = m.featureBranchInput.Update(msg)
		return m, cmd
	case fieldRepository:
		var cmd tea.Cmd
		m.repoPicker, cmd = m.repoPicker.update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CreateFormModel) moveFocus(direction int) (tea.Model, tea.Cmd) {
	if direction > 0 {
		m.focusIndex.Increment()
	} else {
		m.focusIndex.Decrement()
	}
	m.updateFocusAndBlurs()
	return m, nil
}

func (m *CreateFormModel) updateFocusAndBlurs() {
	m.nameInput.Blur()
	m.baseBranchInput.Blur()
	m.featureBranchInput.Blur()
	m.repoPicker.blur()

	switch m.focusIndex.Value() {
	case fieldName:
		m.nameInput.Focus()
	case fieldBaseBranch:
		m.baseBranchInput.Focus()
	case fieldFeatureBranch:
		m.featureBranchInput.Focus()
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

	m.errMsg = ""
	req := service.CreateSessionRequest{
		Name:          name,
		ProjectID:     selection.Project.ID,
		BaseBranch:    strings.TrimSpace(m.baseBranchInput.Value()),
		FeatureBranch: strings.TrimSpace(m.featureBranchInput.Value()),
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
	parts := []string{
		m.styles.Form.Title.Render("New Session"),

		renderField(m.labelStyle(fieldName), "Name", m.nameInput.View()),
		"",
		renderField(m.labelStyle(fieldRepository), "Repository", m.repoPicker.view()),
		renderFieldHint(m.styles, m.repoPickerHint()),
		"",
		renderField(m.labelStyle(fieldBaseBranch), "Base branch", m.baseBranchInput.View()),
		"",
		renderField(m.labelStyle(fieldFeatureBranch), "Feature branch", m.featureBranchInput.View()),
		"",
		renderField(m.labelStyle(fieldLauncher), "Launcher", m.launcherSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle launchers"),
		"",
		renderField(m.labelStyle(fieldEditor), "Editor", m.editorSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle editors"),
	}

	if m.errMsg != "" {
		parts = append(parts, "", m.styles.Form.Field.Error.Render(m.errMsg))
	}
	parts = append(parts, "", m.styles.Form.Hint.Render("Tab: next field  Enter: submit  Esc: cancel"))

	body := padBodyLines(m.styles, strings.Join(parts, "\n"), m.contentWidth)
	return tea.NewView(components.Modal(m.styles, body, m.contentWidth, 0))
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
	name := m.launchers[m.launcherIdx].DisplayName
	if m.focusIndex.Value() == fieldLauncher {
		return m.styles.ListRow.Selected.Render("< " + name + " >")
	}
	return m.styles.ListRow.Normal.Render("  " + name + "  ")
}

func (m CreateFormModel) editorSelectorView() string {
	if len(m.editors) == 0 {
		return m.styles.ListRow.Normal.Render("  (no editors configured)  ")
	}
	name := m.editors[m.editorIdx].DisplayName
	if m.focusIndex.Value() == fieldEditor {
		return m.styles.ListRow.Selected.Render("< " + name + " >")
	}
	return m.styles.ListRow.Normal.Render("  " + name + "  ")
}

func (m CreateFormModel) repoPickerHint() string {
	if m.repoPicker.isPasteMode() {
		return "Enter confirm · Ctrl+L back"
	}
	return "←/→ cycle · Ctrl+P paste path"
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
