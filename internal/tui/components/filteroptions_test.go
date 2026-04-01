package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterOptions_State_ReturnsDefaults(t *testing.T) {
	fo := NewFilterOptions()
	fo = fo.ShowWithRows("incidents", IncidentFilterRows())
	state := fo.State()

	assert.Equal(t, "open", state.Status)
	assert.Equal(t, "all", state.Urgency)
	assert.Equal(t, "all", state.Priority)
	assert.Equal(t, "all", state.Assigned)
	assert.Equal(t, "7d", state.Age, "default age should be 7d")
}

func TestFilterOptions_ShowWithRows_SetsOrigin(t *testing.T) {
	fo := NewFilterOptions()
	fo = fo.ShowWithRows("services", ServiceFilterRows())

	assert.True(t, fo.Visible)
	assert.Equal(t, "services", fo.Origin())
	assert.Len(t, fo.Selections(), 1)
	assert.Equal(t, "all", fo.Selections()["Status"])
}

func TestFilterOptions_Selections_ReturnsCurrentValues(t *testing.T) {
	rows := []FilterRow{
		{Label: "Status", Choices: []string{"all", "active", "disabled"}, Current: 1},
		{Label: "Urgency", Choices: []string{"high", "low", "all"}, Current: 0},
	}
	fo := NewFilterOptions()
	fo = fo.ShowWithRows("test", rows)

	sel := fo.Selections()
	assert.Equal(t, "active", sel["Status"])
	assert.Equal(t, "high", sel["Urgency"])
}
