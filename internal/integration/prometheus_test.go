package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prometheusEnv() AlertEnvelope {
	return UnwrapAlert(map[string]any{
		"cef_details": map[string]any{
			"client":     "Alertmanager",
			"client_url": "http://alertmanager.example/",
			"details": map[string]any{
				"custom_details": map[string]any{
					"firing":       "Labels:\n - alertname = HighErrorRate\n - severity = critical\n",
					"num_firing":   "1",
					"num_resolved": "0",
					"resolved":     "",
				},
			},
		},
	})
}

func prometheusStructuredEnv() AlertEnvelope {
	return UnwrapAlert(map[string]any{
		"client":     "Prometheus Alertmanager",
		"client_url": "http://alertmanager.example/",
		"details": map[string]any{
			"custom_details": map[string]any{
				"alertname":   "HighLatency",
				"cluster":     "prod-cluster",
				"severity":    "critical",
				"description": "P95 latency > 1s for 5m",
				"runbook":     "https://runbooks.example.com/high-latency",
				"num_firing":  "1",
			},
		},
	})
}

func TestPrometheus_MatchesCEFPayload(t *testing.T) {
	t.Parallel()
	s, ok := Prometheus{}.Normalise(prometheusEnv())
	assert.True(t, ok)
	assert.Equal(t, "Prometheus Alertmanager", s.Source)
}

func TestPrometheus_MatchesTopLevelPayload(t *testing.T) {
	t.Parallel()
	s, ok := Prometheus{}.Normalise(prometheusStructuredEnv())
	assert.True(t, ok)
	assert.Equal(t, "Prometheus Alertmanager", s.Source)
}

func TestPrometheus_RejectsNonPrometheus(t *testing.T) {
	t.Parallel()
	_, ok := Prometheus{}.Normalise(gcpEnv())
	assert.False(t, ok)
}

func TestPrometheus_ExtractsDefaultFields(t *testing.T) {
	t.Parallel()
	s, _ := Prometheus{}.Normalise(prometheusEnv())
	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "1", fieldMap["Firing"])
	assert.Contains(t, fieldMap["Firing alerts"], "HighErrorRate")
	require.Len(t, s.Links, 1)
	assert.Equal(t, "Alertmanager", s.Links[0].Label)
}

func TestPrometheus_ExtractsStructuredFields(t *testing.T) {
	t.Parallel()
	s, _ := Prometheus{}.Normalise(prometheusStructuredEnv())
	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "HighLatency", fieldMap["Alert"])
	assert.Equal(t, "critical", fieldMap["Severity"])
	assert.Equal(t, "P95 latency > 1s for 5m", fieldMap["Description"])
}
