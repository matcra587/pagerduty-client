// Package integration detects and normalises alert payloads from
// monitoring tools (GCP, CloudWatch, Datadog, Prometheus, etc.) into
// a uniform Summary of fields and links for display.
package integration

import (
	"encoding/json"
	"fmt"
)

// AlertEnvelope is the unwrapped alert body, normalised from both
// raw Events API payloads and PD REST API responses (cef_details).
// UnwrapAlert handles the envelope extraction once; normalisers
// receive this instead of raw maps.
type AlertEnvelope struct {
	Client        string         // e.g. "Datadog", "Alertmanager"
	ClientURL     string         // link back to source tool
	CustomDetails map[string]any // the integration-specific payload
	Raw           map[string]any // original body for edge cases
}

// UnwrapAlert extracts the common fields from a PD alert body,
// handling the cef_details wrapping transparently. Tries:
//   - Top-level client/client_url + details.custom_details
//   - cef_details.client/client_url + cef_details.details.custom_details
//   - cef_details.details (Datadog V1 fallback)
//   - payload.custom_details (Events API v2)
//   - cef_details.payload.custom_details
func UnwrapAlert(body map[string]any) AlertEnvelope {
	if body == nil {
		return AlertEnvelope{}
	}

	env := AlertEnvelope{Raw: body}

	// Client and client_url: prefer cef_details, fall back to top-level.
	if cef, ok := body["cef_details"].(map[string]any); ok {
		if v, ok := cef["client"].(string); ok {
			env.Client = v
		}
		if v, ok := cef["client_url"].(string); ok {
			env.ClientURL = v
		}
	}
	if env.Client == "" {
		if v, ok := body["client"].(string); ok {
			env.Client = v
		}
	}
	if env.ClientURL == "" {
		if v, ok := body["client_url"].(string); ok {
			env.ClientURL = v
		}
	}

	env.CustomDetails = extractCustomDetails(body)

	return env
}

// FieldType controls how the TUI renderer styles a field.
// The zero value (FieldText) preserves current behaviour - normalisers
// opt into richer rendering by setting a non-zero type.
type FieldType int

const (
	FieldText     FieldType = iota // label: value inline (default, backward compatible)
	FieldBadge                     // coloured pill in header row
	FieldCode                      // left-bordered highlighted block
	FieldMarkdown                  // glamour-rendered, label ignored
	FieldTags                      // comma-separated -> individual background-coloured pills
)

// Field is a single key-value pair extracted from an alert payload.
// Set Type to opt into richer TUI rendering; the zero value (FieldText)
// preserves the existing inline label: value display.
type Field struct {
	Label string
	Value string
	Type  FieldType
}

// Link is a URL back to the source monitoring tool.
type Link struct {
	Label string
	URL   string
}

// Summary holds the structured data extracted from an alert payload.
type Summary struct {
	Source string  // "Google Cloud Monitoring", "Datadog", etc.
	Fields []Field // key-value pairs to display
	Links  []Link  // URLs back to source tool
}

// Normaliser detects and extracts structured data from an unwrapped
// alert envelope in a single pass. Returns the summary and true if
// the payload matches, or a zero summary and false if it does not.
type Normaliser interface {
	Normalise(env AlertEnvelope) (Summary, bool)
}

// normalisers is the ordered list of registered normalisers.
// More specific detectors first.
var normalisers = []Normaliser{
	GCP{},
	CloudWatch{},
	Prometheus{},
	Datadog{},
}

// Detect unwraps the alert body once, then returns the Summary from the
// first normaliser that matches, or the generic fallback if none match.
func Detect(body map[string]any) Summary {
	env := UnwrapAlert(body)
	for _, n := range normalisers {
		if s, ok := n.Normalise(env); ok {
			return s
		}
	}
	s, _ := Generic{}.Normalise(env)
	return s
}

// FormatValue renders a value as a string. Maps and slices become
// indented JSON wrapped in a markdown code fence; scalars use fmt.Sprintf.
func FormatValue(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return "```json\n" + string(b) + "\n```"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// walkMap traverses nested maps by key path, returning nil if any step
// is missing or not a map.
func walkMap(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		v, ok := m[k].(map[string]any)
		if !ok {
			return nil
		}
		m = v
	}
	return m
}

// extractCustomDetails walks common PagerDuty alert body structures to
// find the custom_details map.
func extractCustomDetails(body map[string]any) map[string]any {
	// details.custom_details
	if cd := walkMap(body, "details", "custom_details"); cd != nil {
		return cd
	}
	// payload.custom_details
	if cd := walkMap(body, "payload", "custom_details"); cd != nil {
		return cd
	}
	// cef_details.details.custom_details or cef_details.details
	if details := walkMap(body, "cef_details", "details"); details != nil {
		if cd := walkMap(details, "custom_details"); cd != nil {
			return cd
		}
		return details
	}
	// cef_details.payload.custom_details
	return walkMap(body, "cef_details", "payload", "custom_details")
}
