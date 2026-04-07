package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProbeAPI_Reachable(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities", r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	ok, code := ProbeAPI(context.Background(), server.URL)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestProbeAPI_Unreachable(t *testing.T) {
	t.Parallel()
	ok, code := ProbeAPI(context.Background(), "http://127.0.0.1:1")
	assert.False(t, ok)
	assert.Equal(t, 0, code)
}

func TestProbeAPI_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	ok, code := ProbeAPI(context.Background(), server.URL)
	assert.False(t, ok)
	assert.Equal(t, http.StatusServiceUnavailable, code)
}
