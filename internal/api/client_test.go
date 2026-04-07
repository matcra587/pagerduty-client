package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	c := NewClient("test-token")
	assert.Equal(t, "https://api.pagerduty.com", c.baseURL)
	assert.Equal(t, "test-token", c.token)
}

func TestClientAuthHeader(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	c := NewClient("tok")
	_, err := c.get(context.Background(), "/incidents/../../admin", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid path segment")
}

func TestPutFromRejectsInvalidPath(t *testing.T) {
	t.Parallel()
	c := NewClient("tok")
	_, err := c.putFrom(context.Background(), "/incidents/../../admin", nil, "user@example.com")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid path segment")
}

func TestPostFromRejectsInvalidPath(t *testing.T) {
	t.Parallel()
	c := NewClient("tok")
	_, err := c.postFrom(context.Background(), "/incidents/../../admin", nil, "user@example.com")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid path segment")
}

func TestValidatePathRejectsQueryString(t *testing.T) {
	t.Parallel()
	err := validatePath("/incidents?foo=bar")
	require.Error(t, err)
	assert.ErrorContains(t, err, "query string")
}

func TestValidatePathSegment(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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
	assert.ErrorContains(t, err, "response body too large")
}

// roundTripFunc adapts a function into an http.RoundTripper for use
// with synctest, where real network I/O is not permitted.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// bodyReader wraps a string as a ReadCloser for mock HTTP responses.
func bodyReader(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func TestClientRetryOn5xx(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			n := attempts.Add(1)
			if n < 3 {
				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: bodyReader(`{}`)}, nil
		})

		c := NewClient("test-token", WithHTTPClient(&http.Client{Transport: transport}))
		c.baseURL = "https://mock"

		done := make(chan error, 1)
		go func() {
			_, err := c.get(context.Background(), "/test", nil)
			done <- err
		}()

		// Advance past jittered backoff sleeps (up to 2s + 4s).
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		time.Sleep(4 * time.Second)
		synctest.Wait()

		require.NoError(t, <-done)
		assert.Equal(t, int32(3), attempts.Load())
	})
}

func TestClient5xxExhaustsRetries(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts.Add(1)
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody}, nil
		})

		c := NewClient("test-token", WithHTTPClient(&http.Client{Transport: transport}))
		c.baseURL = "https://mock"

		done := make(chan error, 1)
		go func() {
			_, err := c.get(context.Background(), "/test", nil)
			done <- err
		}()

		// Advance past all jittered backoff sleeps (up to 2s + 4s + 8s).
		for range 3 {
			synctest.Wait()
			time.Sleep(8 * time.Second)
		}
		synctest.Wait()

		err := <-done
		require.Error(t, err)

		apiErr, ok := errors.AsType[*APIError](err)
		require.True(t, ok)
		assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
		assert.Equal(t, int32(maxRetries+1), attempts.Load())
	})
}

func TestSleepContextCancelledStopsTimer(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepContext(ctx, 10*time.Second)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	c := NewClient("tok", WithBaseURL("http://evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_AllowsLocalhost(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", WithBaseURL("http://localhost:4010"))
	assert.Equal(t, "http://localhost:4010", c.baseURL)
}

func TestWithBaseURL_AllowsLoopback(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", WithBaseURL("http://127.0.0.1:4010"))
	assert.Equal(t, "http://127.0.0.1:4010", c.baseURL)
}

func TestWithBaseURL_RejectsLocalhostSubdomain(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", WithBaseURL("http://localhost.evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_RejectsLoopbackSubdomain(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", WithBaseURL("http://127.0.0.1.evil.com"))
	assert.Equal(t, defaultBaseURL, c.baseURL)
}

func TestWithBaseURL_AllowsHTTPS(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", WithBaseURL("https://custom.pagerduty.com"))
	assert.Equal(t, "https://custom.pagerduty.com", c.baseURL)
}

func TestRetryAfterDuration_CapsAt60Seconds(t *testing.T) {
	t.Parallel()
	got := retryAfterDuration("3600", time.Second)
	assert.Equal(t, 60*time.Second, got)
}

func TestRetryAfterDuration_PassesThroughReasonableValues(t *testing.T) {
	t.Parallel()
	got := retryAfterDuration("5", time.Second)
	assert.Equal(t, 5*time.Second, got)
}

func TestBackoffJitter_SleepWithinBounds(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Verify jittered backoff completes within bounded time using
		// virtualised time. Backoff doubles before sleep: first retry
		// sleeps rand.N(2s), second sleeps rand.N(4s).
		var attempts atomic.Int32
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			n := attempts.Add(1)
			if n < 3 {
				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: bodyReader(`{}`)}, nil
		})

		c := NewClient("test-token", WithHTTPClient(&http.Client{Transport: transport}))
		c.baseURL = "https://mock"

		done := make(chan error, 1)
		go func() {
			_, err := c.get(context.Background(), "/test", nil)
			done <- err
		}()

		// Advance past jittered backoff sleeps.
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		time.Sleep(4 * time.Second)
		synctest.Wait()

		require.NoError(t, <-done)
		assert.Equal(t, int32(3), attempts.Load())
	})
}
