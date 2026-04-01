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

func TestListMaintenanceWindows(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"Deploy window","start_time":"2026-03-31T20:00:00Z","end_time":"2026-03-31T22:00:00Z","services":[{"id":"S1","type":"service_reference","summary":"Auth API"}]}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{})

	require.NoError(t, err)
	require.Len(t, windows, 1)
	assert.Equal(t, "PW1", windows[0].ID)
	assert.Equal(t, "Deploy window", windows[0].Description)
	assert.Equal(t, "2026-03-31T20:00:00Z", windows[0].StartTime)
	require.Len(t, windows[0].Services, 1)
	assert.Equal(t, "Auth API", windows[0].Services[0].Summary)
}

func TestListMaintenanceWindows_WithQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deploy", r.URL.Query().Get("query"))
		_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"Deploy window"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{Query: "deploy"})

	require.NoError(t, err)
	assert.Len(t, windows, 1)
}

func TestListMaintenanceWindows_WithFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ongoing", r.URL.Query().Get("filter"))
		_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"Active window"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{Filter: "ongoing"})

	require.NoError(t, err)
	assert.Len(t, windows, 1)
}

func TestListMaintenanceWindows_WithTeamFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1", "T2"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"Team window"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{TeamIDs: []string{"T1", "T2"}})

	require.NoError(t, err)
	assert.Len(t, windows, 1)
}

func TestListMaintenanceWindows_WithServiceFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"S1"}, r.URL.Query()["service_ids[]"])
		_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"Service window"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{ServiceIDs: []string{"S1"}})

	require.NoError(t, err)
	assert.Len(t, windows, 1)
}

func TestListMaintenanceWindows_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var page atomic.Int32
	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, _ *http.Request) {
		n := page.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW1","description":"First"}],"limit":1,"offset":0,"more":true,"total":2}`))
		} else {
			_, _ = w.Write([]byte(`{"maintenance_windows":[{"id":"PW2","description":"Second"}],"limit":1,"offset":1,"more":false,"total":2}`))
		}
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	windows, err := client.ListMaintenanceWindows(context.Background(), ListMaintenanceWindowsOpts{})

	require.NoError(t, err)
	require.Len(t, windows, 2)
	assert.Equal(t, "PW1", windows[0].ID)
	assert.Equal(t, "PW2", windows[1].ID)
}

func TestGetMaintenanceWindow(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows/PW1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"maintenance_window":{"id":"PW1","description":"Deploy window","start_time":"2026-03-31T20:00:00Z","end_time":"2026-03-31T22:00:00Z","services":[{"id":"S1","type":"service_reference","summary":"Auth API"}],"teams":[{"id":"T1","type":"team_reference","summary":"Platform"}],"created_by":{"id":"U1","type":"user_reference","summary":"Alice Smith"}}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	mw, err := client.GetMaintenanceWindow(context.Background(), "PW1")

	require.NoError(t, err)
	require.NotNil(t, mw)
	assert.Equal(t, "PW1", mw.ID)
	assert.Equal(t, "Deploy window", mw.Description)
	assert.Equal(t, "2026-03-31T20:00:00Z", mw.StartTime)
	assert.Equal(t, "2026-03-31T22:00:00Z", mw.EndTime)
	require.Len(t, mw.Services, 1)
	assert.Equal(t, "Auth API", mw.Services[0].Summary)
	require.Len(t, mw.Teams, 1)
	assert.Equal(t, "Platform", mw.Teams[0].Summary)
	require.NotNil(t, mw.CreatedBy)
	assert.Equal(t, "Alice Smith", mw.CreatedBy.Summary)
}

func TestGetMaintenanceWindow_NilCreatedBy(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows/PW2", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"maintenance_window":{"id":"PW2","description":"Automated window","start_time":"2026-03-31T20:00:00Z","end_time":"2026-03-31T22:00:00Z","services":[]}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	mw, err := client.GetMaintenanceWindow(context.Background(), "PW2")

	require.NoError(t, err)
	require.NotNil(t, mw)
	assert.Equal(t, "PW2", mw.ID)
	assert.Nil(t, mw.CreatedBy)
}

func TestGetMaintenanceWindow_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /maintenance_windows/NOTFOUND", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Not Found","code":2100}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	mw, err := client.GetMaintenanceWindow(context.Background(), "NOTFOUND")

	require.Error(t, err)
	assert.Nil(t, mw)
	require.ErrorIs(t, err, ErrNotFound)
}
