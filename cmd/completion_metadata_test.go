package cmd

import (
	"strings"
	"testing"

	clibcobra "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findSubSpecByPath(t *testing.T, subs []complete.SubSpec, path ...string) complete.SubSpec {
	t.Helper()

	current := subs
	var found complete.SubSpec
	for _, segment := range path {
		matched := false
		for _, sub := range current {
			if sub.Name != segment {
				continue
			}
			found = sub
			current = sub.Subs
			matched = true
			break
		}
		require.Truef(t, matched, "subcommand path %q not found", strings.Join(path, " "))
	}

	return found
}

func TestCompletionMetadata_PositionalDynamicArgs(t *testing.T) {
	t.Parallel()

	subs := clibcobra.Subcommands(rootCmd)

	tests := []struct {
		path []string
		want []string
	}{
		{path: []string{"agent", "guide"}, want: []string{"guide"}},
		{path: []string{"ability", "test"}, want: []string{"ability"}},
		{path: []string{"config", "get"}, want: []string{"config_key"}},
		{path: []string{"config", "set"}, want: []string{"config_key", "config_value"}},
		{path: []string{"config", "unset"}, want: []string{"config_key"}},
		{path: []string{"team", "show"}, want: []string{"team"}},
		{path: []string{"service", "show"}, want: []string{"service"}},
		{path: []string{"user", "show"}, want: []string{"user"}},
		{path: []string{"schedule", "show"}, want: []string{"schedule"}},
		{path: []string{"schedule", "override"}, want: []string{"schedule"}},
		{path: []string{"maintenance-window", "show"}, want: []string{"maintenance_window"}},
		{path: []string{"escalation-policy", "show"}, want: []string{"escalation_policy"}},
		{path: []string{"incident", "show"}, want: []string{"incident"}},
		{path: []string{"incident", "ack"}, want: []string{"incident"}},
		{path: []string{"incident", "resolve"}, want: []string{"incident"}},
		{path: []string{"incident", "snooze"}, want: []string{"incident"}},
		{path: []string{"incident", "reassign"}, want: []string{"incident"}},
		{path: []string{"incident", "merge"}, want: []string{"incident"}},
		{path: []string{"incident", "note", "add"}, want: []string{"incident"}},
		{path: []string{"incident", "note", "list"}, want: []string{"incident"}},
		{path: []string{"incident", "log"}, want: []string{"incident"}},
		{path: []string{"incident", "urgency"}, want: []string{"incident", "urgency"}},
		{path: []string{"incident", "title"}, want: []string{"incident", "freeform"}},
		{path: []string{"incident", "priority"}, want: []string{"incident", "priority"}},
		{path: []string{"incident", "escalate"}, want: []string{"incident"}},
		{path: []string{"incident", "resolve-alert"}, want: []string{"incident", "alert"}},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, " "), func(t *testing.T) {
			spec := findSubSpecByPath(t, subs, tt.path...)
			assert.Equal(t, tt.want, spec.DynamicArgs)
		})
	}
}
