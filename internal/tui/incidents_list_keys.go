package tui

import "charm.land/bubbles/v2/key"

// listKeyMap holds all key bindings for the incidentList model.
type listKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	SelectUp  key.Binding
	SelectDn  key.Binding
	Toggle    key.Binding
	SelectAll key.Binding
	Deselect  key.Binding
	Open      key.Binding
	Ack       key.Binding
	Snooze    key.Binding
	Note      key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Filter    key.Binding
}

func newListKeyMap() listKeyMap {
	return listKeyMap{
		Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "move up")),
		Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "move down")),
		SelectUp:  key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+up", "select up")),
		SelectDn:  key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+down", "select down")),
		Toggle:    key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle selection")),
		SelectAll: key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select all")),
		Deselect:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "deselect all")),
		Open:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open detail")),
		Ack:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "acknowledge")),
		Snooze:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snooze")),
		Note:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "add note")),
		Top:       key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "jump to top")),
		Bottom:    key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "jump to bottom")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	}
}
