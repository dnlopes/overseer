package dashboard

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/help"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/preview"
	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/status"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/titlebar"
	servicesession "github.com/dnlopes/overseer/internal/core/service/session"
)

type Pane int

const (
	PaneSessions Pane = iota
	PanePreview
)

type Model struct {
	titlebar     titlebar.Model
	sessionsList sessionui.Model
	statusBar    status.Model
	previewPane  preview.Model
	helpBar      help.Model
	activePane   Pane
	createForm   *sessionui.CreateFormModel
	renameForm   *sessionui.RenameFormModel
	width        int
	height       int
	tooSmall     bool
	styles       *styles.Styles
	createUC     *servicesession.CreateUseCase
	renameUC     *servicesession.RenameUseCase
	reorderUC    *servicesession.ReorderUseCase
	listUC       *servicesession.ListUseCase
}

func New(
	s *styles.Styles,
	createUC *servicesession.CreateUseCase,
	renameUC *servicesession.RenameUseCase,
	reorderUC *servicesession.ReorderUseCase,
	listUC *servicesession.ListUseCase,
	registry *help.Registry,
) Model {
	if registry == nil {
		registry = help.NewRegistry()
	}

	tb := titlebar.New(s, "Overseer")
	tbModel, _ := tb.Update(titlebar.SetActivePaneMsg{Label: "Sessions"})

	sl := sessionui.New(s, listUC)
	sb := status.New(s)
	pp := preview.New(s)
	hb := help.NewHelpBar(registry, s)

	registry.RegisterPane("sessions", append(sl.Keybindings(),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename session")),
	))
	registry.RegisterPane("preview", pp.Keybindings())

	sl.SetFocus(true)
	hb.SetActivePane("sessions")

	return Model{
		titlebar:     tbModel.(titlebar.Model),
		sessionsList: sl,
		statusBar:    sb,
		previewPane:  pp,
		helpBar:      hb,
		activePane:   PaneSessions,
		styles:       s,
		createUC:     createUC,
		renameUC:     renameUC,
		reorderUC:    reorderUC,
		listUC:       listUC,
		width:        80,
		height:       24,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.titlebar.Init(),
		m.sessionsList.Init(),
		m.statusBar.Init(),
		m.previewPane.Init(),
		m.helpBar.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg)
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	case sessionui.SessionCreatedMsg:
		m.createForm = nil
		return m, m.sessionsList.Init()
	case sessionui.SessionRenamedMsg:
		m.renameForm = nil
		return m, m.sessionsList.Init()
	case sessionui.CancelFormMsg:
		m.createForm = nil
		m.renameForm = nil
		return m, nil
	}
	if m.createForm != nil {
		var cmd tea.Cmd
		*m.createForm, cmd = updateModel(*m.createForm, msg)
		return m, cmd
	}
	if m.renameForm != nil {
		var cmd tea.Cmd
		*m.renameForm, cmd = updateModel(*m.renameForm, msg)
		return m, cmd
	}

	return m.routeToActivePane(msg)
}

func (m Model) View() tea.View {
	if m.tooSmall {
		msg := m.styles.TooSmall.Message.Render("Terminal too small. Minimum size: 60x15.")
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
	}

	titlebarView := m.viewString(m.titlebar.View())
	titlebarHeight := lipgloss.Height(titlebarView)
	if titlebarHeight < 1 {
		titlebarHeight = 1
	}
	leftWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth
	helpView := m.viewString(m.helpBar.View())
	helpHeight := lipgloss.Height(helpView)
	helpHeight = max(helpHeight, 1)
	bodyHeight := m.height - titlebarHeight - helpHeight
	bodyHeight = max(bodyHeight, 1)

	left := fit(m.viewString(m.sessionsList.View()), leftWidth, bodyHeight)
	statusView := m.viewString(m.statusBar.View())
	previewHeight := bodyHeight - lipgloss.Height(statusView)
	previewHeight = max(previewHeight, 1)
	right := lipgloss.JoinVertical(lipgloss.Left,
		fit(statusView, rightWidth, lipgloss.Height(statusView)),
		fit(m.viewString(m.previewPane.View()), rightWidth, previewHeight),
	)
	body := fit(lipgloss.JoinHorizontal(lipgloss.Top, left, right), m.width, bodyHeight)
	full := lipgloss.JoinVertical(lipgloss.Left, titlebarView, body, helpView)

	if m.createForm != nil {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewString(m.createForm.View())))
	}
	if m.renameForm != nil {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewString(m.renameForm.View())))
	}

	return tea.NewView(full)
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < 60 || m.height < 15

	leftWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.titlebar, cmd = updateModel(m.titlebar, tea.WindowSizeMsg{Width: m.width, Height: 1})
	cmds = append(cmds, cmd)

	titlebarHeight := lipgloss.Height(m.viewString(m.titlebar.View()))
	if titlebarHeight < 1 {
		titlebarHeight = 1
	}
	bodyHeight := m.height - titlebarHeight - 1
	bodyHeight = max(bodyHeight, 1)
	statusHeight := 1
	previewHeight := bodyHeight - statusHeight
	previewHeight = max(previewHeight, 1)

	m.sessionsList, cmd = updateModel(m.sessionsList, tea.WindowSizeMsg{Width: leftWidth, Height: bodyHeight})
	cmds = append(cmds, cmd)
	m.statusBar, cmd = updateModel(m.statusBar, tea.WindowSizeMsg{Width: rightWidth, Height: statusHeight})
	cmds = append(cmds, cmd)
	m.previewPane, cmd = updateModel(m.previewPane, tea.WindowSizeMsg{Width: rightWidth, Height: previewHeight})
	cmds = append(cmds, cmd)
	m.helpBar, cmd = updateModel(m.helpBar, tea.WindowSizeMsg{Width: m.width, Height: 1})
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		var cmd tea.Cmd
		m.helpBar, cmd = updateModel(m.helpBar, msg)
		return m, cmd
	}

	if m.createForm != nil {
		var cmd tea.Cmd
		*m.createForm, cmd = updateModel(*m.createForm, msg)
		return m, cmd
	}
	if m.renameForm != nil {
		var cmd tea.Cmd
		*m.renameForm, cmd = updateModel(*m.renameForm, msg)
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
	case "1":
		m.focus(PaneSessions)
		return m, nil
	case "2":
		m.focus(PanePreview)
		return m, nil
	case "n":
		if m.activePane == PaneSessions {
			cf := sessionui.NewCreateForm(m.styles, m.createUC)
			m.createForm = &cf
			return m, cf.Init()
		}
	case "r":
		if m.activePane == PaneSessions {
			if sess, ok := m.sessionsList.SelectedSession(); ok {
				rf := sessionui.NewRenameForm(m.styles, m.renameUC, sess)
				m.renameForm = &rf
				return m, rf.Init()
			}
			return m, nil
		}
	}

	return m.routeToActivePane(msg)
}

func (m *Model) focus(p Pane) {
	m.activePane = p
	m.sessionsList.SetFocus(p == PaneSessions)
	m.previewPane.SetFocus(p == PanePreview)
	if p == PaneSessions {
		m.helpBar.SetActivePane("sessions")
		m.titlebar, _ = updateModel(m.titlebar, titlebar.SetActivePaneMsg{Label: "Sessions"})
		return
	}
	m.helpBar.SetActivePane("preview")
	m.titlebar, _ = updateModel(m.titlebar, titlebar.SetActivePaneMsg{Label: "Preview"})
}

func (m Model) routeToActivePane(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.activePane == PaneSessions {
		var cmd tea.Cmd
		m.sessionsList, cmd = updateModel(m.sessionsList, msg)
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

	var cmd tea.Cmd
	m.previewPane, cmd = updateModel(m.previewPane, msg)
	return m, cmd
}

func (m Model) reorder(direction int) tea.Cmd {
	sess, ok := m.sessionsList.SelectedSession()
	if !ok || m.reorderUC == nil {
		return nil
	}
	return func() tea.Msg {
		if _, err := m.reorderUC.Execute(context.Background(), servicesession.ReorderRequest{ID: sess.ID, Direction: direction}); err != nil {
			return nil
		}
		return m.sessionsList.ReloadPreservingSelection(sess.ID)()
	}
}

func updateModel[T any](m T, msg tea.Msg) (T, tea.Cmd) {
	updated, cmd := any(m).(interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	}).Update(msg)
	return updated.(T), cmd
}

func (m Model) viewString(v tea.View) string {
	return v.Content
}

func fit(s string, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(s)
}
