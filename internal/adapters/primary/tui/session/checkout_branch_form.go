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
	checkoutFieldName int = iota
	checkoutFieldRepository
	checkoutFieldBranch
	checkoutFieldLauncher
	checkoutFieldEditor
)

const totalCheckoutBranchFields = 5

type CheckoutBranchFormModel struct {
	nameInput       textinput.Model
	repoPicker      repoPicker
	branchInput     textinput.Model
	launchers       []domain.Launcher
	launcherIdx     int
	editors         []domain.Editor
	editorIdx       int
	focusIndex      shared.CircularInt
	errMsg          string
	sessionsService service.SessionService
	projectsService service.ProjectService
	styles          *styles.Styles
	contentWidth    int
}

func NewCheckoutBranchForm(
	s *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	projects []domain.Project,
	initialProjectID uuid.UUID,
	launchers []domain.Launcher,
	editors []domain.Editor,
	terminalWidth int,
) CheckoutBranchFormModel {
	contentWidth := formContentWidth(terminalWidth)
	inputWidth := formInputWidth(contentWidth)

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(inputWidth)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	branchInput := textinput.New()
	branchInput.Placeholder = "(repo default)"
	branchInput.CharLimit = 200
	branchInput.SetWidth(inputWidth)
	branchInput.SetStyles(s.Form.Input)

	return CheckoutBranchFormModel{
		nameInput:       nameInput,
		repoPicker:      newRepoPicker(s, projects, initialProjectID, inputWidth),
		branchInput:     branchInput,
		launchers:       launchers,
		launcherIdx:     0,
		editors:         editors,
		editorIdx:       0,
		focusIndex:      shared.NewCircularInt(0, totalCheckoutBranchFields-1),
		sessionsService: sessionsService,
		projectsService: projectsService,
		styles:          s,
		contentWidth:    contentWidth,
	}
}

func (m CheckoutBranchFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CheckoutBranchFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, popupCloseKeyBinding) {
			return m, shared.Emit(shared.CheckoutBranchPopupCloseMsg{})
		}

		if m.focusIndex.Value() == checkoutFieldRepository && m.repoPicker.isPasteMode() && key.Matches(msg, popupSubmitFormKeyBinding) {
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

		if m.focusIndex.Value() == checkoutFieldRepository {
			var cmd tea.Cmd
			m.repoPicker, cmd = m.repoPicker.update(msg)
			return m, cmd
		}

		if m.focusIndex.Value() == checkoutFieldLauncher {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleLauncher(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleLauncher(-1)
				return m, nil
			}
		}

		if m.focusIndex.Value() == checkoutFieldEditor {
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleEditor(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleEditor(-1)
				return m, nil
			}
		}

	case shared.SessionCheckoutErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case shared.SessionCheckedOutMsg:
		return m, shared.Emit(shared.CheckoutBranchPopupCloseMsg{})

	case shared.ProjectRegisteredMsg:
		m.repoPicker.adoptRegisteredProject(msg.Project)
		m.errMsg = ""
		return m, nil

	case shared.ProjectRegisterErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil
	}

	switch m.focusIndex.Value() {
	case checkoutFieldName:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	case checkoutFieldBranch:
		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		return m, cmd
	case checkoutFieldRepository:
		var cmd tea.Cmd
		m.repoPicker, cmd = m.repoPicker.update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CheckoutBranchFormModel) moveFocus(direction int) (tea.Model, tea.Cmd) {
	if direction > 0 {
		m.focusIndex.Increment()
	} else {
		m.focusIndex.Decrement()
	}
	m.updateFocusAndBlurs()
	return m, nil
}

func (m *CheckoutBranchFormModel) updateFocusAndBlurs() {
	m.nameInput.Blur()
	m.branchInput.Blur()
	m.repoPicker.blur()

	switch m.focusIndex.Value() {
	case checkoutFieldName:
		m.nameInput.Focus()
	case checkoutFieldBranch:
		m.branchInput.Focus()
	case checkoutFieldRepository:
		m.repoPicker.focus()
	}
}

func (m *CheckoutBranchFormModel) cycleLauncher(direction int) {
	choices := len(m.launchers)
	if choices == 0 {
		return
	}
	m.launcherIdx = ((m.launcherIdx+direction)%choices + choices) % choices
}

func (m *CheckoutBranchFormModel) cycleEditor(direction int) {
	choices := len(m.editors)
	if choices == 0 {
		return
	}
	m.editorIdx = ((m.editorIdx+direction)%choices + choices) % choices
}

func (m CheckoutBranchFormModel) resolvedAgentCommand() string {
	if len(m.launchers) == 0 {
		return ""
	}
	return m.launchers[m.launcherIdx].Command
}

func (m CheckoutBranchFormModel) resolvedEditorCommand() string {
	if len(m.editors) == 0 {
		return ""
	}
	return m.editors[m.editorIdx].Command
}

func (m CheckoutBranchFormModel) confirmPastedPath() (tea.Model, tea.Cmd) {
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

func (m CheckoutBranchFormModel) submit() (tea.Model, tea.Cmd) {
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
		m.errMsg = "press enter to confirm the pasted path"
		return m, nil
	}

	branch := strings.TrimSpace(m.branchInput.Value())

	m.errMsg = ""
	req := service.CheckoutBranchRequest{
		Name:          name,
		ProjectID:     selection.Project.ID,
		Branch:        branch,
		AgentCommand:  m.resolvedAgentCommand(),
		EditorCommand: m.resolvedEditorCommand(),
	}
	svc := m.sessionsService
	return m, func() tea.Msg {
		resp, err := svc.CheckoutBranch(context.Background(), req)
		if err != nil {
			return shared.SessionCheckoutErrMsg{Err: err}
		}
		return shared.SessionCheckedOutMsg{Session: resp.Session}
	}
}

func (m CheckoutBranchFormModel) View() tea.View {
	parts := []string{
		m.styles.Form.Title.Render("Checkout Branch"),

		renderField(m.labelStyle(checkoutFieldName), "Name", m.nameInput.View()),
		"",
		renderField(m.labelStyle(checkoutFieldRepository), "Repository", m.repoPicker.view()),
		renderFieldHint(m.styles, m.repoPickerHint()),
		"",
		renderField(m.labelStyle(checkoutFieldBranch), "Branch", m.branchInput.View()),
		"",
		renderField(m.labelStyle(checkoutFieldLauncher), "Launcher", m.launcherSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle launchers"),
		"",
		renderField(m.labelStyle(checkoutFieldEditor), "Editor", m.editorSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle editors"),
	}

	if m.errMsg != "" {
		parts = append(parts, "", m.styles.Form.Field.Error.Render(m.errMsg))
	}
	parts = append(parts, "", m.styles.Form.Hint.Render("Tab: next field  Enter: checkout  Esc: cancel"))

	body := padBodyLines(m.styles, strings.Join(parts, "\n"), m.contentWidth)
	return tea.NewView(components.Modal(m.styles, body, m.contentWidth, 0))
}

func (m CheckoutBranchFormModel) labelStyle(field int) lipgloss.Style {
	if m.focusIndex.Value() == field {
		return m.styles.Form.Field.LabelFocused
	}
	return m.styles.Form.Field.Label
}

func (m CheckoutBranchFormModel) launcherSelectorView() string {
	if len(m.launchers) == 0 {
		return m.styles.ListRow.Normal.Render("  (no launchers configured)  ")
	}
	name := m.launchers[m.launcherIdx].DisplayName
	if m.focusIndex.Value() == checkoutFieldLauncher {
		return m.styles.ListRow.Selected.Render("< " + name + " >")
	}
	return m.styles.ListRow.Normal.Render("  " + name + "  ")
}

func (m CheckoutBranchFormModel) editorSelectorView() string {
	if len(m.editors) == 0 {
		return m.styles.ListRow.Normal.Render("  (no editors configured)  ")
	}
	name := m.editors[m.editorIdx].DisplayName
	if m.focusIndex.Value() == checkoutFieldEditor {
		return m.styles.ListRow.Selected.Render("< " + name + " >")
	}
	return m.styles.ListRow.Normal.Render("  " + name + "  ")
}

func (m CheckoutBranchFormModel) repoPickerHint() string {
	if m.repoPicker.isPasteMode() {
		return "Enter confirm · Ctrl+L back"
	}
	return "←/→ cycle · Ctrl+P paste path"
}
