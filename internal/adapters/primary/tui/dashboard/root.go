package dashboard

import (
	"context"
	"fmt"
	"slices"
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
	TitleBarGap              = 1
	HelpBarHeight            = 1
)

type popupKind int

const (
	popupNone popupKind = iota
	popupNewSession
	popupDeleteSession
	popupRename
	popupHelp
)

type projectBranchCache struct {
	branches      []domain.BranchInfo
	defaultBranch string
	loadedAt      time.Time
}

type Model struct {
	titlebar                TitleBarModel
	leftPane                leftpane.Model
	inspector               inspector.Model
	helpBar                 shared.HelpBarModel
	createForm              sessionui.CreateFormModel
	deleteForm              sessionui.DeleteFormModel
	renameForm              sessionui.RenameFormModel
	helpPopup               shared.HelpPopupModel
	scheduler               jobs.Model
	activePopup             popupKind
	cachedProjects          []domain.Project
	cachedBranchesByProject map[uuid.UUID]projectBranchCache
	prStatuses              map[uuid.UUID]shared.PRStatusUpdatedMsg

	width           int
	height          int
	minWidth        int
	minHeight       int
	tooSmall        bool
	styles          *styles.Styles
	sessionsService service.SessionService
	projectsService service.ProjectService
	launchers       []domain.Launcher
	editors         []domain.Editor
	labels          []domain.Label
}

func New(
	styles *styles.Styles,
	sessionsService service.SessionService,
	projectsService service.ProjectService,
	scheduler jobs.Model,
	launchers []domain.Launcher,
	editors []domain.Editor,
	labels []domain.Label,
	minWidth, minHeight int,
	previewRefreshInterval time.Duration,
) Model {
	sessionsModel := sessionui.New(styles, sessionsService, labels)
	detailsModel := sessiondetails.New(styles)
	left := leftpane.New(styles, sessionsModel, detailsModel)
	left.SetFocus(true)
	m := Model{
		styles:                  styles,
		titlebar:                newTitlebar(styles, "overseer"),
		leftPane:                left,
		inspector:               inspector.New(styles, sessionsService, previewRefreshInterval),
		helpBar:                 shared.NewHelpBarModel(styles, slices.Concat(sessionsKeyBindings, inspectorKeyBindings, generalKeyBindings)),
		scheduler:               scheduler,
		sessionsService:         sessionsService,
		projectsService:         projectsService,
		labels:                  labels,
		launchers:               launchers,
		editors:                 editors,
		minWidth:                minWidth,
		minHeight:               minHeight,
		prStatuses:              make(map[uuid.UUID]shared.PRStatusUpdatedMsg),
		cachedBranchesByProject: make(map[uuid.UUID]projectBranchCache),
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
		m.scheduleBranchTick(),
	)
}

func (m Model) loadProjects() tea.Cmd {
	svc := m.projectsService
	return func() tea.Msg {
		resp, err := svc.List(context.Background(), service.ListProjectsRequest{})
		return shared.ProjectsLoadedMsg{Projects: resp.Projects, Err: err}
	}
}

func (m Model) scheduleBranchTick() tea.Cmd {
	return tea.Tick(BranchCacheRefreshInterval, func(time.Time) tea.Msg {
		return shared.BranchCacheTickMsg{}
	})
}

func (m Model) loadBranchesForProjectCmd(projectID uuid.UUID) tea.Cmd {
	svc := m.sessionsService
	return func() tea.Msg {
		resp, err := svc.ListBranches(context.Background(), service.ListBranchesRequest{ProjectID: projectID})
		return shared.BranchesLoadedMsg{
			ProjectID:     projectID,
			Branches:      resp.Branches,
			DefaultBranch: resp.DefaultBranch,
			LoadedAt:      time.Now(),
			Err:           err,
		}
	}
}

func (m Model) loadCurrentBranchCmd(projectID uuid.UUID) tea.Cmd {
	svc := m.sessionsService
	return func() tea.Msg {
		resp, err := svc.ProjectCurrentBranch(context.Background(), service.ProjectCurrentBranchRequest{ProjectID: projectID})
		return shared.ProjectCurrentBranchLoadedMsg{ProjectID: projectID, Branch: resp.Branch, Err: err}
	}
}

func (m Model) fanOutBranchRefresh() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.cachedProjects))
	for _, p := range m.cachedProjects {
		cmds = append(cmds, m.loadBranchesForProjectCmd(p.ID))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) branchesByProjectFlat() map[uuid.UUID][]domain.BranchInfo {
	out := make(map[uuid.UUID][]domain.BranchInfo, len(m.cachedBranchesByProject))
	for k, v := range m.cachedBranchesByProject {
		out[k] = v.branches
	}
	return out
}

func (m Model) defaultBranchesByProject() map[uuid.UUID]string {
	out := make(map[uuid.UUID]string, len(m.cachedBranchesByProject))
	for k, v := range m.cachedBranchesByProject {
		if v.defaultBranch != "" {
			out[k] = v.defaultBranch
		}
	}
	return out
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg)
	case shared.ProjectsLoadedMsg:
		if msg.Err == nil {
			m.cachedProjects = msg.Projects
			m.refreshProjectNameLookup()
			return m, m.fanOutBranchRefresh()
		}
		return m, nil
	case shared.BranchesLoadedMsg:
		if msg.Err == nil {
			m.cachedBranchesByProject[msg.ProjectID] = projectBranchCache{
				branches:      msg.Branches,
				defaultBranch: msg.DefaultBranch,
				loadedAt:      msg.LoadedAt,
			}
		}
		if m.activePopup == popupNewSession {
			return m.routeToPopup(msg)
		}
		return m, nil
	case shared.BranchCacheTickMsg:
		return m, tea.Batch(m.fanOutBranchRefresh(), m.scheduleBranchTick())
	case shared.ProjectCurrentBranchLoadedMsg:
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		return m, cmd
	case shared.ProjectRegisteredMsg:
		m.cachedProjects = append(m.cachedProjects, msg.Project)
		m.refreshProjectNameLookup()
		cmds := []tea.Cmd{m.loadBranchesForProjectCmd(msg.Project.ID)}
		if m.activePopup != popupNone {
			updated, popupCmd := m.routeToPopup(msg)
			m = updated.(Model)
			cmds = append(cmds, popupCmd)
		}
		return m, tea.Batch(cmds...)
	case shared.SessionCreatedMsg:
		m.activePopup = popupNone
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
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
	case shared.SessionRenameRequestedMsg:
		m.renameForm = sessionui.NewRenameSessionForm(m.styles, m.sessionsService, msg.Session, m.width)
		m.activePopup = popupRename
		return m, m.renameForm.Init()
	case shared.ProjectRenameRequestedMsg:
		m.renameForm = sessionui.NewRenameProjectForm(m.styles, m.projectsService, msg.ProjectID, msg.CurrentName, m.width)
		m.activePopup = popupRename
		return m, m.renameForm.Init()
	case shared.SessionRenamedMsg:
		m.activePopup = popupNone
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		return m, cmd
	case shared.ProjectRenamedMsg:
		m.activePopup = popupNone
		m.applyRenamedProject(msg.Project)
		return m, nil
	case shared.SessionSelectedMsg:
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
		var inspectorCmd tea.Cmd
		m.inspector, inspectorCmd = shared.UpdateModel(m.inspector, msg)
		if !msg.Session.HasWorktree() {
			return m, tea.Batch(cmd, inspectorCmd, m.loadCurrentBranchCmd(msg.Session.ProjectID))
		}
		return m, tea.Batch(cmd, inspectorCmd)
	case shared.NewSessionPopupCloseMsg, shared.NewSessionDeletePopupCloseMsg, shared.RenamePopupCloseMsg, shared.HelpPopupCloseMsg:
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
		var cmd tea.Cmd
		m.leftPane, cmd = shared.UpdateModel(m.leftPane, msg)
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
		m.helpPopup = shared.NewHelpPopupModel(m.styles, sessionsHelpGroups, m.width)
		m.activePopup = popupHelp
		return m.helpPopup.Init(), true
	}
	if key.Matches(msg, inspector.ToggleViewKeyBinding) {
		var cmd tea.Cmd
		m.inspector, cmd = shared.UpdateModel(m.inspector, msg)
		return cmd, true
	}
	if key.Matches(msg, newSessionKeyBinding) {
		initialProjectID := m.cursorProjectID()
		m.createForm = sessionui.NewCreateForm(
			m.styles,
			m.sessionsService,
			m.projectsService,
			m.cachedProjects,
			initialProjectID,
			m.branchesByProjectFlat(),
			m.defaultBranchesByProject(),
			m.launchers,
			m.editors,
			m.width,
		)
		m.activePopup = popupNewSession
		cmds := []tea.Cmd{m.createForm.Init()}
		if refresh := m.refreshStaleProjectBranchesCmd(initialProjectID); refresh != nil {
			cmds = append(cmds, refresh)
		}
		return tea.Batch(cmds...), true
	}
	if key.Matches(msg, attachKeyBinding) {
		if m.inspector.ActiveViewLabel() == "Shell" {
			if cmd := m.attachSelectedSessionShellCmd(); cmd != nil {
				return cmd, true
			}
		} else {
			if cmd := m.attachSelectedSessionAgentCmd(); cmd != nil {
				return cmd, true
			}
		}
	}
	if key.Matches(msg, openEditorKeyBinding) {
		if cmd := m.openSelectedSessionEditorCmd(); cmd != nil {
			return cmd, true
		}
	}
	return nil, false
}

func (m Model) refreshStaleProjectBranchesCmd(initialProjectID uuid.UUID) tea.Cmd {
	if initialProjectID == uuid.Nil {
		return nil
	}
	cache, ok := m.cachedBranchesByProject[initialProjectID]
	if ok && time.Since(cache.loadedAt) < BranchCacheStaleThreshold {
		return nil
	}
	return m.loadBranchesForProjectCmd(initialProjectID)
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

func (m Model) routeToPopup(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.activePopup {
	case popupNewSession:
		var cmd tea.Cmd
		m.createForm, cmd = shared.UpdateModel(m.createForm, msg)
		return m, cmd
	case popupDeleteSession:
		var cmd tea.Cmd
		m.deleteForm, cmd = shared.UpdateModel(m.deleteForm, msg)
		return m, cmd
	case popupRename:
		var cmd tea.Cmd
		m.renameForm, cmd = shared.UpdateModel(m.renameForm, msg)
		return m, cmd
	case popupHelp:
		var cmd tea.Cmd
		m.helpPopup, cmd = shared.UpdateModel(m.helpPopup, msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) applyRenamedProject(p domain.Project) {
	for i := range m.cachedProjects {
		if m.cachedProjects[i].ID == p.ID {
			m.cachedProjects[i] = p
			break
		}
	}
	m.refreshProjectNameLookup()
}

func (m Model) cursorProjectID() uuid.UUID {
	if sess, ok := m.leftPane.SelectedSession(); ok {
		return sess.ProjectID
	}
	return uuid.Nil
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

	bodyHeight := max(m.height-titlebarHeight-TitleBarGap-helpHeight, 1)
	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth

	left := fit(m.styles, m.leftPane.View().Content, leftWidth, bodyHeight)
	right := fit(m.styles, m.inspector.View().Content, rightWidth, bodyHeight)
	body := fit(m.styles, lipgloss.JoinHorizontal(lipgloss.Top, left, right), m.width, bodyHeight)
	full := lipgloss.JoinVertical(lipgloss.Left, titlebarView, "", body, helpView)

	return tea.NewView(full)
}

func (m Model) popupView() string {
	switch m.activePopup {
	case popupNewSession:
		return m.createForm.View().Content
	case popupDeleteSession:
		return m.deleteForm.View().Content
	case popupRename:
		return m.renameForm.View().Content
	case popupHelp:
		return m.helpPopup.View().Content
	}
	return ""
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < m.minWidth || m.height < m.minHeight

	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth
	bodyHeight := max(m.height-TitleBarHeight-TitleBarGap-HelpBarHeight, 1)

	m.leftPane.SetSize(leftWidth, bodyHeight)
	m.inspector.SetSize(rightWidth, bodyHeight)
	m.helpBar.SetSize(m.width, HelpBarHeight)
	m.titlebar.SetSize(m.width, TitleBarHeight)
	return m, nil
}

func fit(s *styles.Styles, content string, width, height int) string {
	return s.Layout.Box.Width(width).Height(height).Render(content)
}
