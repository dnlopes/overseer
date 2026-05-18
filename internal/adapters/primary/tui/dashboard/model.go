package dashboard

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/shared"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/core/service"
)

type Pane int

const (
	PaneSessions Pane = iota
	PanePreview
	SessionsListWidthPercent = 30
	TitleBarHeight           = 1
)

type Model struct {
	titlebar     TitlebarModel
	sessionsList sessionui.Model
	helpBar      HelpModel
	createForm   *sessionui.CreateFormModel

	// this model state
	activePane      Pane
	width           int
	height          int
	tooSmall        bool
	styles          *styles.Styles
	sessionsService *service.SessionService
}

func New(styles *styles.Styles, sessionsService *service.SessionService, registry HelpRegistry) Model {
	sessions := sessionui.New(styles, sessionsService)
	helpBar := newHelpBar(registry, styles)

	registry.RegisterPane("sessions", sessions.Keybindings())
	sessions.SetFocus(true)
	helpBar.SetActivePane("sessions")

	return Model{
		titlebar:        newTitlebar(styles, "Overseer"),
		sessionsList:    sessions,
		helpBar:         helpBar,
		activePane:      PaneSessions,
		styles:          styles,
		sessionsService: sessionsService,
		width:           80,
		height:          24,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.titlebar.Init(), m.sessionsList.Init(), m.helpBar.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg)
	case sessionui.SessionCreatedMsg:
		m.createForm = nil
		return m, m.sessionsList.Init()
	case sessionui.CancelFormMsg:
		m.createForm = nil
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, shared.QuitKey) {
			return m, tea.Quit
		}
		if key.Matches(msg, shared.HelpMenuKey) {
			var cmd tea.Cmd
			m.helpBar, cmd = shared.UpdateModel(m.helpBar, msg)
			return m, cmd
		}
		if key.Matches(msg, shared.NewSessionKey) && m.createForm == nil && m.activePane == PaneSessions {
			cf := sessionui.NewCreateForm(m.styles, m.sessionsService)
			m.createForm = &cf
			return m, cf.Init()
		}
	}

	// If a form is open, route all messages to it first
	if m.createForm != nil {
		var cmd tea.Cmd
		m.createForm, cmd = shared.UpdateModel(m.createForm, msg)
		return m, cmd
	}

	return m.routeToActivePane(msg)
}

func (m Model) View() tea.View {
	if m.tooSmall {
		msg := m.styles.TooSmall.Message.Render("Terminal too small. Minimum size: 60x15.")
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
	}
	if m.createForm != nil {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.createForm.View().Content))
	}

	titlebarView := m.titlebar.View().Content
	titlebarHeight := max(lipgloss.Height(titlebarView), 1)
	helpView := m.helpBar.View().Content
	helpHeight := max(lipgloss.Height(helpView), 1)

	bodyHeight := max(m.height-titlebarHeight-helpHeight, 1)
	leftWidth := m.width * SessionsListWidthPercent / 100
	_ = m.width - leftWidth // right pane

	left := fit(m.sessionsList.View().Content, leftWidth, bodyHeight)
	right := fit("", leftWidth, bodyHeight)
	body := fit(lipgloss.JoinHorizontal(lipgloss.Top, left, right), m.width, bodyHeight)
	full := lipgloss.JoinVertical(lipgloss.Left, titlebarView, body, helpView)

	return tea.NewView(full)
}

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		var cmd tea.Cmd
		m.helpBar, cmd = shared.UpdateModel(m.helpBar, msg)
		return m, cmd
	}

	if m.createForm != nil {
		var cmd tea.Cmd
		*m.createForm, cmd = shared.UpdateModel(*m.createForm, msg)
		return m, cmd
	}

	switch msg.String() {
	case "tab", "shift+tab":
		if m.activePane == PaneSessions {
			m.focus(PanePreview)
		} else {
			m.focus(PaneSessions)
		}
		return m, nil
	case "n":
		if m.activePane == PaneSessions {
			cf := sessionui.NewCreateForm(m.styles, m.sessionsService)
			m.createForm = &cf
			return m, cf.Init()
		}
	}

	return m.routeToActivePane(msg)
}

func (m *Model) focus(p Pane) {
	m.activePane = p
	m.sessionsList.SetFocus(p == PaneSessions)
	if p == PaneSessions {
		m.helpBar.SetActivePane("sessions")
		m.titlebar, _ = shared.UpdateModel(m.titlebar, TitlebarSetActivePaneMsg{Label: "Sessions"})
		return
	}
	m.helpBar.SetActivePane("preview")
	m.titlebar, _ = shared.UpdateModel(m.titlebar, TitlebarSetActivePaneMsg{Label: "Preview"})
}

func (m Model) routeToActivePane(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sessionsList, cmd = shared.UpdateModel(m.sessionsList, msg)
	if cmd != nil {
		return m, func() tea.Msg {
			cmdMsg := cmd()
			if reorder, ok := cmdMsg.(sessionui.ReorderRequestMsg); ok {
				reorderCmd := m.reorder(reorder.Direction)
				if reorderCmd == nil {
					return nil
				}
				return reorderCmd()
			}
			return cmdMsg
		}
	}
	return m, cmd
}

func (m Model) reorder(direction int) tea.Cmd {
	sess, ok := m.sessionsList.SelectedSession()
	if !ok || m.sessionsService == nil {
		return nil
	}
	return func() tea.Msg {
		if _, err := m.sessionsService.Reorder(context.Background(), service.ReorderSessionRequest{ID: sess.ID, Direction: direction}); err != nil {
			return nil
		}
		return m.sessionsList.ReloadPreservingSelection(sess.ID)()
	}
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < 60 || m.height < 15

	leftWidth := m.width * SessionsListWidthPercent / 100
	_ = m.width - leftWidth // right width
	bodyHeight := max(m.height-TitleBarHeight, 1)

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.titlebar, cmd = shared.UpdateModel(m.titlebar, tea.WindowSizeMsg{Width: m.width, Height: TitleBarHeight})
	cmds = append(cmds, cmd)
	m.sessionsList, cmd = shared.UpdateModel(m.sessionsList, tea.WindowSizeMsg{Width: leftWidth, Height: bodyHeight})
	cmds = append(cmds, cmd)
	m.helpBar, cmd = shared.UpdateModel(m.helpBar, tea.WindowSizeMsg{Width: m.width, Height: 1})
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func fit(s string, width, height int) string {
	return lipgloss.NewStyle().Width(width).Height(height).Render(s)
}
