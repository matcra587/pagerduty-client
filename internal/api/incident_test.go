package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidents(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"incidents": [
				{"id": "P1", "status": "triggered", "title": "Server down"},
				{"id": "P2", "status": "acknowledged", "title": "High latency"}
			],
			"limit": 100, "offset": 0, "more": false, "total": 2
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	incidents, err := c.ListIncidents(context.Background(), ListIncidentsOpts{})
	require.NoError(t, err)
	assert.Len(t, incidents, 2)
	assert.Equal(t, "P1", incidents[0].ID)
	assert.Equal(t, "triggered", incidents[0].Status)
	assert.Equal(t, "Server down", incidents[0].Title)
}

func TestListIncidents_WithFilters(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		q := r.URL.Query()
		assert.Equal(t, "triggered", q.Get("statuses[]"))
		assert.Equal(t, "high", q.Get("urgencies[]"))
		_, _ = w.Write([]byte(`{
			"incidents": [{"id": "P1", "status": "triggered", "urgency": "high"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	incidents, err := c.ListIncidents(context.Background(), ListIncidentsOpts{
		Statuses:  []string{"triggered"},
		Urgencies: []string{"high"},
	})
	require.NoError(t, err)
	assert.Len(t, incidents, 1)
}

func TestListIncidents_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var callCount atomic.Int32
	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{
				"incidents": [{"id": "P1"}, {"id": "P2"}],
				"limit": 2, "offset": 0, "more": true, "total": 4
			}`))
		} else {
			_, _ = w.Write([]byte(`{
				"incidents": [{"id": "P3"}, {"id": "P4"}],
				"limit": 2, "offset": 2, "more": false, "total": 4
			}`))
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	incidents, err := c.ListIncidents(context.Background(), ListIncidentsOpts{})
	require.NoError(t, err)
	assert.Len(t, incidents, 4)
	assert.Equal(t, int32(2), callCount.Load())
}

func TestListIncidents_SinceUntil(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "2024-01-01T00:00:00Z", q.Get("since"))
		assert.Equal(t, "2024-01-31T23:59:59Z", q.Get("until"))
		_, _ = w.Write([]byte(`{"incidents": [], "limit": 100, "offset": 0, "more": false, "total": 0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.ListIncidents(context.Background(), ListIncidentsOpts{
		Since: "2024-01-01T00:00:00Z",
		Until: "2024-01-31T23:59:59Z",
	})
	require.NoError(t, err)
}

func TestListIncidents_DateRange(t *testing.T) {
	t.Parallel()
	v := incidentListParams(ListIncidentsOpts{
		DateRange: "all",
		Statuses:  []string{"triggered"},
	})
	assert.Equal(t, "all", v.Get("date_range"))
	assert.Equal(t, "triggered", v.Get("statuses[]"))
	assert.Empty(t, v.Get("since"))
	assert.Empty(t, v.Get("until"))
}

func TestListIncidents_SinceUntilWithoutDateRange(t *testing.T) {
	t.Parallel()
	v := incidentListParams(ListIncidentsOpts{
		Since: "2026-03-13T00:00:00Z",
		Until: "2026-03-20T00:00:00Z",
	})
	assert.Equal(t, "2026-03-13T00:00:00Z", v.Get("since"))
	assert.Equal(t, "2026-03-20T00:00:00Z", v.Get("until"))
	assert.Empty(t, v.Get("date_range"))
}

func TestListIncidents_TeamIDs(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		teamIDs := q["team_ids[]"]
		assert.Equal(t, []string{"T1", "T2"}, teamIDs)
		_, _ = w.Write([]byte(`{"incidents": [], "limit": 100, "offset": 0, "more": false, "total": 0}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.ListIncidents(context.Background(), ListIncidentsOpts{
		TeamIDs: []string{"T1", "T2"},
	})
	require.NoError(t, err)
}

func TestGetIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"incident": {
				"id": "P1",
				"title": "Server down",
				"status": "triggered",
				"service": {"id": "PSVC001", "type": "service_reference"}
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	incident, err := c.GetIncident(context.Background(), "P1")
	require.NoError(t, err)
	require.NotNil(t, incident)
	assert.Equal(t, "P1", incident.ID)
	assert.Equal(t, "Server down", incident.Title)
	assert.Equal(t, "triggered", incident.Status)
	assert.Equal(t, "PSVC001", incident.Service.ID)
}

func TestGetIncident_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/NOTEXIST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Incident not found","code":2100}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	incident, err := c.GetIncident(context.Background(), "NOTEXIST")
	require.Error(t, err)
	assert.Nil(t, incident)
}

func TestAckIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]string
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "P1", incidents[0]["id"])
		assert.Equal(t, "incident_reference", incidents[0]["type"])
		assert.Equal(t, "acknowledged", incidents[0]["status"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "status": "acknowledged"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.AckIncident(context.Background(), "P1", "user@example.com")
	require.NoError(t, err)
}

func TestAckIncident_FromHeaderCheck(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	received := ""
	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("From")
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "status": "acknowledged"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.AckIncident(context.Background(), "P1", "oncall@acme.com")
	require.NoError(t, err)
	assert.Equal(t, "oncall@acme.com", received)
}

func TestAckIncident_EmptyFromRejected(t *testing.T) {
	t.Parallel()
	c := NewClient("test-token")
	err := c.AckIncident(context.Background(), "P1", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "from email is required")
}

func TestResolveIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]string
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "P1", incidents[0]["id"])
		assert.Equal(t, "resolved", incidents[0]["status"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "status": "resolved"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.ResolveIncident(context.Background(), "P1", "user@example.com")
	require.NoError(t, err)
}

func TestSnoozeIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/snooze", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var duration int
		if !assert.NoError(t, json.Unmarshal(body["duration"], &duration)) {
			return
		}
		assert.Equal(t, 3600, duration)

		_, _ = w.Write([]byte(`{"incident": {"id": "P1", "status": "acknowledged"}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.SnoozeIncident(context.Background(), "P1", "user@example.com", time.Hour)
	require.NoError(t, err)
}

func TestReassignIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]json.RawMessage
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}

		var assignments []map[string]map[string]string
		if !assert.NoError(t, json.Unmarshal(incidents[0]["assignments"], &assignments)) {
			return
		}
		if !assert.Len(t, assignments, 2) {
			return
		}
		assert.Equal(t, "U1", assignments[0]["assignee"]["id"])
		assert.Equal(t, "U2", assignments[1]["assignee"]["id"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.ReassignIncident(context.Background(), "P1", "user@example.com", []string{"U1", "U2"})
	require.NoError(t, err)
}

func TestMergeIncidents(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/merge", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var sourceIncidents []map[string]string
		if !assert.NoError(t, json.Unmarshal(body["source_incidents"], &sourceIncidents)) {
			return
		}
		if !assert.Len(t, sourceIncidents, 2) {
			return
		}
		assert.Equal(t, "P2", sourceIncidents[0]["id"])
		assert.Equal(t, "incident_reference", sourceIncidents[0]["type"])
		assert.Equal(t, "P3", sourceIncidents[1]["id"])

		_, _ = w.Write([]byte(`{"incident": {"id": "P1"}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.MergeIncidents(context.Background(), "P1", "user@example.com", []string{"P2", "P3"})
	require.NoError(t, err)
}

func TestEscalateIncident(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /incidents/P1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"incident": {
				"id": "P1",
				"escalation_policy": {"id": "EP1", "type": "escalation_policy_reference"},
				"escalation_level": 1
			}
		}`))
	})

	mux.HandleFunc("GET /escalation_policies/EP1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"escalation_policy": {
				"id": "EP1",
				"escalation_rules": [
					{"id": "R1", "targets": [{"id": "U1", "type": "user_reference"}]},
					{"id": "R2", "targets": [{"id": "U2", "type": "user_reference"}, {"id": "U3", "type": "user_reference"}]},
					{"id": "R3", "targets": [{"id": "U4", "type": "user_reference"}]}
				]
			}
		}`))
	})

	reassigned := false
	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]json.RawMessage
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}

		var assignments []map[string]map[string]string
		if !assert.NoError(t, json.Unmarshal(incidents[0]["assignments"], &assignments)) {
			return
		}
		if !assert.Len(t, assignments, 2) {
			return
		}
		assert.Equal(t, "U2", assignments[0]["assignee"]["id"])
		assert.Equal(t, "U3", assignments[1]["assignee"]["id"])

		reassigned = true
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.EscalateIncident(context.Background(), "P1", "user@example.com")
	require.NoError(t, err)
	assert.True(t, reassigned, "expected reassign PUT to be called")
}

func TestEscalateIncident_AlreadyAtHighestLevel(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /incidents/P1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"incident": {
				"id": "P1",
				"escalation_policy": {"id": "EP1"},
				"escalation_level": 2
			}
		}`))
	})

	mux.HandleFunc("GET /escalation_policies/EP1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"escalation_policy": {
				"id": "EP1",
				"escalation_rules": [
					{"id": "R1", "targets": [{"id": "U1", "type": "user_reference"}]},
					{"id": "R2", "targets": [{"id": "U2", "type": "user_reference"}]}
				]
			}
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.EscalateIncident(context.Background(), "P1", "user@example.com")
	require.Error(t, err)
	assert.ErrorContains(t, err, "already at highest escalation level")
}

func TestEscalateIncident_Level0(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /incidents/P1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"incident": {
				"id": "P1",
				"escalation_policy": {"id": "EP1"},
				"escalation_level": 0
			}
		}`))
	})

	mux.HandleFunc("GET /escalation_policies/EP1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"escalation_policy": {
				"id": "EP1",
				"escalation_rules": [
					{"id": "R1", "targets": [{"id": "U1", "type": "user_reference"}]},
					{"id": "R2", "targets": [{"id": "U2", "type": "user_reference"}]}
				]
			}
		}`))
	})

	reassigned := false
	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]json.RawMessage
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}

		var assignments []map[string]map[string]string
		if !assert.NoError(t, json.Unmarshal(incidents[0]["assignments"], &assignments)) {
			return
		}
		if !assert.Len(t, assignments, 1) {
			return
		}
		assert.Equal(t, "U2", assignments[0]["assignee"]["id"])

		reassigned = true
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.EscalateIncident(context.Background(), "P1", "user@example.com")
	require.NoError(t, err)
	assert.True(t, reassigned, "expected reassign PUT to be called")
}

func TestAddIncidentNote(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/notes", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var note map[string]string
		if !assert.NoError(t, json.Unmarshal(body["note"], &note)) {
			return
		}
		assert.Equal(t, "This is a test note", note["content"])

		_, _ = w.Write([]byte(`{"note": {"id": "N1", "content": "This is a test note"}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.AddIncidentNote(context.Background(), "P1", "user@example.com", "This is a test note")
	require.NoError(t, err)
}

func TestListPriorities(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /priorities", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"priorities": [
				{"id": "PRIO1", "name": "P1", "description": "Critical"},
				{"id": "PRIO2", "name": "P2", "description": "High"},
				{"id": "PRIO3", "name": "P3", "description": "Medium"}
			]
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	priorities, err := c.ListPriorities(context.Background())
	require.NoError(t, err)
	assert.Len(t, priorities, 3)
	assert.Equal(t, "PRIO1", priorities[0].ID)
	assert.Equal(t, "P1", priorities[0].Name)
}

func TestUpdateIncident_Urgency(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]any
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "P1", incidents[0]["id"])
		assert.Equal(t, "incident_reference", incidents[0]["type"])
		assert.Equal(t, "high", incidents[0]["urgency"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "urgency": "high"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	inc, err := c.UpdateIncident(context.Background(), "P1", "user@example.com", UpdateOpts{
		Urgency: new("high"),
	})
	require.NoError(t, err)
	require.NotNil(t, inc)
	assert.Equal(t, "P1", inc.ID)
	assert.Equal(t, "high", inc.Urgency)
}

func TestUpdateIncident_Title(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]any
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "P1", incidents[0]["id"])
		assert.Equal(t, "New title", incidents[0]["title"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "title": "New title"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	inc, err := c.UpdateIncident(context.Background(), "P1", "user@example.com", UpdateOpts{
		Title: new("New title"),
	})
	require.NoError(t, err)
	require.NotNil(t, inc)
	assert.Equal(t, "New title", inc.Title)
}

func TestUpdateIncident_MultipleFields(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]any
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "Updated title", incidents[0]["title"])
		assert.Equal(t, "low", incidents[0]["urgency"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "title": "Updated title", "urgency": "low"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	inc, err := c.UpdateIncident(context.Background(), "P1", "user@example.com", UpdateOpts{
		Title:   new("Updated title"),
		Urgency: new("low"),
	})
	require.NoError(t, err)
	require.NotNil(t, inc)
	assert.Equal(t, "Updated title", inc.Title)
	assert.Equal(t, "low", inc.Urgency)
}

func TestUpdateIncident_ClearPriority(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]json.RawMessage
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		// priority should be JSON null
		assert.JSONEq(t, "null", string(incidents[0]["priority"]))

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.UpdateIncident(context.Background(), "P1", "user@example.com", UpdateOpts{
		Priority: new(""),
	})
	require.NoError(t, err)
}

func TestSetUrgency(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]any
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "low", incidents[0]["urgency"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "urgency": "low"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.SetUrgency(context.Background(), "P1", "user@example.com", "low")
	require.NoError(t, err)
}

func TestSetTitle(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]any
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}
		assert.Equal(t, "Fixed title", incidents[0]["title"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "title": "Fixed title"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.SetTitle(context.Background(), "P1", "user@example.com", "Fixed title")
	require.NoError(t, err)
}

func TestUpdatePriority(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("PUT /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user@example.com", r.Header.Get("From"))

		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}

		var incidents []map[string]json.RawMessage
		if !assert.NoError(t, json.Unmarshal(body["incidents"], &incidents)) {
			return
		}
		if !assert.Len(t, incidents, 1) {
			return
		}

		var priority map[string]string
		if !assert.NoError(t, json.Unmarshal(incidents[0]["priority"], &priority)) {
			return
		}
		assert.Equal(t, "PRIO-UUID-1", priority["id"])
		assert.Equal(t, "priority_reference", priority["type"])

		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}]}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	err := c.UpdatePriority(context.Background(), "P1", "user@example.com", "PRIO-UUID-1")
	require.NoError(t, err)
}
