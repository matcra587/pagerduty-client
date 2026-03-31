package cmd

import (
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
)

func TestEscalationRuleRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		rules    []pagerduty.EscalationRule
		wantRows [][]string
	}{
		{
			name: "single target",
			rules: []pagerduty.EscalationRule{
				{Delay: 5, Targets: []pagerduty.APIObject{{ID: "U1", Type: "user_reference", Summary: "Alice"}}},
			},
			wantRows: [][]string{{"1", "5 min", "Alice (user)"}},
		},
		{
			name: "multiple targets mixed types",
			rules: []pagerduty.EscalationRule{
				{Delay: 10, Targets: []pagerduty.APIObject{
					{ID: "U1", Type: "user_reference", Summary: "Alice"},
					{ID: "S1", Type: "schedule_reference", Summary: "Primary"},
				}},
			},
			wantRows: [][]string{{"1", "10 min", "Alice (user), Primary (schedule)"}},
		},
		{
			name: "empty targets",
			rules: []pagerduty.EscalationRule{
				{Delay: 5, Targets: nil},
			},
			wantRows: [][]string{{"1", "5 min", ""}},
		},
		{
			name: "target without summary falls back to ID",
			rules: []pagerduty.EscalationRule{
				{Delay: 5, Targets: []pagerduty.APIObject{{ID: "U1", Type: "user_reference"}}},
			},
			wantRows: [][]string{{"1", "5 min", "U1 (user)"}},
		},
		{
			name: "multiple rules",
			rules: []pagerduty.EscalationRule{
				{Delay: 5, Targets: []pagerduty.APIObject{{ID: "U1", Type: "user_reference", Summary: "Alice"}}},
				{Delay: 15, Targets: []pagerduty.APIObject{{ID: "U2", Type: "user_reference", Summary: "Bob"}}},
			},
			wantRows: [][]string{
				{"1", "5 min", "Alice (user)"},
				{"2", "15 min", "Bob (user)"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			headers, rows := escalationRuleRows(tt.rules)
			assert.Equal(t, []string{"Level", "Delay", "Targets"}, headers)
			assert.Equal(t, tt.wantRows, rows)
		})
	}
}
