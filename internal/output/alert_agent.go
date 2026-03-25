package output

import (
	"github.com/PagerDuty/go-pagerduty"
)

// AgentAlert is the agent-mode projection of a PagerDuty alert.
type AgentAlert struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	Severity   string          `json:"severity"`
	Summary    string          `json:"summary"`
	CreatedAt  string          `json:"created_at"`
	AlertKey   string          `json:"alert_key"`
	HTMLURL    string          `json:"html_url"`
	Suppressed bool            `json:"suppressed"`
	Service    AgentRef        `json:"service"`
	IncidentID string          `json:"incident_id"`
	Body       *AgentAlertBody `json:"body,omitempty"`
}

// AgentAlertBody holds the deduplicated alert body content.
// For CEF-format alerts, fields come from cef_details (canonical source).
// For non-CEF alerts, fields fall back to top-level body keys.
type AgentAlertBody struct {
	Client    string           `json:"client,omitempty"`
	ClientURL string           `json:"client_url,omitempty"`
	Details   map[string]any   `json:"details,omitempty"`
	Contexts  []map[string]any `json:"contexts,omitempty"`
}

// ProjectAlertForAgent projects a PagerDuty alert into the compact
// agent-mode representation.
func ProjectAlertForAgent(a pagerduty.IncidentAlert) AgentAlert {
	return AgentAlert{
		ID:         a.ID,
		Status:     a.Status,
		Severity:   a.Severity,
		Summary:    a.Summary,
		CreatedAt:  a.CreatedAt,
		AlertKey:   a.AlertKey,
		HTMLURL:    a.HTMLURL,
		Suppressed: a.Suppressed,
		Service:    AgentRef{ID: a.Service.ID, Summary: a.Service.Summary},
		IncidentID: a.Incident.ID,
		Body:       extractAlertBody(a.Body),
	}
}

// ProjectAlertsForAgent projects a slice of PagerDuty alerts.
func ProjectAlertsForAgent(alerts []pagerduty.IncidentAlert) []AgentAlert {
	out := make([]AgentAlert, len(alerts))
	for idx, a := range alerts {
		out[idx] = ProjectAlertForAgent(a)
	}
	return out
}

func extractAlertBody(raw map[string]any) *AgentAlertBody {
	if len(raw) == 0 {
		return nil
	}

	ab := &AgentAlertBody{}

	cef, _ := raw["cef_details"].(map[string]any)
	if cef != nil {
		ab.Client, _ = cef["client"].(string)
		ab.ClientURL, _ = cef["client_url"].(string)
		ab.Details, _ = cef["details"].(map[string]any)
		ab.Contexts = toMapSlice(cef["contexts"])
	} else {
		ab.Client, _ = raw["client"].(string)
		ab.ClientURL, _ = raw["client_url"].(string)
		ab.Details, _ = raw["details"].(map[string]any)
		ab.Contexts = toMapSlice(raw["contexts"])
	}

	if ab.Client == "" && ab.ClientURL == "" && ab.Details == nil && len(ab.Contexts) == 0 {
		return nil
	}
	return ab
}

func toMapSlice(v any) []map[string]any {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
