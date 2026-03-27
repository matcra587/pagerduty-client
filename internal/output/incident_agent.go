package output

import (
	"github.com/PagerDuty/go-pagerduty"
)

// AgentIncident is the agent-mode projection of a PagerDuty incident.
type AgentIncident struct {
	ID                 string           `json:"id"`
	IncidentNumber     uint             `json:"incident_number"`
	Title              string           `json:"title"`
	Status             string           `json:"status"`
	Urgency            string           `json:"urgency"`
	HTMLURL            string           `json:"html_url"`
	CreatedAt          string           `json:"created_at"`
	UpdatedAt          string           `json:"updated_at"`
	LastStatusChangeAt string           `json:"last_status_change_at"`
	Priority           *string          `json:"priority,omitempty"`
	Service            AgentRef         `json:"service"`
	EscalationPolicy   AgentRef         `json:"escalation_policy"`
	Assignees          []AgentRef       `json:"assignees"`
	AlertCounts        AgentAlertCounts `json:"alert_counts"`
	IsMergeable        bool             `json:"is_mergeable"`
}

// AgentAlertCounts is a compact representation of incident alert counts.
type AgentAlertCounts struct {
	Triggered uint `json:"triggered"`
	Resolved  uint `json:"resolved"`
	All       uint `json:"all"`
}

// ProjectIncidentForAgent projects a PagerDuty incident into the compact
// agent-mode representation, stripping duplicates and reference noise.
func ProjectIncidentForAgent(i pagerduty.Incident) AgentIncident {
	ai := AgentIncident{
		ID:                 i.ID,
		IncidentNumber:     i.IncidentNumber,
		Title:              i.Title,
		Status:             i.Status,
		Urgency:            i.Urgency,
		HTMLURL:            i.HTMLURL,
		CreatedAt:          i.CreatedAt,
		UpdatedAt:          i.UpdatedAt,
		LastStatusChangeAt: i.LastStatusChangeAt,
		Service:            AgentRef{ID: i.Service.ID, Summary: i.Service.Summary},
		EscalationPolicy:   AgentRef{ID: i.EscalationPolicy.ID, Summary: i.EscalationPolicy.Summary},
		AlertCounts: AgentAlertCounts{
			Triggered: i.AlertCounts.Triggered,
			Resolved:  i.AlertCounts.Resolved,
			All:       i.AlertCounts.All,
		},
		IsMergeable: i.IsMergeable,
		Assignees:   []AgentRef{},
	}

	if i.Priority != nil {
		name := i.Priority.Name
		ai.Priority = &name
	}

	if len(i.Assignments) > 0 {
		ai.Assignees = make([]AgentRef, len(i.Assignments))
		for idx, a := range i.Assignments {
			ai.Assignees[idx] = AgentRef{
				ID:      a.Assignee.ID,
				Summary: a.Assignee.Summary,
			}
		}
	}

	return ai
}

// ProjectIncidentsForAgent projects a slice of PagerDuty incidents.
func ProjectIncidentsForAgent(incidents []pagerduty.Incident) []AgentIncident {
	out := make([]AgentIncident, len(incidents))
	for idx, i := range incidents {
		out[idx] = ProjectIncidentForAgent(i)
	}
	return out
}
