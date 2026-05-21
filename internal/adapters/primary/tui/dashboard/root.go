package dashboard

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/inspector"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/jobs"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/leftpane"
	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/sessiondetails"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/domain"
	"github.com/dnlopes/overseer/internal/core/service"
)

const (
	SessionsListWidthPercent = 30
	TitleBarHeight           = 1
	HelpBarHeight            = 1
)

type popupKind int

const (
	popupNone popupKind = iota
	popupNewSession
	popupCheckoutBranch
	popupDeleteSession
)

type Model struct {
	titlebar       TitleBarModel
	leftPane       leftpane.Model
	inspector      inspector.Model
	helpBar        shared.HelpBarModel
	createForm     sessionui.CreateFormModel
	checkoutBranchForm sessionui.CheckoutBranchFormModel
	deleteForm     sessionui.DeleteFormModel
	scheduler      jobs.Model
	activePopup    popupKind
	cachedProjects []domain.Project
	prStatuses     map[uuid.UUID]shared.PRStatusUpdatedMsg

	width           int
	height          int
	minWidth        int
	minHeight       int
	tooSmall        bool
	leftPaneFocused bool
	styles          *styles.Styles
	sessionsService service.SessionService
	projectsService service.ProjectService
	launchers       []domain.Launcher
	editors         []domain.Editor
}

func New(
	styles *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	scheduler jobs.Model,
	launchers []domain.Launcher,
	editors []domain.Editor,
	minWidth, minHeight int,
	previewRefreshInterval time.Duration,
) Model {
	sessionsModel := sessionui.New(styles, sessionsService)
	detailsModel := sessiondetails.New(styles)
	left := leftpane.New(styles, sessionsModel, detailsModel)
	left.SetFocus(true)
	m := Model{
		styles:          styles,
		titlebar:        newTitlebar(styles, "Overseer"),
		leftPane:        left,
		inspector:       inspector.New(styles, sessionsService, previewRefreshInterval),
		helpBar:         shared.NewHelpBarModel(styles, sessionsKeyBindings),
		scheduler:       scheduler,
		sessionsService: sessionsService,
		projectsService: projectsService,
		launchers:       launchers,
		editors:         editors,
		minWidth:        minWidth,
		minHeight:       minHeight,
		leftPaneFocused: true,
		prStatuses:      make(map[uuid.UUID]shared.PRStatusUpdatedMsg),
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.titlebar.Init(),
		m.leftPane.Init(),
		m.inspector.Init(),
		m.helpBar.Init(),
		m.scheduler.Init(),
		m.loadProjects(),
	)
}

func (m Model) loadProjects() tea.Cmd {
	svc := m.projectsService
	return func() tea.Msg {
		resp, err := svc.List(context.Background(), service.ListProjectsRequest{})
		return shared.ProjectsLoadedMsg{Projects: resp.Projects, Err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg)
	case shared.ProjectsLoadedMsg:
		if msg.Err == nil {
			m.cachedProjects = msg.Projects
			m.refreshProjectNameLookup()
		}
		return m, nil
	case shared.ProjectRegisteredMsg:
		m.cachedProjects = append(m.cachedProjects, msg.Project)
		m.refreshProjectNameLookup()
		if m.activePopup != popupNone {
			return m.routeToPopup(msg)
		}
		return m, nil
	case shared.SessionCreatedMsg:
		m.activePopup = popupNone
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		return m, cmd
	case shared.SessionCheckedOutMsg:
		m.activePopup = popupNone
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, shared.SessionCreatedMsg{Session: msg.Session})
		return m, cmd
	case shared.SessionDeleteRequestedMsg:
		m.deleteForm = sessionui.NewDeleteForm(m.styles, m.sessionsService, msg.Session)
		m.activePopup = popupDeleteSession
		return m, m.deleteForm.Init()
	case shared.SessionDeletedMsg:
		m.activePopup = popupNone
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		return m, cmd
	case shared.NewSessionPopupCloseMsg, shared.CheckoutBranchPopupCloseMsg, shared.NewSessionDeletePopupCloseMsg:
		m.activePopup = popupNone
		return m, nil
	case shared.SessionAttachReadyMsg:
		if msg.Err != nil || msg.Command == nil {
			return m, nil
		}
		return m, tea.ExecProcess(msg.Command, func(err error) tea.Msg {
			return shared.SessionAttachedMsg{Err: err}
		})
	case shared.SessionAttachedMsg:
		return m, nil
	case shared.SessionEditorLaunchedMsg:
		return m, nil
	case shared.JobsTickMsg, shared.JobsBatchMsg:
		var cmd tea.Cmd
		m.scheduler, cmd = m.scheduler.Update(msg)
		return m, cmd
	case shared.PRStatusUpdatedMsg:
		m.prStatuses[msg.SessionID] = msg
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		return m, cmd
	}

	if m.activePopup != popupNone {
		return m.routeToPopup(msg)
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if cmd, handled := m.handleKey(keyMsg); handled {
			return m, cmd
		}
		if m.leftPaneFocused {
			var cmd tea.Cmd
			m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.inspector, cmd = shared.UpdateModel(m.inspector, msg)
		return m, cmd
	}

	return m, shared.Broadcast(msg,
		shared.Forward(&m.leftPane),
		shared.Forward(&m.inspector),
	)
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if key.Matches(msg, quitKeyBinding) {
		return tea.Quit, true
	}
	if key.Matches(msg, helpMenuKeyBinding) {
		var cmd tea.Cmd
		m.helpBar, cmd = shared.UpdateModel(m.helpBar, msg)
		return cmd, true
	}
	if key.Matches(msg, nextPaneKeyBinding) {
		m.toggleLeftRightFocus()
		return nil, true
	}
	if key.Matches(msg, inspector.NextViewKeyBinding) || key.Matches(msg, inspector.PrevViewKeyBinding) {
		var cmd tea.Cmd
		m.inspector, cmd = shared.UpdateModel(m.inspector, msg)
		return cmd, true
	}
	if m.leftPaneFocused {
		if key.Matches(msg, newSessionKeyBinding) {
			m.createForm = sessionui.NewCreateForm(m.styles, m.sessionsService, m.projectsService, m.cachedProjects, m.launchers, m.editors)
			m.activePopup = popupNewSession
			return m.createForm.Init(), true
		}
		if key.Matches(msg, checkoutBranchKeyBinding) {
			m.checkoutBranchForm = sessionui.NewCheckoutBranchForm(m.styles, m.sessionsService, m.projectsService, m.cachedProjects, m.launchers, m.editors)
			m.activePopup = popupCheckoutBranch
			return m.checkoutBranchForm.Init(), true
		}
		if key.Matches(msg, attachShellKeyBinding) {
			if cmd := m.attachSelectedSessionShellCmd(); cmd != nil {
				return cmd, true
			}
		}
		if key.Matches(msg, attachAgentKeyBinding) {
			if cmd := m.attachSelectedSessionAgentCmd(); cmd != nil {
				return cmd, true
			}
		}
		if key.Matches(msg, openEditorKeyBinding) {
			if cmd := m.openSelectedSessionEditorCmd(); cmd != nil {
				return cmd, true
			}
		}
	}
	return nil, false
}

func (m Model) attachSelectedSessionShellCmd() tea.Cmd {
	idStr := m.leftPane.SelectedSessionID()
	if idStr == "" {
		return nil
	}
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}
	svc := m.sessionsService
	return func() tea.Msg {
		resp, err := svc.AttachShell(context.Background(), service.AttachShellRequest{ID: sessID})
		return shared.SessionAttachReadyMsg{Command: resp.Command, Err: err}
	}
}

func (m Model) attachSelectedSessionAgentCmd() tea.Cmd {
	idStr := m.leftPane.SelectedSessionID()
	if idStr == "" {
		return nil
	}
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}
	svc := m.sessionsService
	return func() tea.Msg {
		resp, err := svc.AttachAgent(context.Background(), service.AttachAgentRequest{ID: sessID})
		return shared.SessionAttachReadyMsg{Command: resp.Command, Err: err}
	}
}

func (m Model) openSelectedSessionEditorCmd() tea.Cmd {
	idStr := m.leftPane.SelectedSessionID()
	if idStr == "" {
		return nil
	}
	sessID, err := uuid.Parse(idStr)
	if err != nil {
		return nil
	}
	svc := m.sessionsService
	return func() tea.Msg {
		_, err := svc.OpenEditor(context.Background(), service.OpenEditorRequest{ID: sessID})
		return shared.SessionEditorLaunchedMsg{Err: err}
	}
}

func (m *Model) toggleLeftRightFocus() {
	if m.leftPaneFocused {
		m.leftPaneFocused = false
		m.leftPane.SetFocus(false)
		m.inspector.SetFocus(true)
		m.helpBar.SetBindings(detailsPanelKeyBindings)
		return
	}
	m.leftPaneFocused = true
	m.leftPane.SetFocus(true)
	m.inspector.SetFocus(false)
	m.helpBar.SetBindings(sessionsKeyBindings)
}

func (m Model) routeToPopup(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.activePopup {
	case popupNewSession:
		var cmd tea.Cmd
		m.createForm, cmd = shared.UpdateModel(m.createForm, msg)
		return m, cmd
	case popupCheckoutBranch:
		var cmd tea.Cmd
		m.checkoutBranchForm, cmd = shared.UpdateModel(m.checkoutBranchForm, msg)
		return m, cmd
	case popupDeleteSession:
		var cmd tea.Cmd
		m.deleteForm, cmd = shared.UpdateModel(m.deleteForm, msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) refreshProjectNameLookup() {
	names := make(map[uuid.UUID]string, len(m.cachedProjects))
	for _, p := range m.cachedProjects {
		names[p.ID] = p.Name
	}
	m.leftPane.SetProjectNameLookup(names)
}

func (m Model) View() tea.View {
	if m.tooSmall {
		msg := m.styles.TooSmall.Message.Render(fmt.Sprintf("Terminal too small. Minimum size: %dx%d.", m.minWidth, m.minHeight))
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
	}
	if m.activePopup != popupNone {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.popupView()))
	}

	titlebarView := m.titlebar.View().Content
	titlebarHeight := max(lipgloss.Height(titlebarView), 1)
	helpView := m.helpBar.View().Content
	helpHeight := max(lipgloss.Height(helpView), 1)

	bodyHeight := max(m.height-titlebarHeight-helpHeight, 1)
	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth

	left := fit(m.styles, m.leftPane.View().Content, leftWidth, bodyHeight)
	right := fit(m.styles, m.inspector.View().Content, rightWidth, bodyHeight)
	body := fit(m.styles, lipgloss.JoinHorizontal(lipgloss.Top, left, right), m.width, bodyHeight)
	full := lipgloss.JoinVertical(lipgloss.Left, titlebarView, body, helpView)

	return tea.NewView(full)
}

func (m Model) popupView() string {
	switch m.activePopup {
	case popupNewSession:
		return m.createForm.View().Content
	case popupCheckoutBranch:
		return m.checkoutBranchForm.View().Content
	case popupDeleteSession:
		return m.deleteForm.View().Content
	}
	return ""
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < m.minWidth || m.height < m.minHeight

	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth
	bodyHeight := max(m.height-TitleBarHeight-HelpBarHeight, 1)

	m.leftPane.SetSize(leftWidth, bodyHeight)
	m.inspector.SetSize(rightWidth, bodyHeight)
	m.helpBar.SetSize(m.width, HelpBarHeight)
	m.titlebar.SetSize(m.width, TitleBarHeight)
	return m, nil
}

func fit(s *styles.Styles, content string, width, height int) string {
	return s.Layout.Box.Width(width).Height(height).Render(content)
}
