package cmd

import (
	"testing"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchPriority(t *testing.T) {
	t.Parallel()

	priorities := []pagerduty.Priority{
		{APIObject: pagerduty.APIObject{ID: "PRI1"}, Name: "P1"},
		{APIObject: pagerduty.APIObject{ID: "PRI2"}, Name: "P2"},
	}

	tests := []struct {
		name       string
		input      string
		priorities []pagerduty.Priority
		want       string
		wantErr    string
	}{
		{name: "exact match", input: "P1", priorities: priorities, want: "PRI1"},
		{name: "case insensitive", input: "p2", priorities: priorities, want: "PRI2"},
		{name: "none clears", input: "none", priorities: priorities, want: ""},
		{name: "NONE case insensitive", input: "NONE", priorities: priorities, want: ""},
		{name: "no match", input: "P99", priorities: priorities, wantErr: "unknown priority"},
		{name: "empty list", input: "P1", priorities: nil, wantErr: "no priorities configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := matchPriority(tt.input, tt.priorities)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseFromEmail(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"user@example.com", "user@example.com", false},
		{"user@example", "user@example", false}, // valid per RFC 5322
		{"user.example.com", "", true},          // no @
		{"", "", true},
		{"@.", "", true},
		{"a@.", "", true},
		{"user@", "", true},
		{"@example.com", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseFromEmail(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
