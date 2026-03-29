package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPastIncidents(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/past_incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{
			"past_incidents": [
				{
					"incident": {
						"id": "P100",
						"title": "CPU spike on web-01",
						"created_at": "2026-03-20T08:00:00Z",
						"self": "https://api.pagerduty.com/incidents/P100"
					},
					"score": 0.95
				},
				{
					"incident": {
						"id": "P101",
						"title": "CPU spike on web-02",
						"created_at": "2026-03-15T12:00:00Z",
						"self": "https://api.pagerduty.com/incidents/P101"
					},
					"score": 0.82
				}
			],
			"total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	past, err := c.ListPastIncidents(context.Background(), "P1", 5)
	require.NoError(t, err)
	assert.Len(t, past, 2)
	assert.Equal(t, "P100", past[0].Incident.ID)
	assert.Equal(t, "CPU spike on web-01", past[0].Incident.Title)
	assert.InDelta(t, 0.95, past[0].Score, 1e-9)
	assert.Equal(t, "P101", past[1].Incident.ID)
	assert.InDelta(t, 0.82, past[1].Score, 1e-9)
}

func TestListPastIncidents_Empty(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/past_incidents", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"past_incidents": [], "total": 0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	past, err := c.ListPastIncidents(context.Background(), "P1", 5)
	require.NoError(t, err)
	assert.Empty(t, past)
}

func TestListPastIncidents_ZeroLimitDefaultsToFive(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/past_incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"past_incidents": [], "total": 0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.ListPastIncidents(context.Background(), "P1", 0)
	require.NoError(t, err)
}

func TestListPastIncidents_CustomLimit(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/past_incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"past_incidents": [], "total": 0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.ListPastIncidents(context.Background(), "P1", 10)
	require.NoError(t, err)
}
