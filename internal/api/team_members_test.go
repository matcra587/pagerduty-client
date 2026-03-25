package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTeamMembers(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams/PT1/members", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"members": [
				{"user": {"id": "PU1", "summary": "Alice Smith"}},
				{"user": {"id": "PU2", "summary": "Bob Jones"}}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	members, err := c.ListTeamMembers(context.Background(), "PT1")
	require.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, "PU1", members[0].User.ID)
	assert.Equal(t, "Alice Smith", members[0].User.Summary)
	assert.Equal(t, "PU2", members[1].User.ID)
	assert.Equal(t, "Bob Jones", members[1].User.Summary)
}

func TestListTeamMembers_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	callCount := 0
	mux.HandleFunc("/teams/PT1/members", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"members": [{"user": {"id": "PU1", "summary": "Alice"}}],
				"limit": 1, "offset": 0, "more": true, "total": 2
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"members": [{"user": {"id": "PU2", "summary": "Bob"}}],
				"limit": 1, "offset": 1, "more": false, "total": 2
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	members, err := c.ListTeamMembers(context.Background(), "PT1")
	require.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, 2, callCount)
}

func TestListTeamMembers_Empty(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/teams/PT1/members", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"members": [],
			"limit": 100, "offset": 0, "more": false, "total": 0
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	members, err := c.ListTeamMembers(context.Background(), "PT1")
	require.NoError(t, err)
	assert.Empty(t, members)
}
