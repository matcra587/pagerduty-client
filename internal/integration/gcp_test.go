package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gcpEnv() AlertEnvelope {
	return UnwrapAlert(map[string]any{
		"details": map[string]any{
			"custom_details": map[string]any{
				"incident": map[string]any{
					"policy_name":                "HomeAssistant uptime failure",
					"condition_name":             "Failure of uptime check_id homeassistant-3F",
					"condition":                  map[string]any{"conditionThreshold": map[string]any{}},
					"summary":                    "An uptime check is failing.",
					"observed_value":             "2.000",
					"threshold_value":            "1",
					"state":                      "open",
					"severity":                   "No severity",
					"url":                        "https://console.cloud.google.com/monitoring/alerting/alerts/0.abc",
					"resource_type_display_name": "Uptime Check URL",
					"resource": map[string]any{
						"type":   "uptime_url",
						"labels": map[string]any{"host": "monitor.example.com", "project_id": "my-project"},
					},
					"metric": map[string]any{
						"type":        "monitoring.googleapis.com/uptime_check/check_passed",
						"displayName": "Check passed",
					},
					"documentation": map[string]any{
						"content": "Check your HA instance",
						"links": []any{
							map[string]any{"displayName": "Playbook", "url": "https://wiki.example.com/ha-playbook"},
						},
					},
				},
				"version": "1.2",
			},
		},
	})
}

func TestGCP_MatchesPayload(t *testing.T) {
	_, ok := GCP{}.Normalise(gcpEnv())
	assert.True(t, ok)
}

func TestGCP_RejectsNonGCP(t *testing.T) {
	env := UnwrapAlert(map[string]any{
		"details": map[string]any{
			"custom_details": map[string]any{"monitor": "cpu_high"},
		},
	})
	_, ok := GCP{}.Normalise(env)
	assert.False(t, ok)
}

func TestGCP_ExtractsFields(t *testing.T) {
	s, _ := GCP{}.Normalise(gcpEnv())

	assert.Equal(t, "Google Cloud Monitoring", s.Source)
	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}

	assert.Equal(t, "HomeAssistant uptime failure", fieldMap["Policy"])
	assert.Equal(t, "Failure of uptime check_id homeassistant-3F", fieldMap["Condition"])
	assert.Equal(t, "2.000 (threshold: 1)", fieldMap["Observed"])
	assert.Equal(t, "open", fieldMap["State"])
	assert.Contains(t, fieldMap["Resource"], "monitor.example.com")
	assert.Contains(t, fieldMap["Metric"], "Check passed")
}

func TestGCP_ExtractsLinks(t *testing.T) {
	s, _ := GCP{}.Normalise(gcpEnv())

	require.GreaterOrEqual(t, len(s.Links), 2)
	assert.Equal(t, "GCP Console", s.Links[0].Label)
	assert.Contains(t, s.Links[0].URL, "console.cloud.google.com")
	assert.Equal(t, "Playbook", s.Links[1].Label)
	assert.Contains(t, s.Links[1].URL, "wiki.example.com")
}
