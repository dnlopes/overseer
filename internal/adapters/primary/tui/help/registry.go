// Package help provides a keybinding registry and help-bar sub-model that
// integrates with charm.land/bubbles/v2/help.
package help

import "charm.land/bubbles/v2/key"

// Registry maps pane names to their keybindings plus global bindings shown
// regardless of which pane is active.
type Registry struct {
	paneBindings   map[string][]key.Binding
	globalBindings []key.Binding
}

// NewRegistry returns a Registry pre-populated with application-wide bindings:
// quit, next-pane, toggle-help, and the numeric pane-jump keys.
func NewRegistry() *Registry {
	return &Registry{
		paneBindings: make(map[string][]key.Binding),
		globalBindings: []key.Binding{
			key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "jump to pane 1")),
			key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "jump to pane 2")),
		},
	}
}

// RegisterPane adds or replaces the keybindings for the named pane.
// Subsequent calls with the same name overwrite the previous bindings.
func (r *Registry) RegisterPane(name string, bindings []key.Binding) {
	r.paneBindings[name] = bindings
}

// BindingsFor returns the bindings for the named pane followed by the global
// bindings. If no bindings have been registered for name, only the globals
// are returned.
func (r *Registry) BindingsFor(name string) []key.Binding {
	pane := r.paneBindings[name]
	combined := make([]key.Binding, 0, len(pane)+len(r.globalBindings))
	combined = append(combined, pane...)
	combined = append(combined, r.globalBindings...)
	return combined
}
