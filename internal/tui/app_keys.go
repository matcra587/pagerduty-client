package tui

import "charm.land/bubbles/v2/key"

// appKeyMap holds all key bindings for the App model.
type appKeyMap struct {
	// Global - active in all views.
	Quit        key.Binding
	Help        key.Binding
	TogglePause key.Binding
	FilterOpts  key.Binding
	TeamSwitch  key.Binding
	CopyURL     key.Binding
	Open        key.Binding
	OpenExt     key.Binding
	Back        key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding

	// Dashboard-only - action keys for the incident list.
	Ack         key.Binding
	Resolve     key.Binding
	ResolveNow  key.Binding
	Escalate    key.Binding
	EscalateNow key.Binding
	Merge       key.Binding
	MergeNow    key.Binding
}

func newAppKeyMap() appKeyMap {
	return appKeyMap{
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		TogglePause: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "toggle refresh")),
		FilterOpts:  key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "filter options")),
		TeamSwitch:  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "switch team")),
		CopyURL:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy URL")),
		Open:        key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		OpenExt:     key.NewBinding(key.WithKeys("alt+o"), key.WithHelp("alt+o", "open external link")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Tab:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		ShiftTab:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous tab")),
		Ack:         key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "acknowledge")),
		Resolve:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "resolve")),
		ResolveNow:  key.NewBinding(key.WithKeys("alt+r"), key.WithHelp("alt+r", "resolve immediately")),
		Escalate:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "escalate")),
		EscalateNow: key.NewBinding(key.WithKeys("alt+e"), key.WithHelp("alt+e", "escalate immediately")),
		Merge:       key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge selected")),
		MergeNow:    key.NewBinding(key.WithKeys("alt+m"), key.WithHelp("alt+m", "merge immediately")),
	}
}
