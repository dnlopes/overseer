package preview

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

const stubContent = "Stub mode: preview not available.\n\nThis pane will stream the selected session's tmux output when integration lands."

// KeyMap holds the scroll keybindings for the preview pane.
type KeyMap struct {
	PageUp   key.Binding
	PageDown key.Binding
}

// DefaultKeyMap returns pgup/pgdn scroll bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "scroll up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "scroll down"),
		),
	}
}

// Model is the BubbleTea sub-model for the bottom-right preview pane.
type Model struct {
	viewport viewport.Model
	styles   *styles.Styles
	keyMap   KeyMap
	focused  bool
	width    int
	height   int
}

// New returns a Model with stub content pre-loaded.
func New(s *styles.Styles) Model {
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	vp.SetContent(stubContent)

	return Model{
		viewport: vp,
		styles:   s,
		keyMap:   DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		border := m.border()
		m.viewport.SetWidth(max(msg.Width-border.GetHorizontalFrameSize(), 1))
		m.viewport.SetHeight(max(msg.Height-border.GetVerticalFrameSize(), 1))

	case tea.KeyPressMsg:
		if !m.focused {
			break
		}
		switch {
		case key.Matches(msg, m.keyMap.PageUp):
			m.viewport.PageUp()
		case key.Matches(msg, m.keyMap.PageDown):
			m.viewport.PageDown()
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	border := m.border()
	return tea.NewView(border.Render(m.viewport.View()))
}

func (m Model) border() lipgloss.Style {
	if m.focused {
		return m.styles.Border.Focused
	}
	return m.styles.Border.Blurred
}

// SetFocus sets whether the pane receives keyboard input.
func (m *Model) SetFocus(focused bool) {
	m.focused = focused
}

// Focused reports the current focus state.
func (m Model) Focused() bool {
	return m.focused
}

// Keybindings exposes pgup/pgdn for the help registry.
func (m Model) Keybindings() []key.Binding {
	return []key.Binding{m.keyMap.PageUp, m.keyMap.PageDown}
}
