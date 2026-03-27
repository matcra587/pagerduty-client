package output

import (
	"encoding/json"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullIncident() pagerduty.Incident {
	return pagerduty.Incident{
		APIObject: pagerduty.APIObject{
			ID:      "PABC123",
			Type:    "incident",
			Summary: "[#42] CPU High on web-1",
			Self:    "https://api.pagerduty.com/incidents/PABC123",
			HTMLURL: "https://app.pagerduty.com/incidents/PABC123",
		},
		IncidentNumber:     42,
		Title:              "CPU High on web-1",
		Description:        "CPU High on web-1",
		CreatedAt:          "2026-03-19T10:00:00Z",
		UpdatedAt:          "2026-03-19T10:05:00Z",
		LastStatusChangeAt: "2026-03-19T10:05:00Z",
		Status:             "triggered",
		Urgency:            "high",
		IncidentKey:        "cpu-high-web-1",
		Service: pagerduty.APIObject{
			ID:      "PSVC001",
			Type:    "service_reference",
			Summary: "Web Tier",
			Self:    "https://api.pagerduty.com/services/PSVC001",
			HTMLURL: "https://app.pagerduty.com/service-directory/PSVC001",
		},
		EscalationPolicy: pagerduty.APIObject{
			ID:      "PESC001",
			Type:    "escalation_policy_reference",
			Summary: "Web Escalation",
			Self:    "https://api.pagerduty.com/escalation_policies/PESC001",
			HTMLURL: "https://app.pagerduty.com/escalation_policies/PESC001",
		},
		Assignments: []pagerduty.Assignment{
			{
				At: "2026-03-19T10:00:00Z",
				Assignee: pagerduty.APIObject{
					ID:      "PUSR001",
					Type:    "user_reference",
					Summary: "Alice Smith",
					Self:    "https://api.pagerduty.com/users/PUSR001",
					HTMLURL: "https://app.pagerduty.com/users/PUSR001",
				},
			},
			{
				At: "2026-03-19T10:01:00Z",
				Assignee: pagerduty.APIObject{
					ID:      "PUSR002",
					Type:    "user_reference",
					Summary: "Bob Jones",
					Self:    "https://api.pagerduty.com/users/PUSR002",
					HTMLURL: "https://app.pagerduty.com/users/PUSR002",
				},
			},
		},
		LastStatusChangeBy: pagerduty.APIObject{
			ID:      "PSVC001",
			Type:    "service_reference",
			Summary: "Web Tier",
		},
		Priority: &pagerduty.Priority{
			APIObject: pagerduty.APIObject{
				ID:   "PPRI001",
				Type: "priority",
			},
			Name: "P1",
		},
		AlertCounts: pagerduty.AlertCounts{
			Triggered: 2,
			Resolved:  1,
			All:       3,
		},
		IsMergeable: true,
		AssignedVia: "escalation_policy",
	}
}

func TestProjectIncidentForAgent_FullIncident(t *testing.T) {
	t.Parallel()
	inc := fullIncident()
	got := ProjectIncidentForAgent(inc)

	assert.Equal(t, "PABC123", got.ID)
	assert.Equal(t, uint(42), got.IncidentNumber)
	assert.Equal(t, "CPU High on web-1", got.Title)
	assert.Equal(t, "triggered", got.Status)
	assert.Equal(t, "high", got.Urgency)
	assert.Equal(t, "https://app.pagerduty.com/incidents/PABC123", got.HTMLURL)
	assert.Equal(t, "2026-03-19T10:00:00Z", got.CreatedAt)
	assert.Equal(t, "2026-03-19T10:05:00Z", got.UpdatedAt)
	assert.Equal(t, "2026-03-19T10:05:00Z", got.LastStatusChangeAt)

	require.NotNil(t, got.Priority)
	assert.Equal(t, "P1", *got.Priority)

	assert.Equal(t, AgentRef{ID: "PSVC001", Summary: "Web Tier"}, got.Service)
	assert.Equal(t, AgentRef{ID: "PESC001", Summary: "Web Escalation"}, got.EscalationPolicy)

	require.Len(t, got.Assignees, 2)
	assert.Equal(t, AgentRef{ID: "PUSR001", Summary: "Alice Smith"}, got.Assignees[0])
	assert.Equal(t, AgentRef{ID: "PUSR002", Summary: "Bob Jones"}, got.Assignees[1])

	assert.Equal(t, AgentAlertCounts{Triggered: 2, Resolved: 1, All: 3}, got.AlertCounts)
	assert.True(t, got.IsMergeable)
}

func TestProjectIncidentForAgent_NilPriority(t *testing.T) {
	t.Parallel()
	inc := fullIncident()
	inc.Priority = nil

	got := ProjectIncidentForAgent(inc)
	assert.Nil(t, got.Priority)
}

func TestProjectIncidentForAgent_EmptyAssignments(t *testing.T) {
	t.Parallel()
	inc := fullIncident()
	inc.Assignments = nil

	got := ProjectIncidentForAgent(inc)
	assert.Empty(t, got.Assignees)
}

func TestProjectIncidentsForAgent_Slice(t *testing.T) {
	t.Parallel()
	incidents := []pagerduty.Incident{fullIncident(), fullIncident()}
	incidents[1].APIObject.ID = "PXYZ999"

	got := ProjectIncidentsForAgent(incidents)
	require.Len(t, got, 2)
	assert.Equal(t, "PABC123", got[0].ID)
	assert.Equal(t, "PXYZ999", got[1].ID)
}

func TestProjectIncidentForAgent_DroppedFields(t *testing.T) {
	t.Parallel()
	inc := fullIncident()
	got := ProjectIncidentForAgent(inc)

	b, err := json.Marshal(got)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	for _, key := range []string{
		"summary", "description", "self", "type",
		"incident_key", "first_trigger_log_entry",
		"last_status_change_by", "resolve_reason",
		"assigned_via", "body",
	} {
		assert.NotContains(t, m, key)
	}
}
