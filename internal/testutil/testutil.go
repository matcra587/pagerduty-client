// Package testutil provides shared golden JSON fixtures and typed loaders
// for PagerDuty test data across all test consumers.
package testutil

import (
	"embed"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
)

//go:embed testdata/*
var fixtures embed.FS

// LoadRaw returns the raw bytes of a fixture file from testdata/.
func LoadRaw(t *testing.T, filename string) []byte {
	t.Helper()

	data, err := fixtures.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatalf("testutil: load %s: %v", filename, err)
	}

	return data
}

// LoadIncidents returns all fixture incidents.
func LoadIncidents(t *testing.T) []pagerduty.Incident {
	t.Helper()

	var incidents []pagerduty.Incident
	unmarshalFixture(t, "incidents.json", &incidents)

	return incidents
}

// LoadAlerts returns fixture alerts for the given incident ID.
// Returns nil if no alerts exist for that incident.
func LoadAlerts(t *testing.T, incidentID string) []pagerduty.IncidentAlert {
	t.Helper()

	var m map[string][]pagerduty.IncidentAlert
	unmarshalFixture(t, "alerts.json", &m)

	return m[incidentID]
}

// LoadNotes returns fixture notes for the given incident ID.
// Returns nil if no notes exist for that incident.
func LoadNotes(t *testing.T, incidentID string) []pagerduty.IncidentNote {
	t.Helper()

	var m map[string][]pagerduty.IncidentNote
	unmarshalFixture(t, "notes.json", &m)

	return m[incidentID]
}

// MustLoadIncidents is like LoadIncidents but panics on error.
// For use in TUI test mode outside of test context.
func MustLoadIncidents() []pagerduty.Incident {
	var incidents []pagerduty.Incident
	mustUnmarshalFixture("incidents.json", &incidents)

	return incidents
}

// MustLoadAlerts is like LoadAlerts but panics on error.
func MustLoadAlerts(incidentID string) []pagerduty.IncidentAlert {
	var m map[string][]pagerduty.IncidentAlert
	mustUnmarshalFixture("alerts.json", &m)

	return m[incidentID]
}

// MustLoadNotes is like LoadNotes but panics on error.
func MustLoadNotes(incidentID string) []pagerduty.IncidentNote {
	var m map[string][]pagerduty.IncidentNote
	mustUnmarshalFixture("notes.json", &m)

	return m[incidentID]
}

// WrapList wraps a fixture array in a PD pagination envelope for use in
// httptest handlers. Returns JSON bytes: {"<key>": [...], "limit": 100,
// "offset": 0, "more": false, "total": N}.
func WrapList(t *testing.T, key, filename string) []byte {
	t.Helper()

	raw := LoadRaw(t, filename)

	var arr []json.RawMessage
	unmarshalBytes(t, raw, &arr)

	return marshalEnvelope(t, key, raw, len(arr))
}

// WrapAlertsForIncident extracts alerts for a single incident from
// alerts.json and wraps them in a PD API response envelope.
func WrapAlertsForIncident(t *testing.T, incidentID string) []byte {
	t.Helper()

	var m map[string]json.RawMessage
	unmarshalFixture(t, "alerts.json", &m)

	raw := m[incidentID]

	var arr []json.RawMessage
	unmarshalBytes(t, raw, &arr)

	return marshalEnvelope(t, "alerts", raw, len(arr))
}

// WrapNotesForIncident extracts notes for a single incident from
// notes.json and wraps them as {"notes": [...]}. The PD notes endpoint
// does not paginate so no pagination envelope is used.
func WrapNotesForIncident(t *testing.T, incidentID string) []byte {
	t.Helper()

	var m map[string]json.RawMessage
	unmarshalFixture(t, "notes.json", &m)

	result, err := json.Marshal(map[string]json.RawMessage{"notes": m[incidentID]})
	if err != nil {
		t.Fatalf("testutil: wrap notes: %v", err)
	}

	return result
}

// WrapSingleIncident extracts a single incident by index from
// incidents.json and wraps it as {"incident": {...}}.
func WrapSingleIncident(t *testing.T, index int) []byte {
	t.Helper()

	var arr []json.RawMessage
	unmarshalFixture(t, "incidents.json", &arr)

	if index >= len(arr) {
		t.Fatalf("testutil: incident index %d out of range (have %d)", index, len(arr))
	}

	result, err := json.Marshal(map[string]json.RawMessage{"incident": arr[index]})
	if err != nil {
		t.Fatalf("testutil: marshal single incident: %v", err)
	}

	return result
}

func unmarshalBytes(t *testing.T, data []byte, v any) {
	t.Helper()

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("testutil: unmarshal bytes: %v", err)
	}
}

func marshalEnvelope(t *testing.T, key string, data json.RawMessage, total int) []byte {
	t.Helper()

	env := map[string]any{
		key:      data,
		"limit":  100,
		"offset": 0,
		"more":   false,
		"total":  total,
	}

	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("testutil: marshal envelope: %v", err)
	}

	return b
}

func unmarshalFixture(t *testing.T, filename string, v any) {
	t.Helper()

	data := LoadRaw(t, filename)

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("testutil: unmarshal %s: %v", filename, err)
	}
}

func mustUnmarshalFixture(filename string, v any) {
	data, err := fixtures.ReadFile("testdata/" + filename)
	if err != nil {
		panic(fmt.Sprintf("testutil: load %s: %v", filename, err))
	}

	if err := json.Unmarshal(data, v); err != nil {
		panic(fmt.Sprintf("testutil: unmarshal %s: %v", filename, err))
	}
}
