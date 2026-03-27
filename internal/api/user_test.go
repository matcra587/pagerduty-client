package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"users": [
				{"id": "PU1", "name": "Alice Smith", "email": "alice@example.com"},
				{"id": "PU2", "name": "Bob Jones", "email": "bob@example.com"}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	users, err := c.ListUsers(context.Background(), ListUsersOpts{})
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "PU1", users[0].ID)
	assert.Equal(t, "Alice Smith", users[0].Name)
	assert.Equal(t, "alice@example.com", users[0].Email)
}

func TestListUsers_WithTeamFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		teamIDs := q["team_ids[]"]
		assert.Equal(t, []string{"T1", "T2"}, teamIDs)
		_, _ = w.Write([]byte(`{
			"users": [{"id": "PU1", "name": "Alice Smith"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	users, err := c.ListUsers(context.Background(), ListUsersOpts{
		TeamIDs: []string{"T1", "T2"},
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestListUsers_WithQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "alice", q.Get("query"))
		_, _ = w.Write([]byte(`{
			"users": [{"id": "PU1", "name": "Alice Smith"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	users, err := c.ListUsers(context.Background(), ListUsersOpts{
		Query: "alice",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "PU1", users[0].ID)
}

func TestGetUser(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users/PU1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"user": {
				"id": "PU1",
				"name": "Alice Smith",
				"email": "alice@example.com",
				"role": "admin"
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	user, err := c.GetUser(context.Background(), "PU1")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "PU1", user.ID)
	assert.Equal(t, "Alice Smith", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "admin", user.Role)
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users/NOTEXIST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"User not found","code":2100}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	user, err := c.GetUser(context.Background(), "NOTEXIST")
	require.Error(t, err)
	assert.Nil(t, user)
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users/me", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"user": {
				"id": "PU1",
				"name": "Alice Smith",
				"email": "alice@example.com"
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	user, err := c.GetCurrentUser(context.Background())
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "PU1", user.ID)
	assert.Equal(t, "Alice Smith", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestListContactMethods(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/users/PU1/contact_methods", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"contact_methods": [
				{"id": "CM1", "type": "email_contact_method", "address": "alice@example.com", "label": "Work"},
				{"id": "CM2", "type": "phone_contact_method", "address": "5551234567", "label": "Mobile"}
			]
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	methods, err := c.ListContactMethods(context.Background(), "PU1")
	require.NoError(t, err)
	assert.Len(t, methods, 2)
	assert.Equal(t, "CM1", methods[0].ID)
	assert.Equal(t, "email_contact_method", methods[0].Type)
	assert.Equal(t, "alice@example.com", methods[0].Address)
	assert.Equal(t, "Work", methods[0].Label)
}
