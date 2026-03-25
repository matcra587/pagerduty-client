package integration

import "strings"

// Prometheus normalises Prometheus Alertmanager payloads.
// Ref: https://www.pagerduty.com/docs/guides/prometheus-integration-guide/
type Prometheus struct{}

func (Prometheus) Normalise(env AlertEnvelope) (Summary, bool) {
	detected := strings.Contains(env.Client, "Alertmanager")

	if !detected {
		if env.CustomDetails == nil {
			return Summary{}, false
		}
		_, hasNumFiring := env.CustomDetails["num_firing"]
		_, hasFiring := env.CustomDetails["firing"]
		if !hasNumFiring || !hasFiring {
			return Summary{}, false
		}
	}

	cd := env.CustomDetails
	s := Summary{Source: "Prometheus Alertmanager"}
	if cd == nil {
		return s, true
	}

	if v, ok := cd["alertname"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Alert", Value: v})
	}
	if v, ok := cd["severity"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Severity", Value: v})
	}
	if v, ok := cd["description"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Description", Value: v})
	}
	if v, ok := cd["cluster"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Cluster", Value: v})
	}
	if v, ok := cd["namespace"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Namespace", Value: v})
	}
	if v, ok := cd["num_firing"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Firing", Value: v})
	}
	if v, ok := cd["num_resolved"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Resolved", Value: v})
	}
	if v, ok := cd["firing"].(string); ok && v != "" {
		s.Fields = append(s.Fields, Field{Label: "Firing alerts", Value: v})
	}

	if v, ok := cd["runbook"].(string); ok && v != "" {
		s.Links = append(s.Links, Link{Label: "Runbook", URL: v})
	}
	if env.ClientURL != "" {
		s.Links = append(s.Links, Link{Label: "Alertmanager", URL: env.ClientURL})
	}
	return s, true
}
