package shared

import tea "charm.land/bubbletea/v2"

func UpdateModel[T any](m T, msg tea.Msg) (T, tea.Cmd) {
	updated, cmd := any(m).(interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	}).Update(msg)

	return updated.(T), cmd
}
