package integration

import "strings"

// Datadog normalises Datadog alert payloads (best-effort).
type Datadog struct{}

func (Datadog) Normalise(env AlertEnvelope) (Summary, bool) {
	isDatadog := strings.EqualFold(env.Client, "Datadog")
	if !isDatadog && env.CustomDetails != nil {
		_, hasQuery := env.CustomDetails["query"]
		_, hasState := env.CustomDetails["monitor_state"]
		isDatadog = hasQuery && hasState
	}
	if !isDatadog {
		return Summary{}, false
	}

	cd := env.CustomDetails
	s := Summary{Source: "Datadog"}
	if cd == nil {
		return s, true
	}

	if v, ok := cd["title"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Title", Value: v})
	}
	if v, ok := cd["query"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Query", Value: v})
	}
	if v, ok := cd["monitor_state"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "State", Value: v})
	}
	if v, ok := cd["event_type"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Type", Value: v})
	}
	if v, ok := cd["priority"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Priority", Value: v})
	}
	if v, ok := cd["tags"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Tags", Value: v})
	}
	if v, ok := cd["org"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Org", Value: v})
	}
	if v, ok := cd["body"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Body", Value: v})
	}
	if env.ClientURL != "" {
		s.Links = append(s.Links, Link{Label: "Datadog", URL: env.ClientURL})
	}
	return s, true
}
