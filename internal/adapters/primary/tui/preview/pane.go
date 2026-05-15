package preview

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	vp := viewport.New(0, 0)
	vp.HighPerformanceRendering = false
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
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height

	case tea.KeyMsg:
		if !m.focused {
			break
		}
		switch {
		case key.Matches(msg, m.keyMap.PageUp):
			m.viewport.ViewUp()
		case key.Matches(msg, m.keyMap.PageDown):
			m.viewport.ViewDown()
		}
	}

	return m, nil
}

func (m Model) View() string {
	border := m.styles.Border.Blurred
	if m.focused {
		border = m.styles.Border.Focused
	}
	return border.Render(m.viewport.View())
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
