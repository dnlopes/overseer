package dashboard

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	teav1 "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/help"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/preview"
	sessionui "github.com/dnlopes/overseer/internal/adapters/primary/tui/session"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/status"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
	servicesession "github.com/dnlopes/overseer/internal/core/service/session"
)

type Pane int

const (
	PaneSessions Pane = iota
	PanePreview
)

type Model struct {
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

	sl := sessionui.New(s, listUC)
	sb := status.New(s)
	pp := preview.New(s)
	hb := help.NewHelpBar(registry)

	registry.RegisterPane("sessions", append(sl.Keybindings(),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new session")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename session")),
	))
	registry.RegisterPane("preview", pp.Keybindings())

	sl.SetFocus(true)
	hb.SetActivePane("sessions")

	return Model{
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
		m.sessionsList.Init(),
		adaptCmd(m.statusBar.Init()),
		adaptCmd(m.previewPane.Init()),
		adaptCmd(m.helpBar.Init()),
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

	return m.routeToActivePane(msg)
}

func (m Model) View() tea.View {
	if m.tooSmall {
		msg := m.styles.TooSmall.Message.Render("Terminal too small. Minimum size: 60x15.")
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
	}

	leftWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth
	helpView := m.helpBar.View()
	helpHeight := lipgloss.Height(helpView)
	bodyHeight := m.height - helpHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	left := fit(m.viewString(m.sessionsList.View()), leftWidth, bodyHeight)
	statusView := m.statusBar.View()
	previewHeight := bodyHeight - lipgloss.Height(statusView)
	if previewHeight < 1 {
		previewHeight = 1
	}
	right := lipgloss.JoinVertical(lipgloss.Left,
		fit(statusView, rightWidth, lipgloss.Height(statusView)),
		fit(m.previewPane.View(), rightWidth, previewHeight),
	)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	full := lipgloss.JoinVertical(lipgloss.Left, body, helpView)

	if m.createForm != nil {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.createForm.View()))
	}
	if m.renameForm != nil {
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renameForm.View()))
	}

	return tea.NewView(full)
}

func (m Model) resize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.tooSmall = m.width < 60 || m.height < 15

	leftWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth
	bodyHeight := m.height - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	statusHeight := 1
	previewHeight := bodyHeight - statusHeight
	if previewHeight < 1 {
		previewHeight = 1
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.sessionsList, cmd = updateV2(m.sessionsList, tea.WindowSizeMsg{Width: leftWidth, Height: bodyHeight})
	cmds = append(cmds, cmd)
	m.statusBar, cmd = updateV1(m.statusBar, teav1.WindowSizeMsg{Width: rightWidth, Height: statusHeight})
	cmds = append(cmds, cmd)
	m.previewPane, cmd = updateV1(m.previewPane, teav1.WindowSizeMsg{Width: rightWidth, Height: previewHeight})
	cmds = append(cmds, cmd)
	m.helpBar, cmd = updateV1(m.helpBar, teav1.WindowSizeMsg{Width: m.width, Height: 1})
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		var cmd tea.Cmd
		m.helpBar, cmd = updateV1(m.helpBar, adaptKey(msg))
		return m, cmd
	}

	if m.createForm != nil {
		var cmd tea.Cmd
		*m.createForm, cmd = updateV1(*m.createForm, adaptKey(msg))
		return m, cmd
	}
	if m.renameForm != nil {
		var cmd tea.Cmd
		*m.renameForm, cmd = updateV1(*m.renameForm, adaptKey(msg))
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
			return m, adaptCmd(cf.Init())
		}
	case "r":
		if m.activePane == PaneSessions {
			if sess, ok := m.sessionsList.SelectedSession(); ok {
				rf := sessionui.NewRenameForm(m.styles, m.renameUC, sess)
				m.renameForm = &rf
				return m, adaptCmd(rf.Init())
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
		return
	}
	m.helpBar.SetActivePane("preview")
}

func (m Model) routeToActivePane(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.activePane == PaneSessions {
		var cmd tea.Cmd
		m.sessionsList, cmd = updateV2(m.sessionsList, msg)
		if cmd != nil {
			cmdMsg := cmd()
			if reorder, ok := cmdMsg.(sessionui.ReorderRequestMsg); ok {
				return m, m.reorder(reorder.Direction)
			}
			return m, func() tea.Msg { return cmdMsg }
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.previewPane, cmd = updateV1(m.previewPane, adaptMsg(msg))
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
		return m.sessionsList.Init()()
	}
}

func updateV2[T any](m T, msg tea.Msg) (T, tea.Cmd) {
	updated, cmd := any(m).(interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	}).Update(msg)
	return updated.(T), cmd
}

func updateV1[T any](m T, msg teav1.Msg) (T, tea.Cmd) {
	updated, cmd := any(m).(interface {
		Update(teav1.Msg) (teav1.Model, teav1.Cmd)
	}).Update(msg)
	return updated.(T), adaptCmd(cmd)
}

func adaptCmd(cmd teav1.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg { return cmd() }
}

func adaptMsg(msg tea.Msg) teav1.Msg {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return teav1.WindowSizeMsg{Width: msg.Width, Height: msg.Height}
	case tea.KeyPressMsg:
		return adaptKey(msg)
	default:
		return msg
	}
}

func adaptKey(msg tea.KeyPressMsg) teav1.KeyMsg {
	switch msg.String() {
	case "tab":
		return teav1.KeyMsg{Type: teav1.KeyTab}
	case "shift+tab":
		return teav1.KeyMsg{Type: teav1.KeyShiftTab}
	case "enter":
		return teav1.KeyMsg{Type: teav1.KeyEnter}
	case "esc":
		return teav1.KeyMsg{Type: teav1.KeyEsc}
	case "ctrl+c":
		return teav1.KeyMsg{Type: teav1.KeyCtrlC}
	case "backspace":
		return teav1.KeyMsg{Type: teav1.KeyBackspace}
	case "ctrl+k":
		return teav1.KeyMsg{Type: teav1.KeyCtrlK}
	case "ctrl+u":
		return teav1.KeyMsg{Type: teav1.KeyCtrlU}
	case "ctrl+a":
		return teav1.KeyMsg{Type: teav1.KeyCtrlA}
	default:
		return teav1.KeyMsg{Type: teav1.KeyRunes, Runes: []rune(msg.String())}
	}
}

func (m Model) viewString(v tea.View) string {
	if s, ok := v.Layer.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprint(v.Layer)
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
