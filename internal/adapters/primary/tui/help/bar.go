package help

import (
	bubblehelp "charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// keyMapAdapter wraps a flat []key.Binding to satisfy bubblehelp.KeyMap.
type keyMapAdapter struct {
	bindings []key.Binding
}

func (k keyMapAdapter) ShortHelp() []key.Binding    { return k.bindings }
func (k keyMapAdapter) FullHelp() [][]key.Binding   { return [][]key.Binding{k.bindings} }

// barKeys holds the bindings consumed by the help bar itself.
type barKeys struct {
	toggleHelp key.Binding
}

func defaultBarKeys() barKeys {
	return barKeys{
		toggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// Model is the BubbleTea sub-model for the help bar.
type Model struct {
	help       bubblehelp.Model
	registry   *Registry
	activePane string
	showFull   bool
	keys       barKeys
}

// NewHelpBar returns a Model wired to registry.
func NewHelpBar(registry *Registry) Model {
	return Model{
		help:     newHelp(),
		registry: registry,
		keys:     defaultBarKeys(),
	}
}

func newHelp() bubblehelp.Model {
	h := bubblehelp.New()
	h.SetWidth(80)
	h.Styles = bubblehelp.Styles{}
	return h
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.toggleHelp) {
			m.showFull = !m.showFull
			m.help.ShowAll = m.showFull
		}
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)
	}
	return m, nil
}

func (m Model) View() tea.View {
	km := keyMapAdapter{bindings: m.registry.BindingsFor(m.activePane)}
	return tea.NewView(m.help.View(km))
}

// SetActivePane updates which pane's bindings appear in the bar.
func (m *Model) SetActivePane(name string) {
	m.activePane = name
}
