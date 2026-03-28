package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompact_StripsDeniedFields(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":      "P123",
		"summary": "Test incident",
		"self":    "https://api.pagerduty.com/incidents/P123",
		"type":    "incident",
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "Test incident", m["summary"])
	assert.NotContains(t, m, "self")
	assert.NotContains(t, m, "type")
}

func TestCompact_StripsDeniedFieldsAtDepth(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id": "P123",
		"service": map[string]any{
			"id":      "PSVC",
			"summary": "My Service",
			"self":    "https://api.pagerduty.com/services/PSVC",
			"type":    "service",
			"name":    "My Service",
		},
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	svc, ok := m["service"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "PSVC", svc["id"])
	assert.Equal(t, "My Service", svc["summary"])
	assert.Equal(t, "My Service", svc["name"])
	assert.NotContains(t, svc, "self")
	assert.NotContains(t, svc, "type")
}

func TestCompact_FlattensNestedAPIObject(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id": "P123",
		"service": map[string]any{
			"id":       "PSVC",
			"type":     "service_reference",
			"summary":  "My Service",
			"self":     "https://api.pagerduty.com/services/PSVC",
			"html_url": "https://example.pagerduty.com/services/PSVC",
		},
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	svc, ok := m["service"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "PSVC", svc["id"])
	assert.Equal(t, "My Service", svc["summary"])
	assert.NotContains(t, svc, "html_url")
	assert.NotContains(t, svc, "self")
	assert.NotContains(t, svc, "type")
}

func TestCompact_KeepsHTMLURLOnTopLevel(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":       "P123",
		"summary":  "Test incident",
		"html_url": "https://example.pagerduty.com/incidents/P123",
		"self":     "https://api.pagerduty.com/incidents/P123",
		"type":     "incident",
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "Test incident", m["summary"])
	assert.Equal(t, "https://example.pagerduty.com/incidents/P123", m["html_url"])
}

func TestCompact_DoesNotFlattenNonAPIObject(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id": "P123",
		"service": map[string]any{
			"id":       "PSVC",
			"summary":  "My Service",
			"html_url": "https://example.pagerduty.com/services/PSVC",
			"name":     "My Service",
		},
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	svc, ok := m["service"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "PSVC", svc["id"])
	assert.Equal(t, "My Service", svc["summary"])
	assert.Equal(t, "https://example.pagerduty.com/services/PSVC", svc["html_url"])
	assert.Equal(t, "My Service", svc["name"])
}

func TestCompact_StripsNullsAndEmpties(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":          "P123",
		"summary":     "Test",
		"description": nil,
		"notes":       "",
		"tags":        []any{},
		"metadata":    map[string]any{},
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "Test", m["summary"])
	assert.NotContains(t, m, "description")
	assert.NotContains(t, m, "notes")
	assert.NotContains(t, m, "tags")
	assert.NotContains(t, m, "metadata")
}

func TestCompact_PreservesFalseAndZero(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":      "P123",
		"active":  false,
		"count":   float64(0),
		"summary": "Test",
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, false, m["active"])
	assert.InDelta(t, float64(0), m["count"], 0)
}

func TestCompact_CEFBodyDedup(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":          "P123",
		"summary":     "Test",
		"client":      "monitoring",
		"client_url":  "https://monitoring.example.com",
		"details":     map[string]any{"foo": "bar"},
		"contexts":    []any{"ctx1"},
		"cef_details": map[string]any{"foo": "bar"},
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Contains(t, m, "cef_details")
	assert.NotContains(t, m, "client")
	assert.NotContains(t, m, "client_url")
	assert.NotContains(t, m, "details")
	assert.NotContains(t, m, "contexts")
}

func TestCompact_ArrayOfObjects(t *testing.T) {
	t.Parallel()

	input := []any{
		map[string]any{
			"id":      "P1",
			"summary": "First",
			"self":    "https://api.pagerduty.com/incidents/P1",
			"type":    "incident",
			"empty":   "",
		},
		map[string]any{
			"id":      "P2",
			"summary": "Second",
			"self":    "https://api.pagerduty.com/incidents/P2",
			"type":    "incident",
		},
	}

	got := Compact(input)

	arr, ok := got.([]any)
	require.True(t, ok)
	require.Len(t, arr, 2)

	first, ok := arr[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "P1", first["id"])
	assert.NotContains(t, first, "self")
	assert.NotContains(t, first, "type")
	assert.NotContains(t, first, "empty")

	second, ok := arr[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "P2", second["id"])
}

func TestCompact_PublicFunction(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
		Self    string `json:"self"`
		Type    string `json:"type"`
	}

	input := testStruct{
		ID:      "P123",
		Summary: "Test",
		Self:    "https://api.pagerduty.com/incidents/P123",
		Type:    "incident",
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "Test", m["summary"])
	assert.NotContains(t, m, "self")
	assert.NotContains(t, m, "type")
}

func TestCompact_AlreadyCompactMap(t *testing.T) {
	t.Parallel()

	input := map[string]string{
		"id":      "P123",
		"summary": "Test",
	}

	got := Compact(input)

	m, ok := got.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "Test", m["summary"])
}
