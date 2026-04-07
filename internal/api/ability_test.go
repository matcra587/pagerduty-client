package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func TestListAbilities(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities", r.URL.Path)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"abilities":["sso","teams","read_only_users"]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	abilities, err := client.ListAbilities(context.Background())
	require.NoError(t, err)
	require.Len(t, abilities, 3)
	assert.Equal(t, "read_only_users", abilities[0].Name)
	assert.Equal(t, "Read Only Users", abilities[0].Display)
	assert.Equal(t, "sso", abilities[1].Name)
	assert.Equal(t, "SSO", abilities[1].Display)
	assert.Equal(t, "teams", abilities[2].Name)
	assert.Equal(t, "Teams", abilities[2].Display)
}

func TestListAbilities_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"abilities":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	abilities, err := client.ListAbilities(context.Background())
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tc := cases.Title(language.English)
			assert.Equal(t, tt.want, humanise(tc, tt.in))
		})
	}
}

func TestTestAbility_Available(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/abilities/sso", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	err := client.TestAbility(context.Background(), "sso")
	require.NoError(t, err)
}

func TestTestAbility_Unavailable(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"message":"Account does not have the abilities to perform the action","code":2012}}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	err := client.TestAbility(context.Background(), "sso")
	require.ErrorIs(t, err, ErrPaymentRequired)
}

func TestTestAbility_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Not Found","code":2100}}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	err := client.TestAbility(context.Background(), "nonexistent")
	require.ErrorIs(t, err, ErrNotFound)
}
