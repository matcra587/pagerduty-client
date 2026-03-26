package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAbilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities", r.URL.Path)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"abilities":["sso","teams","read_only_users"]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	abilities, err := c.ListAbilities(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"sso", "teams", "read_only_users"}, abilities)
}

func TestListAbilities_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"abilities":[]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	abilities, err := c.ListAbilities(context.Background())
	require.NoError(t, err)
	assert.Empty(t, abilities)
}

// OpenAPI defines 401, 403 and 429 as error responses for GET /abilities.

func TestListAbilities_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Unauthorized","code":2006}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("bad-token", WithBaseURL(srv.URL))
	_, err := c.ListAbilities(context.Background())
	require.Error(t, err)

	apiErr := &APIError{}
	ok := errors.As(err, &apiErr)
	require.True(t, ok)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestListAbilities_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Forbidden","code":2010}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	_, err := c.ListAbilities(context.Background())
	require.Error(t, err)

	apiErr := &APIError{}
	ok := errors.As(err, &apiErr)
	require.True(t, ok)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestListAbilities_RateLimited(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt++
		if attempt == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Rate Limit Exceeded","code":2020}}`))
			return
		}
		_, _ = w.Write([]byte(`{"abilities":["sso"]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	abilities, err := c.ListAbilities(context.Background())
	require.NoError(t, err, "should succeed after retry")
	assert.Equal(t, []string{"sso"}, abilities)
}
