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

type CreateFormModel struct {
	nameInput                textinput.Model
	repoPicker               repoPicker
	createWorktree           bool
	baseBranchPicker         branchPicker
	newBranchInput           textinput.Model
	branchesByProject        map[uuid.UUID][]domain.BranchInfo
	defaultBranchByProject   map[uuid.UUID]string
	launchers                []domain.Launcher
	launcherIdx              int
	editors                  []domain.Editor
	editorIdx                int
	focusOrder               []formField
	focusIdx                 int
	errMsg                   string
	sessionsService          service.SessionService
	projectsService          service.ProjectService
	styles                   *styles.Styles
	contentWidth             int
}

type formField int

const (
	fieldName formField = iota
	fieldRepository
	fieldCreateWorktreeToggle
	fieldBaseBranchPicker
	fieldNewBranch
	fieldLauncher
	fieldEditor
)

func NewCreateForm(
	s *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	projects []domain.Project,
	initialProjectID uuid.UUID,
	branchesByProject map[uuid.UUID][]domain.BranchInfo,
	defaultBranchByProject map[uuid.UUID]string,
	launchers []domain.Launcher,
	editors []domain.Editor,
	terminalWidth int,
) CreateFormModel {
	contentWidth := formContentWidth(terminalWidth)
	inputWidth := formValueColumnWidth(contentWidth)

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 100
	nameInput.SetWidth(inputWidth)
	nameInput.SetStyles(s.Form.Input)
	nameInput.Focus()

	newBranchInput := textinput.New()
	newBranchInput.Placeholder = "(auto-generated if empty)"
	newBranchInput.CharLimit = 200
	newBranchInput.SetWidth(inputWidth)
	newBranchInput.SetStyles(s.Form.Input)

	if branchesByProject == nil {
		branchesByProject = map[uuid.UUID][]domain.BranchInfo{}
	}
	if defaultBranchByProject == nil {
		defaultBranchByProject = map[uuid.UUID]string{}
	}
	repoPicker := newRepoPicker(s, projects, initialProjectID, inputWidth)
	initialBranches := branchesForSelectedProject(repoPicker, branchesByProject)
	initialDefault := defaultBranchForSelectedProject(repoPicker, defaultBranchByProject)

	m := CreateFormModel{
		nameInput:              nameInput,
		repoPicker:             repoPicker,
		createWorktree:         true,
		baseBranchPicker:       newBranchPicker(s, initialBranches, initialDefault, inputWidth),
		newBranchInput:         newBranchInput,
		branchesByProject:      branchesByProject,
		defaultBranchByProject: defaultBranchByProject,
		launchers:              launchers,
		editors:                editors,
		sessionsService:        sessionsService,
		projectsService:        projectsService,
		styles:                 s,
		contentWidth:           contentWidth,
	}
	m.rebuildFocusOrder()
	return m
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

		current := m.currentField()

		if current == fieldRepository && m.repoPicker.isPasteMode() && key.Matches(msg, popupSubmitFormKeyBinding) {
			return m.confirmPastedPath()
		}

		if key.Matches(msg, popupSubmitFormKeyBinding) && current != fieldBaseBranchPicker {
			return m.submit()
		}
		if key.Matches(msg, popupNextFieldKeyBinding) {
			return m.moveFocus(1)
		}
		if key.Matches(msg, popupPrevFieldKeyBinding) {
			return m.moveFocus(-1)
		}

		switch current {
		case fieldRepository:
			var cmd tea.Cmd
			m.repoPicker, cmd = m.repoPicker.update(msg)
			m.refreshBranchPickerFromSelection()
			return m, cmd
		case fieldCreateWorktreeToggle:
			if key.Matches(msg, popupToggleKeyBinding) ||
				key.Matches(msg, popupSelectorNextKeyBinding) ||
				key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.createWorktree = !m.createWorktree
				m.rebuildFocusOrder()
				return m, nil
			}
		case fieldBaseBranchPicker:
			if key.Matches(msg, popupSubmitFormKeyBinding) {
				m.baseBranchPicker.confirmSelection()
				return m.moveFocus(1)
			}
			var cmd tea.Cmd
			m.baseBranchPicker, cmd = m.baseBranchPicker.update(msg)
			return m, cmd
		case fieldLauncher:
			if key.Matches(msg, popupSelectorNextKeyBinding) {
				m.cycleLauncher(1)
				return m, nil
			}
			if key.Matches(msg, popupSelectorPrevKeyBinding) {
				m.cycleLauncher(-1)
				return m, nil
			}
		case fieldEditor:
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
		m.refreshBranchPickerFromSelection()
		m.errMsg = ""
		return m, nil

	case shared.ProjectRegisterErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case shared.BranchesLoadedMsg:
		if msg.Err == nil {
			m.branchesByProject[msg.ProjectID] = msg.Branches
			m.defaultBranchByProject[msg.ProjectID] = msg.DefaultBranch
			if sel := m.repoPicker.selectedProject(); sel != nil && sel.ID == msg.ProjectID {
				m.baseBranchPicker.setBranches(msg.Branches, msg.DefaultBranch)
			}
		}
		return m, nil
	}

	switch m.currentField() {
	case fieldName:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	case fieldNewBranch:
		var cmd tea.Cmd
		m.newBranchInput, cmd = m.newBranchInput.Update(msg)
		return m, cmd
	case fieldRepository:
		var cmd tea.Cmd
		m.repoPicker, cmd = m.repoPicker.update(msg)
		return m, cmd
	case fieldBaseBranchPicker:
		var cmd tea.Cmd
		m.baseBranchPicker, cmd = m.baseBranchPicker.update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *CreateFormModel) rebuildFocusOrder() {
	order := []formField{fieldName, fieldRepository, fieldCreateWorktreeToggle}
	if m.createWorktree {
		order = append(order, fieldBaseBranchPicker, fieldNewBranch)
	}
	order = append(order, fieldLauncher, fieldEditor)
	m.focusOrder = order
	if m.focusIdx >= len(order) {
		m.focusIdx = 0
	}
	m.updateFocusAndBlurs()
}

func (m CreateFormModel) currentField() formField {
	if m.focusIdx < 0 || m.focusIdx >= len(m.focusOrder) {
		return fieldName
	}
	return m.focusOrder[m.focusIdx]
}

func (m CreateFormModel) moveFocus(direction int) (tea.Model, tea.Cmd) {
	n := len(m.focusOrder)
	if n == 0 {
		return m, nil
	}
	m.focusIdx = ((m.focusIdx+direction)%n + n) % n
	m.updateFocusAndBlurs()
	return m, nil
}

func (m *CreateFormModel) updateFocusAndBlurs() {
	m.nameInput.Blur()
	m.newBranchInput.Blur()
	m.repoPicker.blur()
	m.baseBranchPicker.blur()

	switch m.currentField() {
	case fieldName:
		m.nameInput.Focus()
	case fieldNewBranch:
		m.newBranchInput.Focus()
	case fieldRepository:
		m.repoPicker.focus()
	case fieldBaseBranchPicker:
		m.baseBranchPicker.focus()
	}
}

func (m *CreateFormModel) refreshBranchPickerFromSelection() {
	if sel := m.repoPicker.selectedProject(); sel != nil {
		if branches, ok := m.branchesByProject[sel.ID]; ok {
			m.baseBranchPicker.setBranches(branches, m.defaultBranchByProject[sel.ID])
		}
	}
}

func branchesForSelectedProject(p repoPicker, m map[uuid.UUID][]domain.BranchInfo) []domain.BranchInfo {
	sel := p.selectedProject()
	if sel == nil {
		return nil
	}
	return m[sel.ID]
}

func defaultBranchForSelectedProject(p repoPicker, m map[uuid.UUID]string) string {
	sel := p.selectedProject()
	if sel == nil {
		return ""
	}
	return m[sel.ID]
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

func (m CreateFormModel) resolvedAgentType() domain.AgentType {
	if len(m.launchers) == 0 {
		return ""
	}
	return m.launchers[m.launcherIdx].AgentType
}

func (m CreateFormModel) resolvedEditorCommand() string {
	if len(m.editors) == 0 {
		return ""
	}
	return m.editors[m.editorIdx].Command
}

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
		m.errMsg = "press enter to confirm the pasted path"
		return m, nil
	}

	req := service.CreateSessionRequest{
		Name:           name,
		ProjectID:      selection.Project.ID,
		CreateWorktree: m.createWorktree,
		AgentCommand:   m.resolvedAgentCommand(),
		EditorCommand:  m.resolvedEditorCommand(),
		AgentType:      m.resolvedAgentType(),
	}
	if m.createWorktree {
		if sel, ok := m.baseBranchPicker.selected(); ok {
			req.BaseBranch = sel.Name
		}
		req.Branch = strings.TrimSpace(m.newBranchInput.Value())
	}

	m.errMsg = ""
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
		renderField(m.styles, m.labelStyle(fieldName), "Name", m.nameInput.View()),
		"",
		renderField(m.styles, m.labelStyle(fieldRepository), "Repository", m.repoPicker.view()),
		renderFieldHint(m.styles, m.repoPickerHint()),
		"",
		renderField(m.styles, m.labelStyle(fieldCreateWorktreeToggle), "Create worktree?", m.toggleView()),
		renderFieldHint(m.styles, "space / ← → toggle"),
	}

	if m.createWorktree {
		parts = append(parts,
			"",
			renderField(m.styles, m.labelStyle(fieldBaseBranchPicker), "Base branch", m.baseBranchPicker.view()),
			"",
			renderField(m.styles, m.labelStyle(fieldNewBranch), "New branch", m.newBranchInput.View()),
		)
	}

	parts = append(parts,
		"",
		renderField(m.styles, m.labelStyle(fieldLauncher), "Launcher", m.launcherSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle launchers"),
		"",
		renderField(m.styles, m.labelStyle(fieldEditor), "Editor", m.editorSelectorView()),
		renderFieldHint(m.styles, "←/→ cycle editors"),
	)

	if m.errMsg != "" {
		parts = append(parts, "", m.styles.Form.Field.Error.Render(m.errMsg))
	}
	parts = append(parts, "", m.styles.Help.Description.Render("Tab: next field  Enter: submit  Esc: cancel"))

	body := padBodyLines(m.styles, strings.Join(parts, "\n"), m.contentWidth)
	return tea.NewView(components.Modal(m.styles, body, m.contentWidth, 0))
}

func (m CreateFormModel) labelStyle(field formField) lipgloss.Style {
	if m.currentField() == field {
		return m.styles.Form.Field.LabelFocused
	}
	return m.styles.Form.Field.Label
}

func (m CreateFormModel) toggleView() string {
	value := "Off"
	if m.createWorktree {
		value = "On"
	}
	focused := m.currentField() == fieldCreateWorktreeToggle
	if focused {
		return modalListRow(m.styles, true).Render("< " + value + " >")
	}
	return modalListRow(m.styles, false).Render("  " + value + "  ")
}

func (m CreateFormModel) launcherSelectorView() string {
	if len(m.launchers) == 0 {
		return modalListRow(m.styles, false).Render("  (no launchers configured)  ")
	}
	name := m.launchers[m.launcherIdx].DisplayName
	if m.currentField() == fieldLauncher {
		return modalListRow(m.styles, true).Render("< " + name + " >")
	}
	return modalListRow(m.styles, false).Render("  " + name + "  ")
}

func (m CreateFormModel) editorSelectorView() string {
	if len(m.editors) == 0 {
		return modalListRow(m.styles, false).Render("  (no editors configured)  ")
	}
	name := m.editors[m.editorIdx].DisplayName
	if m.currentField() == fieldEditor {
		return modalListRow(m.styles, true).Render("< " + name + " >")
	}
	return modalListRow(m.styles, false).Render("  " + name + "  ")
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
		popupToggleKeyBinding,
		popupSelectorNextKeyBinding,
		popupSelectorPrevKeyBinding,
		repoPickerEnterPasteKeyBinding,
		repoPickerExitPasteKeyBinding,
		branchPickerUpKeyBinding,
		branchPickerDownKeyBinding,
		popupSubmitFormKeyBinding,
		popupCloseKeyBinding,
	}
}
