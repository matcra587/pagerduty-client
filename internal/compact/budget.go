package compact

import (
	"encoding/json"
	"math"
	"sort"
)

// fieldCandidate holds a field's metadata for greedy knapsack selection.
type fieldCandidate struct {
	key        string
	value      any
	importance float64
	tokens     int
	score      float64
}

// BudgetSelect applies token-budget field selection to data.
// Maps are filtered via selectFields, slices have each element
// selected independently, and all other types pass through unchanged.
func BudgetSelect(data any, w ResourceWeights) any {
	switch v := data.(type) {
	case map[string]any:
		return selectFields(v, w)
	case []any:
		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = BudgetSelect(elem, w)
		}
		return out
	default:
		return data
	}
}

// selectFields performs two-pass greedy knapsack selection on a map.
// Pass 1 force-includes must-have fields (weight >= 1.0). Pass 2
// fills remaining budget by score (importance / tokens) descending.
func selectFields(m map[string]any, w ResourceWeights) map[string]any {
	budget := w.Budget
	out := make(map[string]any, len(m))

	// Pass 1: force-include must-have fields.
	var candidates []fieldCandidate

	for k, v := range m {
		importance := w.ForField(k)
		tokens := estimateTokens(k, v)

		if importance >= 1.0 {
			out[k] = v
			budget -= tokens

			continue
		}

		if importance <= 0 {
			continue
		}

		candidates = append(candidates, fieldCandidate{
			key:        k,
			value:      v,
			importance: importance,
			tokens:     tokens,
			score:      importance / float64(tokens),
		})
	}

	// Pass 2: sort candidates by score descending, tiebreak by
	// importance descending.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}

		return candidates[i].importance > candidates[j].importance
	})

	// Fill greedily, skipping fields that exceed remaining budget.
	for _, c := range candidates {
		if c.tokens > budget {
			continue
		}

		out[c.key] = c.value
		budget -= c.tokens
	}

	return out
}

// estimateTokens approximates the token cost of a single key-value
// pair by marshalling it to JSON and dividing byte length by 4.
func estimateTokens(key string, value any) int {
	b, err := json.Marshal(map[string]any{key: value})
	if err != nil {
		return 1
	}

	// Subtract 2 for the outer {} wrapper.
	content := len(b) - 2
	if content <= 0 {
		return 1
	}

	return int(math.Ceil(float64(content) / 4))
}
