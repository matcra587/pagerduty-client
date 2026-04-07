package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichIncident_NoAlerts(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"alerts": [], "limit": 100, "offset": 0, "more": false}`))
	})

	client := api.NewClient("test-token", api.WithBaseURL(server.URL))
	incident := &pagerduty.Incident{APIObject: pagerduty.APIObject{ID: "P1"}}
	result := enrichIncident(context.Background(), client, incident)
	assert.Equal(t, "P1", result.ID)
	assert.Nil(t, result.Integration)
}

func TestEnrichIncident_DatadogAlert(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"alerts": [{
				"id": "A1",
				"status": "triggered",
				"body": {
					"cef_details": {
						"client": "Datadog",
						"client_url": "https://app.datadoghq.com/monitors/123",
						"details": {
							"event_id": "9876543210",
							"monitor_state": "Alert",
							"query": "avg:cpu > 90",
							"tags": "env:prod,host:web-1"
						}
					}
				}
			}],
			"limit": 100, "offset": 0, "more": false
		}`))
	})

	client := api.NewClient("test-token", api.WithBaseURL(server.URL))
	incident := &pagerduty.Incident{APIObject: pagerduty.APIObject{ID: "P1"}}
	result := enrichIncident(context.Background(), client, incident)
	require.NotNil(t, result.Integration)
	assert.Equal(t, "Datadog", result.Integration.Source)
	assert.NotEmpty(t, result.Integration.Fields)
	assert.NotEmpty(t, result.Integration.Links)

	fieldMap := make(map[string]string)
	for _, f := range result.Integration.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "Alert", fieldMap["State"])
	assert.Equal(t, "avg:cpu > 90", fieldMap["Query"])
	assert.Equal(t, "9876543210", fieldMap["Event ID"])
}

func TestEnrichIncident_GCPAlert(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"alerts": [{
				"id": "A1",
				"status": "triggered",
				"body": {
					"cef_details": {
						"client": null,
						"details": {
							"incident": {
								"state": "open",
								"policy_name": "HomeAssistant uptime failure",
								"condition_name": "Failure of uptime check",
								"condition": {"displayName": "Failure of uptime check"},
								"observed_value": "2.000",
								"threshold_value": "1",
								"summary": "Uptime check is failing.",
								"url": "https://console.cloud.google.com/monitoring/alerting/incidents/test",
								"metric": {"type": "monitoring.googleapis.com/uptime_check/check_passed", "displayName": "Check passed"},
								"resource": {"labels": {"host": "home.example.com"}, "type": "uptime_url"}
							},
							"version": "1.2"
						}
					}
				}
			}],
			"limit": 100, "offset": 0, "more": false
		}`))
	})

	client := api.NewClient("test-token", api.WithBaseURL(server.URL))
	incident := &pagerduty.Incident{APIObject: pagerduty.APIObject{ID: "P1"}}
	result := enrichIncident(context.Background(), client, incident)
	require.NotNil(t, result.Integration)
	assert.Equal(t, "Google Cloud Monitoring", result.Integration.Source)
	assert.NotEmpty(t, result.Integration.Fields)

	fieldMap := make(map[string]string)
	for _, f := range result.Integration.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "open", fieldMap["State"])
	assert.Equal(t, "HomeAssistant uptime failure", fieldMap["Policy"])
}

func TestEnrichIncident_AlertFetchError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"server error","code":2010}}`))
	})

	client := api.NewClient("test-token", api.WithBaseURL(server.URL))
	incident := &pagerduty.Incident{APIObject: pagerduty.APIObject{ID: "P1"}}
	result := enrichIncident(context.Background(), client, incident)
	assert.Equal(t, "P1", result.ID)
	assert.Nil(t, result.Integration)
}
