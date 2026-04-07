// Package compact normalises PagerDuty API data for agent JSON output,
// applying deny-list filtering, empty-value stripping, APIObject
// flattening and token-budget field selection.
package compact

import (
	"encoding/json"

	"github.com/gechr/clog"
)

// deniedFields lists PagerDuty API fields stripped during compaction.
// These are noisy or irrelevant for agent consumption.
var deniedFields = map[string]struct{}{
	// APIObject noise - present on every nested reference.
	"self": {},
	"type": {},

	// Incident noise.
	"assigned_via":            {},
	"first_trigger_log_entry": {},
	"last_status_change_by":   {},
	"incident_responders":     {},
	"responder_requests":      {},
	"pending_actions":         {},
	"is_mergeable":            {},
	"occurrence":              {},
	"conference_bridge":       {},
	"resolve_reason":          {},

	// Service noise.
	"alert_grouping":                      {},
	"alert_grouping_timeout":              {},
	"alert_grouping_parameters":           {},
	"alert_creation":                      {},
	"auto_pause_notifications_parameters": {},
	"scheduled_actions":                   {},
	"addons":                              {},
	"response_play":                       {},

	// User noise.
	"avatar_url":         {},
	"color":              {},
	"contact_methods":    {},
	"invitation_sent":    {},
	"notification_rules": {},

	// Schedule noise.
	"final_schedule": {},
}

// apiObjectKeys are the only keys present in a PagerDuty APIObject reference.
var apiObjectKeys = map[string]struct{}{
	"id":       {},
	"summary":  {},
	"html_url": {},
}

// cefDuplicateKeys are top-level keys superseded by cef_details.
var cefDuplicateKeys = map[string]struct{}{
	"client":     {},
	"client_url": {},
	"details":    {},
	"contexts":   {},
}

// Compact normalises data for agent JSON output. It marshals data to
// JSON, unmarshals to untyped Go values, then applies deny-list
// filtering, empty-value stripping and APIObject flattening. If the
// marshal round-trip fails, the original data is returned unchanged.
func Compact(data any) any {
	b, err := json.Marshal(data)
	if err != nil {
		clog.Debug().Err(err).Msg("compact: marshal failed, returning original")
		return data
	}

	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		clog.Debug().Err(err).Msg("compact: unmarshal failed, returning original")
		return data
	}

	return compactValue(raw, 0)
}

// compactValue dispatches to the appropriate handler based on type.
func compactValue(v any, depth int) any {
	switch val := v.(type) {
	case map[string]any:
		return compactObject(val, depth)
	case []any:
		return compactArray(val, depth)
	default:
		return v
	}
}

// compactObject applies CEF dedup, deny-list filtering, empty stripping
// and APIObject flattening to a map.
func compactObject(m map[string]any, depth int) any {
	hasCEF := m["cef_details"] != nil

	out := make(map[string]any, len(m))

	for k, v := range m {
		if _, denied := deniedFields[k]; denied {
			continue
		}

		// CEF dedup: skip top-level duplicates when cef_details exists.
		if hasCEF {
			if _, dup := cefDuplicateKeys[k]; dup {
				continue
			}
		}

		v = compactValue(v, depth+1)

		if isEmpty(v) {
			continue
		}

		out[k] = v
	}

	// Flatten nested APIObject references to {id, summary}.
	if depth > 0 && isAPIObjectRef(out) {
		flat := map[string]any{"id": out["id"]}
		if s, ok := out["summary"]; ok {
			flat["summary"] = s
		}
		return flat
	}

	return out
}

// compactArray compacts each element and drops empties.
func compactArray(arr []any, depth int) any {
	var out []any

	for _, v := range arr {
		v = compactValue(v, depth+1)
		if isEmpty(v) {
			continue
		}
		out = append(out, v)
	}

	if len(out) == 0 {
		return []any{}
	}

	return out
}

// isAPIObjectRef returns true if m contains only APIObject keys
// (id, summary, html_url) and has an id field.
func isAPIObjectRef(m map[string]any) bool {
	if _, ok := m["id"]; !ok {
		return false
	}

	for k := range m {
		if _, ok := apiObjectKeys[k]; !ok {
			return false
		}
	}

	return true
}

// isEmpty returns true for nil, empty string, empty slice or empty map.
// It does not consider false or zero to be empty.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		return false
	}
}
