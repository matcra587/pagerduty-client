package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func TestListAbilities(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities", r.URL.Path)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"abilities":["sso","teams","read_only_users"]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	abilities, err := c.ListAbilities(context.Background())
	require.NoError(t, err)
	require.Len(t, abilities, 3)
	assert.Equal(t, "sso", abilities[0].Name)
	assert.Equal(t, "SSO", abilities[0].Display)
	assert.Equal(t, "teams", abilities[1].Name)
	assert.Equal(t, "Teams", abilities[1].Display)
	assert.Equal(t, "read_only_users", abilities[2].Name)
	assert.Equal(t, "Read Only Users", abilities[2].Display)
}

func TestListAbilities_Empty(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"abilities":[]}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	abilities, err := c.ListAbilities(context.Background())
	require.NoError(t, err)
	assert.Empty(t, abilities)
}

func TestHumanise(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"title case", "service_support_hours", "Service Support Hours"},
		{"acronym SSO", "sso", "SSO"},
		{"acronym iOS", "liveness_ios_v1", "Liveness iOS v1"},
		{"acronym API", "manage_api_keys", "Manage API Keys"},
		{"multiple acronyms", "api_v2_access", "API v2 Access"},
		{"single word", "teams", "Teams"},
		{"version suffix", "liveness_android_v1", "Liveness Android v1"},
	}
	tc := cases.Title(language.English)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, humanise(tc, tt.in))
		})
	}
}

// OpenAPI defines 401, 403 and 429 as error responses for GET /abilities.

func TestListAbilities_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Unauthorized","code":2006}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("bad-token", WithBaseURL(srv.URL))
	_, err := c.ListAbilities(context.Background())
	require.Error(t, err)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestListAbilities_Forbidden(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Forbidden","code":2010}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	_, err := c.ListAbilities(context.Background())
	require.Error(t, err)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestListAbilities_RateLimited(t *testing.T) {
	t.Parallel()
	var attempt atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempt.Add(1)
		if n == 1 {
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
	require.Len(t, abilities, 1)
	assert.Equal(t, "sso", abilities[0].Name)
}

// OpenAPI defines 204, 401, 402, 403, 404 and 429 for GET /abilities/{id}.

func TestTestAbility_Available(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities/sso", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	err := c.TestAbility(context.Background(), "sso")
	require.NoError(t, err)
}

func TestTestAbility_Unavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"message":"Account does not have the abilities to perform the action","code":2012}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	err := c.TestAbility(context.Background(), "sso")
	require.ErrorIs(t, err, ErrPaymentRequired)
}

func TestTestAbility_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Not Found","code":2100}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	err := c.TestAbility(context.Background(), "nonexistent")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestTestAbility_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Unauthorized","code":2006}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("bad-token", WithBaseURL(srv.URL))
	err := c.TestAbility(context.Background(), "sso")
	require.Error(t, err)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestTestAbility_Forbidden(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Forbidden","code":2010}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("test-token", WithBaseURL(srv.URL))
	err := c.TestAbility(context.Background(), "sso")
	require.Error(t, err)

	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok)
	assert.Equal(t, 403, apiErr.StatusCode)
}
