package tui

import (
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

type cachedSlice[T any] struct {
	at    time.Time
	value []T
	ok    bool
}

type tabReadCache struct {
	ttl time.Duration
	now func() time.Time

	services           cachedSlice[pagerduty.Service]
	escalationPolicies cachedSlice[pagerduty.EscalationPolicy]
}

func newTabReadCache(ttl time.Duration, now func() time.Time) *tabReadCache {
	if now == nil {
		now = time.Now
	}

	return &tabReadCache{
		ttl: ttl,
		now: now,
	}
}

func readCachedSlice[T any](entry *cachedSlice[T], now func() time.Time, ttl time.Duration) ([]T, bool) {
	if !entry.ok {
		return nil, false
	}

	if now == nil {
		now = time.Now
	}

	if now().Sub(entry.at) >= ttl {
		*entry = cachedSlice[T]{}
		return nil, false
	}

	return entry.value, true
}

func writeCachedSlice[T any](now func() time.Time, value []T) cachedSlice[T] {
	if now == nil {
		now = time.Now
	}

	return cachedSlice[T]{
		at:    now(),
		value: append([]T(nil), value...),
		ok:    true,
	}
}

func (c *tabReadCache) Services() ([]pagerduty.Service, bool) {
	return readCachedSlice(&c.services, c.now, c.ttl)
}

func (c *tabReadCache) PutServices(value []pagerduty.Service) {
	c.services = writeCachedSlice(c.now, value)
}

func (c *tabReadCache) EscalationPolicies() ([]pagerduty.EscalationPolicy, bool) {
	return readCachedSlice(&c.escalationPolicies, c.now, c.ttl)
}

func (c *tabReadCache) PutEscalationPolicies(value []pagerduty.EscalationPolicy) {
	c.escalationPolicies = writeCachedSlice(c.now, value)
}

func (c *tabReadCache) InvalidateServices() {
	c.services = cachedSlice[pagerduty.Service]{}
}

func (c *tabReadCache) InvalidateEscalationPolicies() {
	c.escalationPolicies = cachedSlice[pagerduty.EscalationPolicy]{}
}

func (c *tabReadCache) InvalidateTeamScopedData() {
	c.InvalidateServices()
	c.InvalidateEscalationPolicies()
}
