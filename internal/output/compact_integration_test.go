//go:build integration

package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockBaseURL = "https://stoplight.io/mocks/pagerduty-upgrade/api-schema/2748099"
	mockToken   = "y_NbAkKc66ryYTWUXYEu"
)

// fetchMockItems fetches a single page from the Stoplight mock and
// returns the items as []map[string]any. Decodes raw JSON to avoid
// the mock's negative-number-in-uint issue.
func fetchMockItems(t *testing.T, path, key string) []map[string]any {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	reqURL := mockBaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	require.NoError(t, err)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token token="+mockToken)
	req.Header.Set("Prefer", "code=200, dynamic=true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("stoplight mock unreachable: ", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if resp.StatusCode != http.StatusOK {
		t.Skipf("mock returned %d (Stoplight free tier is flaky with dynamic=true)", resp.StatusCode)
	}

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &envelope))

	rawItems, ok := envelope[key]
	if !ok {
		t.Skipf("response missing %q key", key)
	}

	var items []map[string]any
	require.NoError(t, json.Unmarshal(rawItems, &items))

	return items
}

// TestCompactComparison_Integration hits the Stoplight mock API and
// compares raw, compact and budget output sizes across multiple
// PagerDuty resource types.
func TestCompactComparison_Integration(t *testing.T) {
	// Close idle HTTP/2 connections after test to avoid goroutine leak.
	t.Cleanup(func() { http.DefaultTransport.(*http.Transport).CloseIdleConnections() })
	resources := []struct {
		name     string
		path     string
		key      string
		resource Resource
	}{
		{"Incidents", "/incidents", "incidents", ResourceIncident},
		{"Services", "/services", "services", ResourceService},
		{"Users", "/users", "users", ResourceUser},
		{"Teams", "/teams", "teams", ResourceTeam},
	}

	for _, rc := range resources {
		t.Run(rc.name, func(t *testing.T) {
			items := fetchMockItems(t, rc.path, rc.key)
			if len(items) == 0 {
				t.Skip("no items returned")
			}

			count := len(items)

			// Raw: no filtering.
			rawBytes, err := json.Marshal(items)
			require.NoError(t, err)

			// Layer 1: compact only.
			compacted := Compact(items)
			compactBytes, err := json.Marshal(compacted)
			require.NoError(t, err)

			// Layer 1+2: compact + budget (if weights exist).
			rw, hasWeights := WeightsForResource(rc.resource)
			var budgetBytes []byte
			if hasWeights {
				budgeted := budgetSelect(compacted, rw)
				budgetBytes, err = json.Marshal(budgeted)
				require.NoError(t, err)
			}

			rawTok := len(rawBytes) / 4
			compactTok := len(compactBytes) / 4

			t.Log("")
			t.Log(fmt.Sprintf("%-25s %8s %8s %8s %10s",
				"Mode", "Chars", "Tokens", "Per Item", "Reduction"))
			t.Log("--------------------------------------------------------------")
			t.Log(fmt.Sprintf("%-25s %8d %8d %8d %10s",
				"Raw", len(rawBytes), rawTok, rawTok/count, "-"))
			t.Log(fmt.Sprintf("%-25s %8d %8d %8d %9.0f%%",
				"Compact (layer 1)", len(compactBytes), compactTok, compactTok/count,
				(1-float64(compactTok)/float64(rawTok))*100))

			if hasWeights {
				budgetTok := len(budgetBytes) / 4
				t.Log(fmt.Sprintf("%-25s %8d %8d %8d %9.0f%%",
					"Budget (layer 1+2)", len(budgetBytes), budgetTok, budgetTok/count,
					(1-float64(budgetTok)/float64(rawTok))*100))
				assert.Less(t, len(budgetBytes), len(rawBytes),
					"budget output should be smaller than raw")
			}

			assert.Less(t, len(compactBytes), len(rawBytes),
				"compact output should be smaller than raw")
		})
	}
}
