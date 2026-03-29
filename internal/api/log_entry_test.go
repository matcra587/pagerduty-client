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

func TestListIncidentLogEntries(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/log_entries", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{
			"log_entries": [
				{
					"id": "L1",
					"type": "acknowledge_log_entry",
					"created_at": "2026-03-28T10:00:00Z",
					"agent": {"id": "U1", "type": "user_reference", "summary": "Alice"},
					"channel": {"type": "web_app"}
				},
				{
					"id": "L2",
					"type": "trigger_log_entry",
					"created_at": "2026-03-28T09:00:00Z"
				}
			],
			"limit": 100,
			"offset": 0,
			"more": false,
			"total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	entries, err := c.ListIncidentLogEntries(context.Background(), "P1", LogEntryOpts{})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "L1", entries[0].ID)
	assert.Equal(t, "acknowledge_log_entry", entries[0].Type)
	assert.Equal(t, "U1", entries[0].Agent.ID)
	assert.Equal(t, "L2", entries[1].ID)
}

func TestListIncidentLogEntries_WithFilters(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/log_entries", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-03-01T00:00:00Z", r.URL.Query().Get("since"))
		assert.Equal(t, "2026-03-28T00:00:00Z", r.URL.Query().Get("until"))
		assert.Equal(t, "true", r.URL.Query().Get("is_overview"))
		_, _ = w.Write([]byte(`{"log_entries": [], "limit": 100, "offset": 0, "more": false}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	entries, err := c.ListIncidentLogEntries(context.Background(), "P1", LogEntryOpts{
		Since:      "2026-03-01T00:00:00Z",
		Until:      "2026-03-28T00:00:00Z",
		IsOverview: true,
	})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestListIncidentLogEntries_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var callCount atomic.Int32
	mux.HandleFunc("/incidents/P1/log_entries", func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			_, _ = w.Write([]byte(`{
				"log_entries": [{"id": "L1", "type": "trigger_log_entry"}],
				"limit": 1, "offset": 0, "more": true, "total": 2
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"log_entries": [{"id": "L2", "type": "resolve_log_entry"}],
				"limit": 1, "offset": 1, "more": false, "total": 2
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	entries, err := c.ListIncidentLogEntries(context.Background(), "P1", LogEntryOpts{})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "L1", entries[0].ID)
	assert.Equal(t, "L2", entries[1].ID)
}
