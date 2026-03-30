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

	// Badges.
	if v, ok := cd["monitor_state"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "State", Value: v, Type: FieldBadge})
	}
	if v, ok := cd["priority"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Priority", Value: v, Type: FieldBadge})
	}
	// Text.
	if v, ok := cd["title"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Title", Value: v})
	}
	// Code.
	if v, ok := cd["query"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Query", Value: v, Type: FieldCode})
	}
	// Markdown.
	if v, ok := cd["body"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Body", Value: v, Type: FieldMarkdown})
	}
	// Tags: handle both string and []any formats.
	switch tags := cd["tags"].(type) {
	case string:
		s.Fields = append(s.Fields, Field{Label: "Tags", Value: tags, Type: FieldTags})
	case []any:
		var parts []string
		for _, t := range tags {
			if str, ok := t.(string); ok {
				parts = append(parts, str)
			}
		}
		if len(parts) > 0 {
			s.Fields = append(s.Fields, Field{Label: "Tags", Value: strings.Join(parts, ", "), Type: FieldTags})
		}
	}
	if env.ClientURL != "" {
		s.Links = append(s.Links, Link{Label: "Datadog", URL: env.ClientURL})
	}
	return s, true
}
