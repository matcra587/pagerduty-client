package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterOptions_State_ReturnsDefaults(t *testing.T) {
	fo := NewFilterOptions()
	state := fo.State()

	assert.Equal(t, "open", state.Status)
	assert.Equal(t, "all", state.Urgency)
	assert.Equal(t, "all", state.Priority)
	assert.Equal(t, "all", state.Assigned)
	assert.Equal(t, "7d", state.Age, "default age should be 7d")
}
