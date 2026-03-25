package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTeams(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"teams": [
				{"id": "PT1", "name": "Platform"},
				{"id": "PT2", "name": "Backend"}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	teams, err := c.ListTeams(context.Background(), ListTeamsOpts{})
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Equal(t, "PT1", teams[0].ID)
	assert.Equal(t, "Platform", teams[0].Name)
}

func TestListTeams_WithQuery(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "platform", q.Get("query"))
		_, _ = w.Write([]byte(`{
			"teams": [{"id": "PT1", "name": "Platform"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	teams, err := c.ListTeams(context.Background(), ListTeamsOpts{
		Query: "platform",
	})
	require.NoError(t, err)
	assert.Len(t, teams, 1)
	assert.Equal(t, "PT1", teams[0].ID)
}

func TestListTeams_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	callCount := 0
	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"teams": [{"id": "PT1"}, {"id": "PT2"}],
				"limit": 2, "offset": 0, "more": true, "total": 3
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"teams": [{"id": "PT3"}],
				"limit": 2, "offset": 2, "more": false, "total": 3
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	teams, err := c.ListTeams(context.Background(), ListTeamsOpts{})
	require.NoError(t, err)
	assert.Len(t, teams, 3)
	assert.Equal(t, 2, callCount)
}

func TestGetTeam(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams/PT1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"team": {
				"id": "PT1",
				"name": "Platform",
				"description": "Platform engineering team"
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	team, err := c.GetTeam(context.Background(), "PT1")
	require.NoError(t, err)
	require.NotNil(t, team)
	assert.Equal(t, "PT1", team.ID)
	assert.Equal(t, "Platform", team.Name)
	assert.Equal(t, "Platform engineering team", team.Description)
}

func TestGetTeam_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams/NOTEXIST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Team not found","code":2100}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	team, err := c.GetTeam(context.Background(), "NOTEXIST")
	require.Error(t, err)
	assert.Nil(t, team)
}
