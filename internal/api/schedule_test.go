package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSchedules(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"schedules":[{"id":"S1","summary":"Primary","name":"Primary On-Call"}],"limit":100,"offset":0,"more":false,"total":1}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	schedules, err := client.ListSchedules(context.Background(), ListSchedulesOpts{})

	require.NoError(t, err)
	assert.Len(t, schedules, 1)
	assert.Equal(t, "S1", schedules[0].ID)
	assert.Equal(t, "Primary On-Call", schedules[0].Name)
}

func TestListSchedules_WithQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "primary", r.URL.Query().Get("query"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"schedules":[{"id":"S1","name":"Primary On-Call"}],"limit":100,"offset":0,"more":false,"total":1}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	schedules, err := client.ListSchedules(context.Background(), ListSchedulesOpts{Query: "primary"})

	require.NoError(t, err)
	assert.Len(t, schedules, 1)
}

func TestListSchedules_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var page atomic.Int32
	mux.HandleFunc("/schedules", func(w http.ResponseWriter, r *http.Request) {
		n := page.Add(1)
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			_, _ = w.Write([]byte(`{"schedules":[{"id":"S1","name":"Schedule 1"}],"limit":1,"offset":0,"more":true,"total":2}`))
		} else {
			_, _ = w.Write([]byte(`{"schedules":[{"id":"S2","name":"Schedule 2"}],"limit":1,"offset":1,"more":false,"total":2}`))
		}
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	schedules, err := client.ListSchedules(context.Background(), ListSchedulesOpts{})

	require.NoError(t, err)
	assert.Len(t, schedules, 2)
	assert.Equal(t, "S1", schedules[0].ID)
	assert.Equal(t, "S2", schedules[1].ID)
}

func TestGetSchedule(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules/S1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"schedule":{"id":"S1","name":"Primary On-Call","time_zone":"UTC"}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	schedule, err := client.GetSchedule(context.Background(), "S1")

	require.NoError(t, err)
	require.NotNil(t, schedule)
	assert.Equal(t, "S1", schedule.ID)
	assert.Equal(t, "Primary On-Call", schedule.Name)
	assert.Equal(t, "UTC", schedule.TimeZone)
}

func TestGetSchedule_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules/NOTFOUND", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Schedule not found","code":2100}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	schedule, err := client.GetSchedule(context.Background(), "NOTFOUND")

	require.Error(t, err)
	assert.Nil(t, schedule)
	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestListOverrides(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules/S1/overrides", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "2024-01-01T00:00:00Z", r.URL.Query().Get("since"))
		assert.Equal(t, "2024-01-07T00:00:00Z", r.URL.Query().Get("until"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"overrides":[{"id":"OV1","start":"2024-01-02T00:00:00Z","end":"2024-01-03T00:00:00Z","user":{"id":"U1","type":"user"}}]}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	overrides, err := client.ListOverrides(context.Background(), "S1", "2024-01-01T00:00:00Z", "2024-01-07T00:00:00Z")

	require.NoError(t, err)
	assert.Len(t, overrides, 1)
	assert.Equal(t, "OV1", overrides[0].ID)
	assert.Equal(t, "U1", overrides[0].User.ID)
}

func TestCreateOverride(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules/S1/overrides", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "admin@example.com", r.Header.Get("From"))

		var body map[string]any
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}
		overrides, ok := body["overrides"].([]any)
		if !assert.True(t, ok, "expected overrides array") {
			return
		}
		if !assert.Len(t, overrides, 1) {
			return
		}
		override, ok := overrides[0].(map[string]any)
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, "2024-01-02T00:00:00Z", override["start"])
		assert.Equal(t, "2024-01-03T00:00:00Z", override["end"])
		user, ok := override["user"].(map[string]any)
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, "U1", user["id"])
		assert.Equal(t, "user_reference", user["type"])

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[{"status":201,"override":{"id":"OV1","start":"2024-01-02T00:00:00Z","end":"2024-01-03T00:00:00Z","user":{"id":"U1","type":"user_reference"}}}]`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	err := client.CreateOverride(context.Background(), "S1", "admin@example.com", CreateOverrideOpts{
		UserID: "U1",
		Start:  "2024-01-02T00:00:00Z",
		End:    "2024-01-03T00:00:00Z",
	})

	require.NoError(t, err)
}

func TestCreateOverride_Error(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/schedules/S1/overrides", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid override","code":2001}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	err := client.CreateOverride(context.Background(), "S1", "admin@example.com", CreateOverrideOpts{
		UserID: "U1",
		Start:  "bad-date",
		End:    "bad-date",
	})

	require.Error(t, err)
	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}
