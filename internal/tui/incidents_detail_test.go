package tui

import (
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
)

func TestBodySection_UsesNormaliser_GCP(t *testing.T) {
	m := incidentDetail{
		incident: testIncidents()[0],
		alerts: []pagerduty.IncidentAlert{{
			Body: map[string]any{
				"details": map[string]any{
					"custom_details": map[string]any{
						"incident": map[string]any{
							"policy_name":     "Uptime failure",
							"condition_name":  "Check failed",
							"condition":       map[string]any{},
							"observed_value":  "2.0",
							"threshold_value": "1",
							"state":           "open",
							"url":             "https://console.cloud.google.com/test",
						},
						"version": "1.2",
					},
				},
			},
		}},
		width: 80,
	}
	body := m.bodySection()
	assert.Contains(t, body, "Google Cloud Monitoring")
	assert.Contains(t, body, "Uptime failure")
	assert.Contains(t, body, "Check failed")
	assert.NotContains(t, body, "map[")
}
