package cmd

import (
	"bytes"
	"testing"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/gechr/clib/theme"
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

func TestFormatAssignees(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		assignments []pagerduty.Assignment
		want        string
	}{
		{
			name: "single assignee",
			assignments: []pagerduty.Assignment{
				{Assignee: pagerduty.APIObject{Summary: "Jane Smith"}},
			},
			want: "Jane Smith",
		},
		{
			name: "multiple assignees",
			assignments: []pagerduty.Assignment{
				{Assignee: pagerduty.APIObject{Summary: "Jane Smith"}},
				{Assignee: pagerduty.APIObject{Summary: "John Doe"}},
			},
			want: "Jane Smith, John Doe",
		},
		{
			name:        "nil assignments",
			assignments: nil,
			want:        "",
		},
		{
			name: "empty summary skipped",
			assignments: []pagerduty.Assignment{
				{Assignee: pagerduty.APIObject{Summary: "Jane Smith"}},
				{Assignee: pagerduty.APIObject{Summary: ""}},
				{Assignee: pagerduty.APIObject{Summary: "John Doe"}},
			},
			want: "Jane Smith, John Doe",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatAssignees(tt.assignments))
		})
	}
}

func TestFormatAlertCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		triggered uint
		resolved  uint
		want      string
	}{
		{name: "mixed", triggered: 3, resolved: 2, want: "5 total, 3 triggered, 2 resolved"},
		{name: "all triggered", triggered: 4, resolved: 0, want: "4 total, 4 triggered, 0 resolved"},
		{name: "all resolved", triggered: 0, resolved: 7, want: "7 total, 0 triggered, 7 resolved"},
		{name: "zero", triggered: 0, resolved: 0, want: "0 total, 0 triggered, 0 resolved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatAlertCounts(tt.triggered, tt.resolved))
		})
	}
}

func TestFormatPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		priority *pagerduty.Priority
		urgency  string
		want     string
	}{
		{
			name:     "priority set",
			priority: &pagerduty.Priority{Name: "P1"},
			urgency:  "high",
			want:     "P1",
		},
		{
			name:     "nil priority returns urgency",
			priority: nil,
			urgency:  "high",
			want:     "high",
		},
		{
			name:     "nil priority low",
			priority: nil,
			urgency:  "low",
			want:     "low",
		},
		{
			name:     "priority with empty name returns urgency",
			priority: &pagerduty.Priority{Name: ""},
			urgency:  "high",
			want:     "high",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatPriority(tt.priority, tt.urgency))
		})
	}
}

func TestRenderShowDetail(t *testing.T) {
	t.Parallel()

	t.Run("plain output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		rows := []showRow{
			{"ID", "P000001"},
			{"Status", "triggered"},
		}
		err := renderShowDetail(&buf, rows, nil)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "ID")
		assert.Contains(t, buf.String(), "P000001")
		assert.Contains(t, buf.String(), "Status")
		assert.Contains(t, buf.String(), "triggered")
	})

	t.Run("themed output bolds labels", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		th := theme.Default()
		rows := []showRow{
			{"ID", "P000001"},
		}
		err := renderShowDetail(&buf, rows, th)
		require.NoError(t, err)
		// Themed output contains ANSI escapes from bold label.
		assert.Contains(t, buf.String(), "\x1b")
		assert.Contains(t, buf.String(), "P000001")
	})

	t.Run("pre-styled value passes through", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		th := theme.Default()
		styled := "\x1b[31mtriggered\x1b[0m"
		rows := []showRow{
			{"Status", styled},
		}
		err := renderShowDetail(&buf, rows, th)
		require.NoError(t, err)
		// Pre-styled value should not be double-wrapped with dim.
		assert.Contains(t, buf.String(), "\x1b[31mtriggered\x1b[0m")
	})

	t.Run("empty rows", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := renderShowDetail(&buf, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}
