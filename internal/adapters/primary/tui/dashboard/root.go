package dashboard

import (
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
	SessionsListWidthPercent = 30
	TitleBarHeight           = 1
	HelpBarHeight            = 1
	RootDefaultWidth         = 80
	RootDefaultHeight        = 24
	SessionsPaneID           = "sessions"
	DetailsPaneID            = "details"
)

type Model struct {
	// children
	titlebar      TitleBarModel
	sessionsModel sessionui.Model
	detailsModel  DetailsModel
	helpBar       HelpBarModel
	createForm    sessionui.CreateFormModel

	// model state
	width           int
	height          int
	tooSmall        bool
	focused         bool
	styles          *styles.Styles
	sessionsService service.SessionService
}

func New(styles *styles.Styles, sessionsService service.SessionService, registry HelpRegistry) Model {
	m := Model{styles: styles, titlebar: newTitlebar(styles, "Overseer"), width: RootDefaultWidth, height: RootDefaultHeight, focused: true,
		sessionsModel:   sessionui.New(styles, sessionsService),
		helpBar:         newHelpBar(registry, styles),
		detailsModel:    newDetailsModel(*styles),
		sessionsService: sessionsService,
	}

	m.sessionsModel.SetFocus(true)
	m.helpBar.SetActivePane(SessionsPaneID)
	registry.RegisterPane(SessionsPaneID, m.sessionsModel.KeyBindings())
	registry.RegisterPane(DetailsPaneID, m.detailsModel.KeyBindings())

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.titlebar.Init(), m.sessionsModel.Init(), m.detailsModel.Init(), m.helpBar.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg)
	case shared.SessionCreatedMsg:
		return m, m.sessionsModel.Init()
	case shared.NewSessionPopupCloseMsg:
		m.focused = true
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, quitKeyBinding) && m.focused {
			return m, tea.Quit
		}
		if key.Matches(msg, helpMenuKeyBinding) && m.focused {
			var cmd tea.Cmd
			m.helpBar, cmd = shared.UpdateModel(m.helpBar, msg)
			return m, cmd
		}
		if key.Matches(msg, newSessionKeyBinding) && m.focused && m.sessionsModel.IsFocused() {
			m.createForm = sessionui.NewCreateForm(m.styles, m.sessionsService)
			m.focused = false
			return m, m.createForm.Init()
		}
		if key.Matches(msg, nextTabKeyBinding) && m.focused {
			if m.sessionsModel.IsFocused() {
				m.sessionsModel.SetFocus(false)
				m.detailsModel.SetFocus(true)
				m.helpBar.SetActivePane(DetailsPaneID)
			} else {
				m.sessionsModel.SetFocus(true)
				m.detailsModel.SetFocus(false)
				m.helpBar.SetActivePane(SessionsPaneID)
			}
			return m, nil
		}
	}

	// If a form is open, route all messages to it first
	if !m.focused {
		var cmd tea.Cmd
		m.createForm, cmd = shared.UpdateModel(m.createForm, msg)
		return m, cmd
	}

	// forward to the focused child
	var cmd tea.Cmd
	if m.sessionsModel.IsFocused() {
		m.sessionsModel, cmd = shared.UpdateModel(m.sessionsModel, msg)
		return m, cmd
	} else {
		m.detailsModel, cmd = shared.UpdateModel(m.detailsModel, msg)
		return m, cmd
	}
}

func (m Model) View() tea.View {
	if m.tooSmall {
		msg := m.styles.TooSmall.Message.Render("Terminal too small. Minimum size: 60x15.")
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
	}
	if !m.focused {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.createForm.View().Content))
	}

	titlebarView := m.titlebar.View().Content
	titlebarHeight := max(lipgloss.Height(titlebarView), 1)
	helpView := m.helpBar.View().Content
	helpHeight := max(lipgloss.Height(helpView), 1)

	bodyHeight := max(m.height-titlebarHeight-helpHeight, 1)
	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth

	left := fit(m.styles, m.sessionsModel.View().Content, leftWidth, bodyHeight)
	right := fit(m.styles, m.detailsModel.View().Content, rightWidth, bodyHeight)
	body := fit(m.styles, lipgloss.JoinHorizontal(lipgloss.Top, left, right), m.width, bodyHeight)
	full := lipgloss.JoinVertical(lipgloss.Left, titlebarView, body, helpView)

	return tea.NewView(full)
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < 60 || m.height < 15

	leftWidth := m.width * SessionsListWidthPercent / 100
	rightWidth := m.width - leftWidth
	bodyHeight := max(m.height-TitleBarHeight-HelpBarHeight, 1)

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.sessionsModel.SetSize(leftWidth, bodyHeight)
	m.detailsModel.SetSize(rightWidth, bodyHeight)

	m.titlebar, cmd = shared.UpdateModel(m.titlebar, tea.WindowSizeMsg{Width: m.width, Height: TitleBarHeight})
	cmds = append(cmds, cmd)
	m.sessionsModel, cmd = shared.UpdateModel(m.sessionsModel, tea.WindowSizeMsg{Width: leftWidth, Height: bodyHeight})
	cmds = append(cmds, cmd)
	m.detailsModel, cmd = shared.UpdateModel(m.detailsModel, tea.WindowSizeMsg{Width: rightWidth, Height: bodyHeight})
	cmds = append(cmds, cmd)
	m.helpBar, cmd = shared.UpdateModel(m.helpBar, tea.WindowSizeMsg{Width: m.width, Height: HelpBarHeight})
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func fit(_ *styles.Styles, s string, width, height int) string {
	return lipgloss.NewStyle().Width(width).Height(height).Render(s)
}
