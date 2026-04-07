package compact

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetSelect_MustHaveFieldsAlwaysKept(t *testing.T) {
	t.Parallel()

	w := ResourceWeights{
		Budget:        10, // tight budget
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":     1.0,
			"status": 1.0,
			"notes":  0.3,
		},
	}

	input := map[string]any{
		"id":     "P123",
		"status": "triggered",
		"notes":  "this is a long note that should be dropped due to budget constraints",
	}

	result := BudgetSelect(input, w)

	m, ok := result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P123", m["id"])
	assert.Equal(t, "triggered", m["status"])
	assert.NotContains(t, m, "notes")
}

func TestBudgetSelect_FillsByScore(t *testing.T) {
	t.Parallel()

	// Give enough budget for one extra field beyond must-haves.
	// "priority" is small and high importance (high score).
	// "description" is large and low importance (low score).
	// The selector should pick "priority" over "description".
	w := ResourceWeights{
		Budget:        20,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":          1.0,
			"priority":    0.9,
			"description": 0.3,
		},
	}

	input := map[string]any{
		"id":          "P1",
		"priority":    "high",
		"description": "a]very long description field that consumes many tokens and has low importance",
	}

	result := BudgetSelect(input, w)

	m, ok := result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P1", m["id"])
	assert.Equal(t, "high", m["priority"])
	assert.NotContains(t, m, "description")
}

func TestBudgetSelect_ArrayOfItems(t *testing.T) {
	t.Parallel()

	w := ResourceWeights{
		Budget:        10,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":   1.0,
			"name": 0.3,
		},
	}

	input := []any{
		map[string]any{"id": "1", "name": "this name is too long for the budget"},
		map[string]any{"id": "2", "name": "another long name exceeding budget"},
	}

	result := BudgetSelect(input, w)

	arr, ok := result.([]any)
	require.True(t, ok)
	require.Len(t, arr, 2)

	for _, elem := range arr {
		m, ok := elem.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, m, "id")
		assert.NotContains(t, m, "name")
	}
}

func TestBudgetSelect_NonMapPassthrough(t *testing.T) {
	t.Parallel()

	w := ResourceWeights{Budget: 100, DefaultWeight: 0.5}

	assert.Equal(t, "hello", BudgetSelect("hello", w))
	assert.InDelta(t, 42.0, BudgetSelect(42.0, w), 1e-9)
	assert.Equal(t, true, BudgetSelect(true, w))
	assert.Nil(t, BudgetSelect(nil, w))
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		value     any
		minTokens int
	}{
		{
			name:      "short string",
			key:       "id",
			value:     "P1",
			minTokens: 1,
		},
		{
			name:      "longer string",
			key:       "description",
			value:     "a]long description that spans many characters and should produce more tokens",
			minTokens: 5,
		},
		{
			name:      "number",
			key:       "count",
			value:     42,
			minTokens: 1,
		},
		{
			name:      "nested object",
			key:       "service",
			value:     map[string]any{"id": "SVC1", "name": "My Service"},
			minTokens: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokens := estimateTokens(tc.key, tc.value)
			assert.GreaterOrEqual(t, tokens, tc.minTokens)
			assert.GreaterOrEqual(t, tokens, 1, "minimum token count is 1")
		})
	}
}

func TestBudgetSelect_SkipsOversizedField(t *testing.T) {
	t.Parallel()

	// Budget is tight. Must-have "id" takes some budget.
	// "big" is oversised but comes first alphabetically.
	// "small" is small enough to fit. The selector should skip
	// "big" and still include "small".
	w := ResourceWeights{
		Budget:        15,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":    1.0,
			"big":   0.8,
			"small": 0.5,
		},
	}

	input := map[string]any{
		"id":    "P1",
		"big":   "this is a very large field value that will exceed the remaining token budget after must-haves are included",
		"small": "ok",
	}

	result := BudgetSelect(input, w)

	m, ok := result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "P1", m["id"])
	assert.Contains(t, m, "small")
	assert.NotContains(t, m, "big")
}
