package dashboard

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

const previewStubContent = "Stub mode: preview not available.\n\nThis pane will stream the selected session's tmux output when integration lands."

// previewKeyMap holds the scroll keybindings for the preview pane.
type previewKeyMap struct {
	PageUp   key.Binding
	PageDown key.Binding
}

func defaultPreviewKeyMap() previewKeyMap {
	return previewKeyMap{
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

// PreviewModel is the BubbleTea sub-model for the bottom-right preview pane.
type PreviewModel struct {
	viewport viewport.Model
	styles   *styles.Styles
	keyMap   previewKeyMap
	focused  bool
	width    int
	height   int
}

func newPreview(s *styles.Styles) PreviewModel {
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	vp.SetContent(previewStubContent)

	return PreviewModel{
		viewport: vp,
		styles:   s,
		keyMap:   defaultPreviewKeyMap(),
	}
}

func (m PreviewModel) Init() tea.Cmd { return nil }

func (m PreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		border := m.border()
		pane := m.styles.Pane.Preview
		m.viewport.SetWidth(max(msg.Width-border.GetHorizontalFrameSize()-pane.GetHorizontalPadding(), 1))
		m.viewport.SetHeight(max(msg.Height-border.GetVerticalFrameSize()-pane.GetVerticalPadding(), 1))

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

func (m PreviewModel) View() tea.View {
	border := m.border()
	inner := m.styles.Pane.Preview.Render(m.viewport.View())
	return tea.NewView(border.Render(inner))
}

func (m PreviewModel) border() lipgloss.Style {
	if m.focused {
		return m.styles.Border.Focused
	}
	return m.styles.Border.Blurred
}

func (m *PreviewModel) SetFocus(focused bool) {
	m.focused = focused
}

func (m PreviewModel) Focused() bool {
	return m.focused
}

func (m PreviewModel) Keybindings() []key.Binding {
	return []key.Binding{m.keyMap.PageUp, m.keyMap.PageDown}
}
