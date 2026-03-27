package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-token")
	assert.Equal(t, "https://api.pagerduty.com", c.baseURL)
	assert.Equal(t, "test-token", c.token)
}

func TestClientAuthHeader(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.pagerduty+json;version=2", r.Header.Get("Accept"))
		assert.Contains(t, r.Header.Get("User-Agent"), "pagerduty-client/")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)
	require.NoError(t, err)
}

func TestClientAPIError(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Not Found","code":2001}}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)

	require.Error(t, err)
	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestClientRetryOn429(t *testing.T) {
	var attempts atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestGetRejectsInvalidPath(t *testing.T) {
	c := NewClient("tok")
	_, err := c.get(context.Background(), "/incidents/../../admin", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path segment")
}

func TestPutFromRejectsInvalidPath(t *testing.T) {
	c := NewClient("tok")
	_, err := c.putFrom(context.Background(), "/incidents/../../admin", nil, "user@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path segment")
}

func TestPostFromRejectsInvalidPath(t *testing.T) {
	c := NewClient("tok")
	_, err := c.postFrom(context.Background(), "/incidents/../../admin", nil, "user@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path segment")
}

func TestValidatePathRejectsQueryString(t *testing.T) {
	err := validatePath("/incidents?foo=bar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query string")
}

func TestValidatePathSegment(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		wantErr bool
	}{
		{"valid alphanumeric", "PABCDEF", false},
		{"valid with hyphens", "P123-ABCD", false},
		{"valid with underscores", "P123_ABCD", false},
		{"empty string", "", true},
		{"contains slash", "abc/def", true},
		{"contains dot-dot", "abc..def", true},
		{"contains question mark", "abc?def", true},
		{"contains space", "abc def", true},
		{"contains ampersand", "abc&def", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathSegment(tt.segment)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxResponseSize+1))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response body too large")
}

func TestClientRetryOn5xx(t *testing.T) {
	var attempts atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClient5xxExhaustsRetries(t *testing.T) {
	var attempts atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/test", nil)
	require.Error(t, err)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
	assert.Equal(t, int32(maxRetries+1), attempts.Load())
}

func TestSleepContextCancelledStopsTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepContext(ctx, 10*time.Second)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusFound)
	})
	mux.HandleFunc("/target", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))
	_, err := c.get(context.Background(), "/redirect", nil)

	// Should NOT follow the redirect - should return the 302 as an error.
	require.Error(t, err)
}

func TestWithBaseURL_RejectsPlainHTTP(t *testing.T) {
	c := NewClient("tok", WithBaseURL("http://evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_AllowsLocalhost(t *testing.T) {
	c := NewClient("tok", WithBaseURL("http://localhost:4010"))
	assert.Equal(t, "http://localhost:4010", c.baseURL)
}

func TestWithBaseURL_AllowsLoopback(t *testing.T) {
	c := NewClient("tok", WithBaseURL("http://127.0.0.1:4010"))
	assert.Equal(t, "http://127.0.0.1:4010", c.baseURL)
}

func TestWithBaseURL_RejectsLocalhostSubdomain(t *testing.T) {
	c := NewClient("tok", WithBaseURL("http://localhost.evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_RejectsLoopbackSubdomain(t *testing.T) {
	c := NewClient("tok", WithBaseURL("http://127.0.0.1.evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_AllowsHTTPS(t *testing.T) {
	c := NewClient("tok", WithBaseURL("https://custom.pagerduty.com"))
	assert.Equal(t, "https://custom.pagerduty.com", c.baseURL)
}

func TestRetryAfterDuration_CapsAt60Seconds(t *testing.T) {
	got := retryAfterDuration("3600", time.Second)
	assert.Equal(t, 60*time.Second, got)
}

func TestRetryAfterDuration_PassesThroughReasonableValues(t *testing.T) {
	got := retryAfterDuration("5", time.Second)
	assert.Equal(t, 5*time.Second, got)
}
