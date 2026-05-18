package dashboard

import (
	bubblehelp "charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/dnlopes/overseer/internal/adapters/primary/tui/styles"
)

// HelpRegistry maps pane names to their keybindings plus global bindings shown
// regardless of which pane is active.
type HelpRegistry struct {
	paneBindings   map[string][]key.Binding
	globalBindings []key.Binding
}

// NewHelpRegistry returns a HelpRegistry pre-populated with application-wide bindings:
// quit, next-pane, toggle-help, and the numeric pane-jump keys.
func NewHelpRegistry() HelpRegistry {
	return HelpRegistry{
		paneBindings: make(map[string][]key.Binding),
		globalBindings: []key.Binding{
			key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		},
	}
}

// RegisterPane adds or replaces the keybindings for the named pane.
func (r *HelpRegistry) RegisterPane(name string, bindings []key.Binding) {
	r.paneBindings[name] = bindings
}

// BindingsFor returns the bindings for the named pane followed by the global bindings.
func (r *HelpRegistry) BindingsFor(name string) []key.Binding {
	pane := r.paneBindings[name]
	combined := make([]key.Binding, 0, len(pane)+len(r.globalBindings))
	combined = append(combined, pane...)
	combined = append(combined, r.globalBindings...)
	return combined
}

// helpKeyMapAdapter wraps a flat []key.Binding to satisfy bubblehelp.KeyMap.
type helpKeyMapAdapter struct {
	bindings []key.Binding
}

func (k helpKeyMapAdapter) ShortHelp() []key.Binding  { return k.bindings }
func (k helpKeyMapAdapter) FullHelp() [][]key.Binding { return [][]key.Binding{k.bindings} }

// helpBarKeys holds the bindings consumed by the help bar itself.
type helpBarKeys struct {
	toggleHelp key.Binding
}

func defaultHelpBarKeys() helpBarKeys {
	return helpBarKeys{
		toggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// HelpModel is the BubbleTea sub-model for the help bar.
type HelpModel struct {
	help       bubblehelp.Model
	registry   HelpRegistry
	activePane string
	showFull   bool
	keys       helpBarKeys
}

func newHelpBar(registry HelpRegistry, s *styles.Styles) HelpModel {
	return HelpModel{
		help:     newHelpComponent(s),
		registry: registry,
		keys:     defaultHelpBarKeys(),
	}
}

func newHelpComponent(s *styles.Styles) bubblehelp.Model {
	h := bubblehelp.New()
	h.SetWidth(80)
	if s != nil {
		h.Styles = bubblehelp.Styles{
			ShortKey:       s.Help.Key,
			ShortDesc:      s.Help.Description,
			ShortSeparator: s.Help.Separator,
			FullKey:        s.Help.Key,
			FullDesc:       s.Help.Description,
			FullSeparator:  s.Help.Separator,
			Ellipsis:       s.Help.Separator,
		}
	}
	return h
}

func (m HelpModel) Init() tea.Cmd { return nil }

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m HelpModel) View() tea.View {
	km := helpKeyMapAdapter{bindings: m.registry.BindingsFor(m.activePane)}
	return tea.NewView(m.help.View(km))
}

func (m *HelpModel) SetActivePane(name string) {
	m.activePane = name
}
