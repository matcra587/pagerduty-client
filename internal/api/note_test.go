package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidentNotes(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/notes", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"notes": [
				{"id": "N1", "content": "First note"},
				{"id": "N2", "content": "Second note"}
			]
		}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	notes, err := c.ListIncidentNotes(context.Background(), "P1")
	require.NoError(t, err)
	assert.Len(t, notes, 2)
	assert.Equal(t, "N1", notes[0].ID)
	assert.Equal(t, "First note", notes[0].Content)
	assert.Equal(t, "N2", notes[1].ID)
}

func TestListIncidentNotes_Empty(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/incidents/P1/notes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"notes": []}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	notes, err := c.ListIncidentNotes(context.Background(), "P1")
	require.NoError(t, err)
	assert.Empty(t, notes)
}
