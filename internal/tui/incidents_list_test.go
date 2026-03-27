package tui

import (
	"context"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAckCmdNilClient(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.incidents = []pagerduty.Incident{{APIObject: pagerduty.APIObject{ID: "P1"}}}
	cmd := m.ackCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(statusMsg)
	assert.True(t, ok, "expected statusMsg for nil client, got %T", msg)
}

func TestResolveCmdNilClient(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.incidents = []pagerduty.Incident{{APIObject: pagerduty.APIObject{ID: "P1"}}}
	cmd := m.resolveCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(statusMsg)
	assert.True(t, ok, "expected statusMsg for nil client, got %T", msg)
}

func TestBatchAckCmdNilClient(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.incidents = []pagerduty.Incident{{APIObject: pagerduty.APIObject{ID: "P1"}}}
	m.selections = map[string]bool{"P1": true}
	cmd := m.batchAckCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(statusMsg)
	assert.True(t, ok, "expected statusMsg for nil client, got %T", msg)
}

func TestBatchResolveCmdNilClient(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.incidents = []pagerduty.Incident{{APIObject: pagerduty.APIObject{ID: "P1"}}}
	m.selections = map[string]bool{"P1": true}
	cmd := m.batchResolveCmd()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(statusMsg)
	assert.True(t, ok, "expected statusMsg for nil client, got %T", msg)
}

func TestMatchesStructuredFilter_AgeNotFilteredClientSide(t *testing.T) {
	oldIncident := pagerduty.Incident{
		CreatedAt: time.Now().Add(-14 * 24 * time.Hour).UTC().Format(time.RFC3339),
	}

	fs := components.FilterState{
		Priority: "all",
		Assigned: "all",
		Age:      "7d",
	}
	assert.True(t, matchesStructuredFilter(oldIncident, fs),
		"old incident should pass client-side filter; age is API-level")
}

func TestSearchIndex_BuiltOnSetIncidents(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.SetIncidents([]pagerduty.Incident{
		{
			APIObject: pagerduty.APIObject{ID: "PABC123"},
			Title:     "Database CPU High",
			Service:   pagerduty.APIObject{Summary: "Web API"},
			Assignments: []pagerduty.Assignment{
				{Assignee: pagerduty.APIObject{Summary: "Alice Smith"}},
			},
		},
	})

	require.Len(t, m.searchIndex, 1)
	assert.Contains(t, m.searchIndex[0], "pabc123")
	assert.Contains(t, m.searchIndex[0], "database cpu high")
	assert.Contains(t, m.searchIndex[0], "web api")
	assert.Contains(t, m.searchIndex[0], "alice smith")
}

func TestSearchIndex_SeparatorPreventsCrossFieldMatch(t *testing.T) {
	m := newIncidentList(context.Background(), nil, "test@example.com", false)
	m.SetIncidents([]pagerduty.Incident{
		{
			APIObject: pagerduty.APIObject{ID: "abc"},
			Title:     "def",
		},
	})

	m.filterInput.SetValue("cde")
	vis := m.visibleIncidents()
	assert.Empty(t, vis, "cross-field query should not match")
}

func TestIsDefaultFilter(t *testing.T) {
	tests := []struct {
		name string
		fs   components.FilterState
		want bool
	}{
		{
			name: "all defaults",
			fs:   components.FilterState{Priority: "all", Assigned: "all", Age: "7d"},
			want: true,
		},
		{
			name: "empty values treated as default",
			fs:   components.FilterState{},
			want: true,
		},
		{
			name: "priority set",
			fs:   components.FilterState{Priority: "P1", Assigned: "all"},
			want: false,
		},
		{
			name: "assigned set",
			fs:   components.FilterState{Priority: "all", Assigned: "assigned"},
			want: false,
		},
		{
			name: "age does not affect client-side default check",
			fs:   components.FilterState{Priority: "all", Assigned: "all", Age: "30d"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDefaultFilter(tt.fs))
		})
	}
}
