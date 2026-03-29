package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestAppKeyMap_BindingsMatch(t *testing.T) {
	t.Parallel()
	km := newAppKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"quit q", km.Quit, []string{"q"}},
		{"quit ctrl+c", km.Quit, []string{"ctrl+c"}},
		{"help", km.Help, []string{"?"}},
		{"toggle pause", km.TogglePause, []string{"R"}},
		{"filter opts", km.FilterOpts, []string{"O"}},
		{"team switch", km.TeamSwitch, []string{"t"}},
		{"copy url", km.CopyURL, []string{"y"}},
		{"open", km.Open, []string{"o"}},
		{"open ext", km.OpenExt, []string{"alt+o"}},
		{"back", km.Back, []string{"esc"}},
		{"tab", km.Tab, []string{"tab"}},
		{"shift tab", km.ShiftTab, []string{"shift+tab"}},
		{"ack", km.Ack, []string{"a"}},
		{"resolve", km.Resolve, []string{"r"}},
		{"resolve now", km.ResolveNow, []string{"alt+r"}},
		{"edit", km.Edit, []string{"e"}},
		{"escalate", km.Escalate, []string{"x"}},
		{"escalate now", km.EscalateNow, []string{"alt+x"}},
		{"merge", km.Merge, []string{"m"}},
		{"merge now", km.MergeNow, []string{"alt+m"}},
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
