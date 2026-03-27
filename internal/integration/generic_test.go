package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneric_NoMatch_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := Generic{}
	env := UnwrapAlert(map[string]any{"random": true})
	_, ok := g.Normalise(env)
	assert.False(t, ok)
}

func TestGeneric_FlattensScalars(t *testing.T) {
	t.Parallel()
	env := UnwrapAlert(map[string]any{
		"details": map[string]any{
			"custom_details": map[string]any{
				"title":  "CPU high",
				"query":  "avg:cpu > 90",
				"nested": map[string]any{"deep": "value"},
			},
		},
	})
	g := Generic{}
	s, _ := g.Normalise(env)

	assert.Equal(t, "Unknown", s.Source)
	require.GreaterOrEqual(t, len(s.Fields), 2)

	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "CPU high", fieldMap["title"])
	assert.Equal(t, "avg:cpu > 90", fieldMap["query"])
	assert.Contains(t, fieldMap["nested"], "deep")
}
