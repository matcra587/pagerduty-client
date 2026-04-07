package compact

import (
	"encoding/json"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompactComparison runs the fixture incidents through all three
// output modes and prints a comparison table. Run with -v to see it.
func TestCompactComparison(t *testing.T) {
	t.Parallel()
	incidents := testutil.LoadIncidents(t)
	require.NotEmpty(t, incidents)

	items := len(incidents)

	// Raw: no filtering at all.
	rawBytes, err := json.Marshal(incidents)
	require.NoError(t, err)

	// Layer 1 only: compact (deny list + flatten + strip empties).
	compacted := Compact(incidents)
	compactBytes, err := json.Marshal(compacted)
	require.NoError(t, err)

	// Layer 1 + 2: compact then budget select.
	rw, ok := WeightsForResource(ResourceIncident)
	require.True(t, ok)
	budgeted := BudgetSelect(compacted, rw)
	budgetBytes, err := json.Marshal(budgeted)
	require.NoError(t, err)

	rawTok := len(rawBytes) / 4
	compactTok := len(compactBytes) / 4
	budgetTok := len(budgetBytes) / 4

	t.Log("")
	t.Logf("%-25s %8s %8s %8s %10s", "Mode", "Chars", "Tokens", "Per Item", "Reduction")
	t.Log("--------------------------------------------------------------")
	t.Logf("%-25s %8d %8d %8d %10s",
		"Raw (no filtering)", len(rawBytes), rawTok, rawTok/items, "-")
	t.Logf("%-25s %8d %8d %8d %9.0f%%",
		"Compact (layer 1)", len(compactBytes), compactTok, compactTok/items,
		(1-float64(compactTok)/float64(rawTok))*100)
	t.Logf("%-25s %8d %8d %8d %9.0f%%",
		"Budget (layer 1+2)", len(budgetBytes), budgetTok, budgetTok/items,
		(1-float64(budgetTok)/float64(rawTok))*100)

	// Show fields kept by budget pipeline.
	if arr, ok := budgeted.([]any); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			t.Logf("\nBudget fields kept: %v", keys)
		}
	}

	// Budget output must be smaller than raw.
	assert.Less(t, len(budgetBytes), len(rawBytes),
		"budget pipeline should produce smaller output than raw")

	// Budget output must be smaller than compact-only.
	assert.Less(t, len(budgetBytes), len(compactBytes),
		"budget pipeline should produce smaller output than compact-only")
}
