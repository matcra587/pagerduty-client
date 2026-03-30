package integration

import "fmt"

// GCP normalises Google Cloud Monitoring alert payloads (schema v1.2).
// Ref: https://docs.cloud.google.com/monitoring/support/notification-options
type GCP struct{}

func (GCP) Normalise(env AlertEnvelope) (Summary, bool) {
	if env.CustomDetails == nil {
		return Summary{}, false
	}
	inc, ok := env.CustomDetails["incident"].(map[string]any)
	if !ok {
		return Summary{}, false
	}
	if _, has := inc["policy_name"]; !has {
		return Summary{}, false
	}
	if _, has := inc["condition"]; !has {
		return Summary{}, false
	}

	s := Summary{Source: "Google Cloud Monitoring"}

	if v, ok := inc["policy_name"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Policy", Value: v})
	}
	if v, ok := inc["condition_name"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Condition", Value: v})
	}
	if res, ok := inc["resource"].(map[string]any); ok {
		resDisplay, _ := inc["resource_type_display_name"].(string)
		if labels, ok := res["labels"].(map[string]any); ok {
			if host, ok := labels["host"].(string); ok && host != "" {
				if resDisplay != "" {
					resDisplay += " (" + host + ")"
				} else {
					resDisplay = host
				}
			}
		}
		if resDisplay != "" {
			s.Fields = append(s.Fields, Field{Label: "Resource", Value: resDisplay})
		}
	}
	if m, ok := inc["metric"].(map[string]any); ok {
		if name, ok := m["displayName"].(string); ok {
			s.Fields = append(s.Fields, Field{Label: "Metric", Value: name})
		}
	}
	if obs, ok := inc["observed_value"].(string); ok {
		val := obs
		if thr, ok := inc["threshold_value"].(string); ok {
			val = fmt.Sprintf("%s (threshold: %s)", obs, thr)
		}
		s.Fields = append(s.Fields, Field{Label: "Observed", Value: val})
	}
	if v, ok := inc["severity"].(string); ok && v != "No severity" {
		s.Fields = append(s.Fields, Field{Label: "Severity", Value: v})
	}
	if v, ok := inc["state"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "State", Value: v, Type: FieldBadge})
	}
	if v, ok := inc["summary"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Summary", Value: v})
	}
	if u, ok := inc["url"].(string); ok {
		s.Links = append(s.Links, Link{Label: "GCP Console", URL: u})
	}
	if doc, ok := inc["documentation"].(map[string]any); ok {
		if links, ok := doc["links"].([]any); ok {
			for _, l := range links {
				if lm, ok := l.(map[string]any); ok {
					name, _ := lm["displayName"].(string)
					url, _ := lm["url"].(string)
					if url != "" {
						if name == "" {
							name = "Documentation"
						}
						s.Links = append(s.Links, Link{Label: name, URL: url})
					}
				}
			}
		}
	}

	return s, true
}
