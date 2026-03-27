package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnwrapAlert_TopLevel(t *testing.T) {
	t.Parallel()
	body := map[string]any{
		"client":     "Alertmanager",
		"client_url": "http://am.example/",
		"details": map[string]any{
			"custom_details": map[string]any{"key": "value"},
		},
	}
	env := UnwrapAlert(body)
	assert.Equal(t, "Alertmanager", env.Client)
	assert.Equal(t, "http://am.example/", env.ClientURL)
	require.NotNil(t, env.CustomDetails)
	assert.Equal(t, "value", env.CustomDetails["key"])
}

func TestUnwrapAlert_CEFDetails(t *testing.T) {
	t.Parallel()
	body := map[string]any{
		"cef_details": map[string]any{
			"client":     "Datadog",
			"client_url": "https://app.datadoghq.com/monitors/1",
			"details": map[string]any{
				"custom_details": map[string]any{"monitor": "cpu"},
			},
		},
	}
	env := UnwrapAlert(body)
	assert.Equal(t, "Datadog", env.Client)
	assert.Equal(t, "https://app.datadoghq.com/monitors/1", env.ClientURL)
	require.NotNil(t, env.CustomDetails)
	assert.Equal(t, "cpu", env.CustomDetails["monitor"])
}

func TestUnwrapAlert_CEFDetailsFallback(t *testing.T) {
	t.Parallel()
	// Datadog V1: cef_details.details IS the field map (no custom_details).
	body := map[string]any{
		"cef_details": map[string]any{
			"client": "Datadog",
			"details": map[string]any{
				"query": "avg:cpu > 90",
				"title": "CPU high",
			},
		},
	}
	env := UnwrapAlert(body)
	assert.Equal(t, "Datadog", env.Client)
	assert.Equal(t, "avg:cpu > 90", env.CustomDetails["query"])
}

func TestUnwrapAlert_NilBody(t *testing.T) {
	t.Parallel()
	env := UnwrapAlert(nil)
	assert.Empty(t, env.Client)
	assert.Nil(t, env.CustomDetails)
}

func TestDetect_UnknownPayload_ReturnsGeneric(t *testing.T) {
	t.Parallel()
	s := Detect(map[string]any{"unknown_key": "value"})
	assert.Equal(t, "Unknown", s.Source)
}

func TestDetect_NilBody_ReturnsGeneric(t *testing.T) {
	t.Parallel()
	s := Detect(nil)
	assert.Equal(t, "Unknown", s.Source)
}

func TestDetect_GCPPayload(t *testing.T) {
	t.Parallel()
	s := Detect(gcpEnv().Raw)
	assert.Equal(t, "Google Cloud Monitoring", s.Source)
}

func TestDetect_CloudWatchPayload(t *testing.T) {
	t.Parallel()
	s := Detect(cloudwatchEnv().Raw)
	assert.Equal(t, "AWS CloudWatch", s.Source)
}

func TestDetect_DatadogPayload(t *testing.T) {
	t.Parallel()
	s := Detect(datadogEnv().Raw)
	assert.Equal(t, "Datadog", s.Source)
}

func TestDetect_PrometheusPayload(t *testing.T) {
	t.Parallel()
	s := Detect(prometheusEnv().Raw)
	assert.Equal(t, "Prometheus Alertmanager", s.Source)
}

func TestDetect_PayloadCustomDetails(t *testing.T) {
	t.Parallel()
	body := map[string]any{
		"payload": map[string]any{
			"custom_details": map[string]any{"alert": "test"},
		},
	}
	env := UnwrapAlert(body)
	require.NotNil(t, env.CustomDetails)
	assert.Equal(t, "test", env.CustomDetails["alert"])
}

func TestDetect_CEFPayloadCustomDetails(t *testing.T) {
	t.Parallel()
	body := map[string]any{
		"cef_details": map[string]any{
			"payload": map[string]any{
				"custom_details": map[string]any{"alert": "cef-test"},
			},
		},
	}
	env := UnwrapAlert(body)
	require.NotNil(t, env.CustomDetails)
	assert.Equal(t, "cef-test", env.CustomDetails["alert"])
}
