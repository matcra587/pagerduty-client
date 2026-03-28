package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeightsForResource_Incident(t *testing.T) {
	t.Parallel()

	w, ok := WeightsForResource(ResourceIncident)
	require.True(t, ok)

	assert.Equal(t, 250, w.Budget)
	assert.InDelta(t, 0.1, w.DefaultWeight, 1e-9)
	assert.InDelta(t, 1.0, w.Fields["id"], 1e-9)
	assert.InDelta(t, 1.0, w.Fields["title"], 1e-9)
	assert.InDelta(t, 1.0, w.Fields["status"], 1e-9)
	assert.InDelta(t, 1.0, w.Fields["urgency"], 1e-9)
	assert.InDelta(t, 1.0, w.Fields["priority"], 1e-9)
	assert.InDelta(t, 0.9, w.Fields["service"], 1e-9)
	assert.InDelta(t, 0.9, w.Fields["assignments"], 1e-9)
	assert.InDelta(t, 0.8, w.Fields["incident_number"], 1e-9)
	assert.InDelta(t, 0.8, w.Fields["created_at"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["escalation_policy"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["teams"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["acknowledgements"], 1e-9)
	assert.InDelta(t, 0.6, w.Fields["alert_counts"], 1e-9)
	assert.InDelta(t, 0.5, w.Fields["last_status_change_at"], 1e-9)
	assert.InDelta(t, 0.4, w.Fields["html_url"], 1e-9)
	assert.InDelta(t, 0.3, w.Fields["description"], 1e-9)
	assert.InDelta(t, 0.3, w.Fields["incident_key"], 1e-9)
}

func TestWeightsForResource_Alert(t *testing.T) {
	t.Parallel()

	w, ok := WeightsForResource(ResourceAlert)
	require.True(t, ok)

	assert.Equal(t, 150, w.Budget)
	assert.InDelta(t, 0.1, w.DefaultWeight, 1e-9)
	assert.InDelta(t, 1.0, w.Fields["id"], 1e-9)
	assert.InDelta(t, 1.0, w.Fields["status"], 1e-9)
	assert.InDelta(t, 0.9, w.Fields["severity"], 1e-9)
	assert.InDelta(t, 0.8, w.Fields["service"], 1e-9)
	assert.InDelta(t, 0.8, w.Fields["incident"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["body"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["summary"], 1e-9)
	assert.InDelta(t, 0.7, w.Fields["created_at"], 1e-9)
	assert.InDelta(t, 0.6, w.Fields["suppressed"], 1e-9)
	assert.InDelta(t, 0.5, w.Fields["alert_key"], 1e-9)
	assert.InDelta(t, 0.4, w.Fields["integration"], 1e-9)
	assert.InDelta(t, 0.3, w.Fields["html_url"], 1e-9)
}

func TestWeightsForResource_None(t *testing.T) {
	t.Parallel()

	_, ok := WeightsForResource(ResourceNone)
	assert.False(t, ok)
}

func TestWeightsForResource_Unknown(t *testing.T) {
	t.Parallel()

	_, ok := WeightsForResource(Resource("unknown"))
	assert.False(t, ok)
}

func TestResourceWeights_ForField(t *testing.T) {
	t.Parallel()

	w, ok := WeightsForResource(ResourceIncident)
	require.True(t, ok)

	assert.InDelta(t, 1.0, w.ForField("id"), 1e-9)
	assert.InDelta(t, 0.9, w.ForField("service"), 1e-9)
	assert.InDelta(t, 0.1, w.ForField("nonexistent_field"), 1e-9)
}

func TestBudgetOverride(t *testing.T) {
	t.Run("unset returns zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, budgetOverride())
	})

	t.Run("valid value", func(t *testing.T) {
		t.Setenv("PDC_AGENT_BUDGET", "500")
		assert.Equal(t, 500, budgetOverride())
	})

	t.Run("invalid value returns zero", func(t *testing.T) {
		t.Setenv("PDC_AGENT_BUDGET", "not-a-number")
		assert.Equal(t, 0, budgetOverride())
	})

	t.Run("negative value returns zero", func(t *testing.T) {
		t.Setenv("PDC_AGENT_BUDGET", "-10")
		assert.Equal(t, 0, budgetOverride())
	})
}
