package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListServices(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"services": [
				{"id": "PSVC1", "name": "Web App", "status": "active"},
				{"id": "PSVC2", "name": "API Gateway", "status": "active"}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	services, err := c.ListServices(context.Background(), ListServicesOpts{})
	require.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Equal(t, "PSVC1", services[0].ID)
	assert.Equal(t, "Web App", services[0].Name)
}

func TestListServices_WithTeamFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		teamIDs := q["team_ids[]"]
		assert.Equal(t, []string{"T1", "T2"}, teamIDs)
		_, _ = w.Write([]byte(`{
			"services": [{"id": "PSVC1", "name": "Web App"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	services, err := c.ListServices(context.Background(), ListServicesOpts{
		TeamIDs: []string{"T1", "T2"},
	})
	require.NoError(t, err)
	assert.Len(t, services, 1)
}

func TestListServices_WithQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "web", q.Get("query"))
		assert.Equal(t, "name:asc", q.Get("sort_by"))
		_, _ = w.Write([]byte(`{
			"services": [{"id": "PSVC1", "name": "Web App"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	services, err := c.ListServices(context.Background(), ListServicesOpts{
		Query:  "web",
		SortBy: "name:asc",
	})
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, "PSVC1", services[0].ID)
}

func TestGetService(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/services/PSVC1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"service": {
				"id": "PSVC1",
				"name": "Web App",
				"status": "active",
				"description": "Production web application"
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	svc, err := c.GetService(context.Background(), "PSVC1")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "PSVC1", svc.ID)
	assert.Equal(t, "Web App", svc.Name)
	assert.Equal(t, "active", svc.Status)
	assert.Equal(t, "Production web application", svc.Description)
}

func TestGetService_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/services/NOTEXIST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Service not found","code":2100}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	svc, err := c.GetService(context.Background(), "NOTEXIST")
	require.Error(t, err)
	assert.Nil(t, svc)
}
