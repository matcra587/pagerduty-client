package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOnCalls(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/oncalls", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"oncalls": [
				{
					"escalation_level": 1,
					"user": {"id": "U1", "summary": "Alice"},
					"schedule": {"id": "S1", "summary": "Primary"},
					"escalation_policy": {"id": "EP1", "summary": "Default"}
				}
			],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	oncalls, err := c.ListOnCalls(context.Background(), ListOnCallsOpts{})
	require.NoError(t, err)
	assert.Len(t, oncalls, 1)
	assert.Equal(t, "U1", oncalls[0].User.APIObject.ID)
	assert.Equal(t, uint(1), oncalls[0].EscalationLevel)
}

func TestListOnCalls_WithFilters(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/oncalls", func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query()["schedule_ids[]"], "S1")
		_, _ = w.Write([]byte(`{"oncalls":[],"limit":100,"offset":0,"more":false,"total":0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	oncalls, err := c.ListOnCalls(context.Background(), ListOnCallsOpts{
		ScheduleIDs: []string{"S1"},
	})
	require.NoError(t, err)
	assert.Empty(t, oncalls)
}

func TestListOnCalls_WithMultipleFilters(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/oncalls", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Contains(t, q["user_ids[]"], "U1")
		assert.Contains(t, q["user_ids[]"], "U2")
		assert.Contains(t, q["escalation_policy_ids[]"], "EP1")
		assert.Equal(t, "2024-01-01T00:00:00Z", q.Get("since"))
		assert.Equal(t, "2024-01-31T23:59:59Z", q.Get("until"))
		_, _ = w.Write([]byte(`{"oncalls":[],"limit":100,"offset":0,"more":false,"total":0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	oncalls, err := c.ListOnCalls(context.Background(), ListOnCallsOpts{
		UserIDs:             []string{"U1", "U2"},
		EscalationPolicyIDs: []string{"EP1"},
		Since:               "2024-01-01T00:00:00Z",
		Until:               "2024-01-31T23:59:59Z",
	})
	require.NoError(t, err)
	assert.Empty(t, oncalls)
}

func TestListOnCalls_Earliest(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/oncalls", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("earliest"))
		_, _ = w.Write([]byte(`{"oncalls":[],"limit":100,"offset":0,"more":false,"total":0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	oncalls, err := c.ListOnCalls(context.Background(), ListOnCallsOpts{
		Earliest: true,
	})
	require.NoError(t, err)
	assert.Empty(t, oncalls)
}

func TestListOnCalls_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var callCount atomic.Int32
	mux.HandleFunc("/oncalls", func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{
				"oncalls": [
					{"escalation_level": 1, "user": {"id": "U1", "summary": "Alice"}},
					{"escalation_level": 2, "user": {"id": "U2", "summary": "Bob"}}
				],
				"limit": 2, "offset": 0, "more": true, "total": 3
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"oncalls": [
					{"escalation_level": 1, "user": {"id": "U3", "summary": "Charlie"}}
				],
				"limit": 2, "offset": 2, "more": false, "total": 3
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	oncalls, err := c.ListOnCalls(context.Background(), ListOnCallsOpts{})
	require.NoError(t, err)
	assert.Len(t, oncalls, 3)
	assert.Equal(t, "U1", oncalls[0].User.APIObject.ID)
	assert.Equal(t, "U2", oncalls[1].User.APIObject.ID)
	assert.Equal(t, "U3", oncalls[2].User.APIObject.ID)
	assert.Equal(t, int32(2), callCount.Load())
}
