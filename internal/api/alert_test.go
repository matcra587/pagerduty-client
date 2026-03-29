package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidentAlerts(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"alerts": [
				{"id": "A1", "status": "triggered"},
				{"id": "A2", "status": "resolved"}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	alerts, err := c.ListIncidentAlerts(context.Background(), "P1")
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
	assert.Equal(t, "A1", alerts[0].ID)
	assert.Equal(t, "A2", alerts[1].ID)
}

func TestListIncidentAlerts_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var callCount atomic.Int32
	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{
				"alerts": [{"id": "A1"}, {"id": "A2"}],
				"limit": 2, "offset": 0, "more": true, "total": 4
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"alerts": [{"id": "A3"}, {"id": "A4"}],
				"limit": 2, "offset": 2, "more": false, "total": 4
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	alerts, err := c.ListIncidentAlerts(context.Background(), "P1")
	require.NoError(t, err)
	assert.Len(t, alerts, 4)
	assert.Equal(t, int32(2), callCount.Load())
}

func TestGetIncidentAlert(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/alerts/A1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"alert": {
				"id": "A1",
				"status": "triggered"
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	alert, err := c.GetIncidentAlert(context.Background(), "P1", "A1")
	require.NoError(t, err)
	require.NotNil(t, alert)
	assert.Equal(t, "A1", alert.ID)
	assert.Equal(t, "triggered", alert.Status)
}

func TestResolveAlerts(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents/P1/alerts", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var alerts []map[string]string
		if !assert.NoError(t, json.Unmarshal(body["alerts"], &alerts)) {
			return
		}
		if !assert.Len(t, alerts, 1) {
			return
		}
		assert.Equal(t, "A1", alerts[0]["id"])
		assert.Equal(t, "alert", alerts[0]["type"])
		assert.Equal(t, "resolved", alerts[0]["status"])

		_, _ = w.Write([]byte(`{"alerts": [{"id": "A1", "status": "resolved"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.ResolveAlerts(context.Background(), "P1", "user@example.com", []string{"A1"})
	require.NoError(t, err)
}

func TestResolveAlerts_Multiple(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents/P1/alerts", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var alerts []map[string]string
		if !assert.NoError(t, json.Unmarshal(body["alerts"], &alerts)) {
			return
		}
		if !assert.Len(t, alerts, 3) {
			return
		}
		assert.Equal(t, "A1", alerts[0]["id"])
		assert.Equal(t, "A2", alerts[1]["id"])
		assert.Equal(t, "A3", alerts[2]["id"])
		for _, a := range alerts {
			assert.Equal(t, "resolved", a["status"])
		}

		_, _ = w.Write([]byte(`{"alerts": [{"id": "A1"}, {"id": "A2"}, {"id": "A3"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.ResolveAlerts(context.Background(), "P1", "user@example.com", []string{"A1", "A2", "A3"})
	require.NoError(t, err)
}

func TestResolveAlerts_Empty(t *testing.T) {
	t.Parallel()
	c := NewClient("test-token")
	err := c.ResolveAlerts(context.Background(), "P1", "user@example.com", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "at least one alert ID")
}
