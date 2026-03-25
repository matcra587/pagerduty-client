//go:build integration

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockBaseURL = "https://stoplight.io/mocks/pagerduty-upgrade/api-schema/2748099"
	mockToken   = "y_NbAkKc66ryYTWUXYEu"
)

func newMockClient(t *testing.T) *Client {
	t.Helper()

	// Skip if mock is unreachable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mockBaseURL+"/users/me", nil)
	require.NoError(t, err)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Prefer", "code=200, dynamic=true")
	req.Header.Set("Authorization", "Token token="+mockToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("stoplight mock unreachable: ", err)
	}
	resp.Body.Close()

	return NewClient(mockToken,
		WithBaseURL(mockBaseURL),
		WithExtraHeaders(map[string]string{
			"Accept": "application/json",
			"Prefer": "code=200, dynamic=true",
		}),
	)
}

// fetchListPage fetches a single page from a list endpoint and returns
// the raw JSON items under the given key. The Stoplight mock returns
// random dynamic data including negative numbers for uint fields, so we
// decode into []map[string]any to avoid strict type errors.
func fetchListPage(t *testing.T, c *Client, path, key string) []map[string]any {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	params := url.Values{}
	params.Set("limit", "5")
	params.Set("offset", "0")

	body, err := c.get(ctx, path, params)
	require.NoError(t, err, "GET %s failed", path)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &raw), "decoding response envelope")

	rawItems, ok := raw[key]
	require.True(t, ok, "response missing %q key", key)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(rawItems, &items), "decoding %s items", key)

	return items
}

func TestIntegration_ListIncidents(t *testing.T) {
	c := newMockClient(t)

	incidents := fetchListPage(t, c, "/incidents", "incidents")
	require.NotEmpty(t, incidents, "expected at least one incident")
	assert.NotEmpty(t, incidents[0]["id"], "first incident should have a non-empty id")
}

func TestIntegration_ListServices(t *testing.T) {
	c := newMockClient(t)

	services := fetchListPage(t, c, "/services", "services")
	require.NotEmpty(t, services, "expected at least one service")
	assert.NotEmpty(t, services[0]["name"], "first service should have a non-empty name")
}

func TestIntegration_ListTeams(t *testing.T) {
	c := newMockClient(t)

	teams := fetchListPage(t, c, "/teams", "teams")
	require.NotEmpty(t, teams, "expected at least one team")
}

func TestIntegration_ListUsers(t *testing.T) {
	c := newMockClient(t)

	users := fetchListPage(t, c, "/users", "users")
	require.NotEmpty(t, users, "expected at least one user")
}

func TestIntegration_GetCurrentUser(t *testing.T) {
	c := newMockClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// GetCurrentUser decodes into pagerduty.User which has no uint fields
	// that would clash with random negative mock data.
	user, err := c.GetCurrentUser(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, user.Email, "current user should have a non-empty Email")
}

func TestIntegration_ListOnCalls(t *testing.T) {
	c := newMockClient(t)

	oncalls := fetchListPage(t, c, "/oncalls", "oncalls")
	require.NotEmpty(t, oncalls, "expected at least one on-call entry")

	userRaw, ok := oncalls[0]["user"]
	require.True(t, ok, "first on-call entry should have a user field")
	user, ok := userRaw.(map[string]any)
	require.True(t, ok, "user field should be an object")
	assert.NotEmpty(t, user["id"], "first on-call entry should have a non-empty user.id")
}

func TestIntegration_ListSchedules(t *testing.T) {
	c := newMockClient(t)

	schedules := fetchListPage(t, c, "/schedules", "schedules")
	require.NotEmpty(t, schedules, "expected at least one schedule")
}
