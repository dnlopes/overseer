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
}

func NewCheckoutBranchForm(
	s *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	projects []domain.Project,
	launchers []domain.Launcher,
	editors []domain.Editor,
) CheckoutBranchFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "(defaults to branch)"
	nameInput.CharLimit = 100
	nameInput.SetWidth(36)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	branchInput := textinput.New()
	branchInput.Placeholder = "main"
	branchInput.CharLimit = 200
	branchInput.SetWidth(36)
	branchInput.SetStyles(s.Form.Input)

	return CheckoutBranchFormModel{
		nameInput:       nameInput,
		repoPicker:      newRepoPicker(s, projects),
		branchInput:     branchInput,
		launchers:       launchers,
		launcherIdx:     0,
		editors:         editors,
		editorIdx:       0,
		focusIndex:      shared.NewCircularInt(0, totalCheckoutBranchFields-1),
		sessionsService: sessionsService,
		projectsService: projectsService,
		styles:          s,
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
		return m, m.detectDefaultBranchCmd(msg.Project.ID)

	case shared.ProjectRegisterErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case checkoutBranchDefaultDetectedMsg:
		if msg.Err == nil && strings.TrimSpace(m.branchInput.Value()) == "" {
			m.branchInput.SetValue(msg.Branch)
		}
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

type checkoutBranchDefaultDetectedMsg struct {
	Branch string
	Err    error
}

func (m CheckoutBranchFormModel) detectDefaultBranchCmd(projectID uuid.UUID) tea.Cmd {
	svc := m.projectsService
	return func() tea.Msg {
		resp, err := svc.DetectDefaultBranch(context.Background(), service.DetectDefaultBranchRequest{ProjectID: projectID})
		return checkoutBranchDefaultDetectedMsg{Branch: resp.Branch, Err: err}
	}
}

func (m CheckoutBranchFormModel) moveFocus(direction int) (tea.Model, tea.Cmd) {
	if direction > 0 {
		m.focusIndex.Increment()
	} else {
		m.focusIndex.Decrement()
	}
	m.updateFocusAndBlurs()

	if m.focusIndex.Value() == checkoutFieldBranch && strings.TrimSpace(m.branchInput.Value()) == "" {
		if proj := m.repoPicker.selectedProject(); proj != nil {
			return m, m.detectDefaultBranchCmd(proj.ID)
		}
	}
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
	if branch == "" {
		m.errMsg = "branch is required"
		return m, nil
	}

	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		name = branch
	}

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
	s := m.styles.Form.Field

	var b strings.Builder
	b.WriteString(m.styles.Form.Title.Render("Checkout Branch"))
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(checkoutFieldName).Render("Name"))
	b.WriteByte('\n')
	b.WriteString(m.nameInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(checkoutFieldRepository).Render("Repository"))
	b.WriteByte('\n')
	b.WriteString(m.repoPicker.view())
	b.WriteByte('\n')
	b.WriteString(m.styles.Help.Description.Render(m.repoPickerHint()))
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(checkoutFieldBranch).Render("Branch"))
	b.WriteByte('\n')
	b.WriteString(m.branchInput.View())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(checkoutFieldLauncher).Render("Launcher"))
	b.WriteByte('\n')
	b.WriteString(m.launcherSelectorView())
	b.WriteByte('\n')
	b.WriteString(m.labelStyle(checkoutFieldEditor).Render("Editor"))
	b.WriteByte('\n')
	b.WriteString(m.editorSelectorView())
	b.WriteByte('\n')
	b.WriteString(s.Error.Render(m.errMsg))
	if m.errMsg != "" {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(m.styles.Help.Description.Render("Tab: next field  Enter: checkout  Esc: cancel"))
	return tea.NewView(components.Modal(m.styles, b.String(), 0, 0))
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

func (m CheckoutBranchFormModel) editorSelectorView() string {
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

func (m CheckoutBranchFormModel) repoPickerHint() string {
	if m.repoPicker.isPasteMode() {
		return "Enter: confirm path  Ctrl+L: back to list"
	}
	return "↑/↓: cycle repos  Ctrl+P: paste a new path"
}
