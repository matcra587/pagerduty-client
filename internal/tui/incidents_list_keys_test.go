package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestListKeyMap_BindingsMatch(t *testing.T) {
	t.Parallel()
	km := newListKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"up k", km.Up, []string{"k"}},
		{"up arrow", km.Up, []string{"up"}},
		{"down j", km.Down, []string{"j"}},
		{"down arrow", km.Down, []string{"down"}},
		{"select up", km.SelectUp, []string{"shift+up"}},
		{"select down", km.SelectDn, []string{"shift+down"}},
		{"toggle", km.Toggle, []string{"space"}},
		{"select all", km.SelectAll, []string{"ctrl+a"}},
		{"deselect", km.Deselect, []string{"esc"}},
		{"open", km.Open, []string{"enter"}},
		{"ack", km.Ack, []string{"a"}},
		{"snooze", km.Snooze, []string{"s"}},
		{"note", km.Note, []string{"n"}},
		{"top", km.Top, []string{"g"}},
		{"bottom", km.Bottom, []string{"G"}},
		{"filter", km.Filter, []string{"/"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, k := range tt.keys {
				// Code -2 forces String() to return Text as-is,
				// bypassing rune/special-key logic.
				msg := tea.KeyPressMsg{Code: -2, Text: k}
				assert.True(t, key.Matches(msg, tt.binding),
					"key.Matches should match %q for binding %q", k, tt.name)
			}
		})
	}
}
