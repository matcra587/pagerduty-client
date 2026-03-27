package output

import (
	"encoding/json"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cefAlertBody() map[string]any {
	details := map[string]any{
		"body":       "Alert fired for high CPU",
		"event_id":   "12345",
		"event_type": "metric_alert_monitor",
		"query":      "avg(last_5m):avg:system.cpu{host:web-1} > 90",
		"tags":       "env:prod, host:web-1",
		"title":      "[TRIGGERED] CPU High on web-1",
	}
	contexts := []any{
		map[string]any{
			"href": "https://app.datadoghq.com/monitors/123",
			"text": "Monitor Link",
			"type": "link",
		},
	}
	return map[string]any{
		"type": "alert_body",
		"cef_details": map[string]any{
			"client":           "Datadog",
			"client_url":       "https://app.datadoghq.com/monitors/123",
			"details":          details,
			"contexts":         contexts,
			"description":      "[TRIGGERED] CPU High on web-1",
			"message":          "[TRIGGERED] CPU High on web-1",
			"source_component": "avg(last_5m):avg:system.cpu{host:web-1} > 90",
			"service_group":    "env:prod, host:web-1",
			"event_class":      "metric_alert_monitor",
			"message_id":       "12345",
			"severity":         "warning",
			"priority":         "normal",
			"dedup_key":        "abc-123",
			"creation_time":    "2026-03-19T10:00:00.000000Z",
			"source_location":  nil,
			"source_origin":    nil,
		},
		"details":  details,
		"contexts": contexts,
	}
}

func fullAlert() pagerduty.IncidentAlert {
	return pagerduty.IncidentAlert{
		APIObject: pagerduty.APIObject{
			ID:      "PALRT001",
			Type:    "alert",
			Summary: "[TRIGGERED] CPU High on web-1",
			Self:    "https://api.pagerduty.com/alerts/PALRT001",
			HTMLURL: "https://app.pagerduty.com/alerts/PALRT001",
		},
		CreatedAt:  "2026-03-19T10:00:00Z",
		Status:     "triggered",
		AlertKey:   "abc-123",
		Severity:   "warning",
		Suppressed: false,
		Service: pagerduty.APIObject{
			ID:      "PSVC001",
			Type:    "service_reference",
			Summary: "Web Tier",
			Self:    "https://api.pagerduty.com/services/PSVC001",
			HTMLURL: "https://app.pagerduty.com/service-directory/PSVC001",
		},
		Incident: pagerduty.APIReference{
			ID:   "PINC001",
			Type: "incident_reference",
		},
		Integration: pagerduty.APIObject{
			ID:   "PINT001",
			Type: "generic_email_inbound_integration_reference",
		},
		Body: cefAlertBody(),
	}
}

func TestProjectAlertForAgent_FullCEFAlert(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	got := ProjectAlertForAgent(a)

	assert.Equal(t, "PALRT001", got.ID)
	assert.Equal(t, "triggered", got.Status)
	assert.Equal(t, "warning", got.Severity)
	assert.Equal(t, "[TRIGGERED] CPU High on web-1", got.Summary)
	assert.Equal(t, "2026-03-19T10:00:00Z", got.CreatedAt)
	assert.Equal(t, "abc-123", got.AlertKey)
	assert.Equal(t, "https://app.pagerduty.com/alerts/PALRT001", got.HTMLURL)
	assert.False(t, got.Suppressed)
	assert.Equal(t, AgentRef{ID: "PSVC001", Summary: "Web Tier"}, got.Service)
	assert.Equal(t, "PINC001", got.IncidentID)

	require.NotNil(t, got.Body)
	assert.Equal(t, "Datadog", got.Body.Client)
	assert.Equal(t, "https://app.datadoghq.com/monitors/123", got.Body.ClientURL)
	require.NotNil(t, got.Body.Details)
	assert.Equal(t, "Alert fired for high CPU", got.Body.Details["body"])
	assert.Equal(t, "avg(last_5m):avg:system.cpu{host:web-1} > 90", got.Body.Details["query"])
	require.Len(t, got.Body.Contexts, 1)
	assert.Equal(t, "https://app.datadoghq.com/monitors/123", got.Body.Contexts[0]["href"])
}

func TestProjectAlertForAgent_NilBody(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	a.Body = nil

	got := ProjectAlertForAgent(a)
	assert.Nil(t, got.Body)
}

func TestProjectAlertForAgent_EmptyBody(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	a.Body = map[string]any{}

	got := ProjectAlertForAgent(a)
	assert.Nil(t, got.Body)
}

func TestProjectAlertForAgent_BodyWithUnexpectedTypes(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	a.Body = map[string]any{
		"cef_details": map[string]any{
			"details": "not a map",
		},
	}

	got := ProjectAlertForAgent(a)
	assert.Nil(t, got.Body)
}

func TestProjectAlertForAgent_NonCEFBody(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	a.Body = map[string]any{
		"details": map[string]any{
			"body":  "Custom event body",
			"query": "some-query",
		},
		"contexts": []any{
			map[string]any{
				"href": "https://example.com",
				"text": "Example",
				"type": "link",
			},
		},
	}

	got := ProjectAlertForAgent(a)
	require.NotNil(t, got.Body)
	assert.Empty(t, got.Body.Client)
	assert.Empty(t, got.Body.ClientURL)
	assert.Equal(t, "Custom event body", got.Body.Details["body"])
	require.Len(t, got.Body.Contexts, 1)
	assert.Equal(t, "https://example.com", got.Body.Contexts[0]["href"])
}

func TestProjectAlertsForAgent_Slice(t *testing.T) {
	t.Parallel()
	alerts := []pagerduty.IncidentAlert{fullAlert(), fullAlert()}
	alerts[1].APIObject.ID = "PALRT999"

	got := ProjectAlertsForAgent(alerts)
	require.Len(t, got, 2)
	assert.Equal(t, "PALRT001", got[0].ID)
	assert.Equal(t, "PALRT999", got[1].ID)
}

func TestProjectAlertForAgent_DroppedFields(t *testing.T) {
	t.Parallel()
	a := fullAlert()
	got := ProjectAlertForAgent(a)

	b, err := json.Marshal(got)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	for _, key := range []string{
		"self", "type", "integration",
		"description", "message",
	} {
		assert.NotContains(t, m, key)
	}

	if bodyMap, ok := m["body"].(map[string]any); ok {
		for _, key := range []string{
			"cef_details", "source_component", "service_group",
		} {
			assert.NotContains(t, bodyMap, key)
		}
	}
}
