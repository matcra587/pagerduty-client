package tui

import (
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTabReadCache_ServicesHitBeforeTTL(t *testing.T) {
	start := time.Date(2026, time.January, 2, 15, 0, 0, 0, time.UTC)
	now := start
	cache := newTabReadCache(5*time.Minute, func() time.Time { return now })

	want := []pagerduty.Service{
		{APIObject: pagerduty.APIObject{ID: "SVC1", Summary: "Auth"}},
		{APIObject: pagerduty.APIObject{ID: "SVC2", Summary: "Billing"}},
	}

	cache.PutServices(want)

	now = start.Add(4 * time.Minute)
	got, ok := cache.Services()

	require.True(t, ok)
	assert.Equal(t, want, got)
}

func TestTabReadCache_ExpiresAfterTTL(t *testing.T) {
	start := time.Date(2026, time.January, 2, 15, 0, 0, 0, time.UTC)
	now := start
	cache := newTabReadCache(5*time.Minute, func() time.Time { return now })

	cache.PutEscalationPolicies([]pagerduty.EscalationPolicy{
		{APIObject: pagerduty.APIObject{ID: "EP1", Summary: "Default"}},
	})

	now = start.Add(6 * time.Minute)
	got, ok := cache.EscalationPolicies()

	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestTabReadCache_InvalidateTeamScopedData(t *testing.T) {
	now := time.Date(2026, time.January, 2, 15, 0, 0, 0, time.UTC)
	cache := newTabReadCache(30*time.Minute, func() time.Time { return now })

	cache.PutServices([]pagerduty.Service{
		{APIObject: pagerduty.APIObject{ID: "SVC1", Summary: "Auth"}},
	})
	cache.PutEscalationPolicies([]pagerduty.EscalationPolicy{
		{APIObject: pagerduty.APIObject{ID: "EP1", Summary: "Default"}},
	})

	cache.InvalidateTeamScopedData()

	services, ok := cache.Services()
	assert.False(t, ok)
	assert.Nil(t, services)

	policies, ok := cache.EscalationPolicies()
	assert.False(t, ok)
	assert.Nil(t, policies)
}
