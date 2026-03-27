package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func datadogEnv() AlertEnvelope {
	return UnwrapAlert(map[string]any{
		"cef_details": map[string]any{
			"client":     "Datadog",
			"client_url": "https://app.datadoghq.com/monitors/123",
			"details": map[string]any{
				"body":          "Alert fired for high CPU",
				"query":         "avg(last_5m):avg:system.cpu{host:web-1} > 90",
				"event_type":    "metric_alert",
				"monitor_state": "Alert",
				"org":           "myorg",
				"priority":      "P2",
				"tags":          "host:web-1,env:prod",
				"title":         "CPU is too high on web-1",
			},
		},
	})
}

func TestDatadog_MatchesPayload(t *testing.T) {
	t.Parallel()
	s, ok := Datadog{}.Normalise(datadogEnv())
	assert.True(t, ok)
	assert.Equal(t, "Datadog", s.Source)
}

func TestDatadog_RejectsNonDatadog(t *testing.T) {
	t.Parallel()
	_, ok := Datadog{}.Normalise(gcpEnv())
	assert.False(t, ok)
}

func TestDatadog_ExtractsFields(t *testing.T) {
	t.Parallel()
	s, _ := Datadog{}.Normalise(datadogEnv())
	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "CPU is too high on web-1", fieldMap["Title"])
	assert.Equal(t, "avg(last_5m):avg:system.cpu{host:web-1} > 90", fieldMap["Query"])
	assert.Equal(t, "Alert", fieldMap["State"])
	assert.Equal(t, "host:web-1,env:prod", fieldMap["Tags"])
	require.Len(t, s.Links, 1)
	assert.Contains(t, s.Links[0].URL, "datadoghq.com")
}
