package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditFields_Diff_NoChanges(t *testing.T) {
	t.Parallel()
	orig := editFields{title: "Server down", urgency: "high", priority: "PRIO1"}
	current := editFields{title: "Server down", urgency: "high", priority: "PRIO1"}

	d := orig.diff(current)
	assert.True(t, d.IsEmpty())
	assert.Nil(t, d.Title)
	assert.Nil(t, d.Urgency)
	assert.Nil(t, d.Priority)
}

func TestEditFields_Diff_SingleChange(t *testing.T) {
	t.Parallel()
	orig := editFields{title: "Server down", urgency: "high", priority: "PRIO1"}
	current := editFields{title: "Server down", urgency: "low", priority: "PRIO1"}

	d := orig.diff(current)
	assert.False(t, d.IsEmpty())
	assert.Nil(t, d.Title)
	require.NotNil(t, d.Urgency)
	assert.Equal(t, "low", *d.Urgency)
	assert.Nil(t, d.Priority)
}

func TestEditFields_Diff_AllChanged(t *testing.T) {
	t.Parallel()
	orig := editFields{title: "Server down", urgency: "high", priority: "PRIO1"}
	current := editFields{title: "Memory leak", urgency: "low", priority: "PRIO2"}

	d := orig.diff(current)
	assert.False(t, d.IsEmpty())
	require.NotNil(t, d.Title)
	assert.Equal(t, "Memory leak", *d.Title)
	require.NotNil(t, d.Urgency)
	assert.Equal(t, "low", *d.Urgency)
	require.NotNil(t, d.Priority)
	assert.Equal(t, "PRIO2", *d.Priority)
}

func TestEditFields_Diff_PriorityClear(t *testing.T) {
	t.Parallel()
	orig := editFields{title: "Server down", urgency: "high", priority: "PRIO1"}
	current := editFields{title: "Server down", urgency: "high", priority: ""}

	d := orig.diff(current)
	assert.False(t, d.IsEmpty())
	assert.Nil(t, d.Title)
	assert.Nil(t, d.Urgency)
	require.NotNil(t, d.Priority)
	assert.Empty(t, *d.Priority)
}
