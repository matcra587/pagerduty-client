package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRelatedIncidents(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/related_incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{
			"related_incidents": [
				{
					"incident": {
						"id": "P200",
						"title": "Database connection pool exhausted",
						"status": "triggered",
						"urgency": "high"
					},
					"relationships": [
						{
							"type": "machine_learning_inferred",
							"metadata": {"grouping_classification": "similar_contents"}
						}
					]
				},
				{
					"incident": {
						"id": "P201",
						"title": "API latency spike",
						"status": "acknowledged",
						"urgency": "high"
					},
					"relationships": [
						{
							"type": "service_dependency",
							"metadata": {"dependent_services": [{"id": "S1", "type": "business_service_reference"}]}
						}
					]
				}
			]
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	related, err := c.ListRelatedIncidents(context.Background(), "P1")
	require.NoError(t, err)
	assert.Len(t, related, 2)
	assert.Equal(t, "P200", related[0].Incident.ID)
	assert.Equal(t, "Database connection pool exhausted", related[0].Incident.Title)
	require.Len(t, related[0].Relationships, 1)
	assert.Equal(t, "machine_learning_inferred", related[0].Relationships[0].Type)
	assert.Equal(t, "P201", related[1].Incident.ID)
	assert.Equal(t, "service_dependency", related[1].Relationships[0].Type)
}

func TestListRelatedIncidents_Empty(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/related_incidents", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"related_incidents": []}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	related, err := c.ListRelatedIncidents(context.Background(), "P1")
	require.NoError(t, err)
	assert.Empty(t, related)
}
