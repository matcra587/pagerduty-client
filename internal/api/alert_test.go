package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidentAlerts(t *testing.T) {
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
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	callCount := 0
	mux.HandleFunc("/incidents/P1/alerts", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
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
	assert.Equal(t, 2, callCount)
}

func TestGetIncidentAlert(t *testing.T) {
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
