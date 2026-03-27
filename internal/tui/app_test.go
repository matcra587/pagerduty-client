package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp_DefaultFilterState(t *testing.T) {
	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)

	assert.Equal(t, "open", app.filterState.Status)
	assert.Equal(t, "all", app.filterState.Urgency)
	assert.Equal(t, "all", app.filterState.Priority)
	assert.Equal(t, "all", app.filterState.Assigned)
	assert.Equal(t, "7d", app.filterState.Age, "default age lookback should be 7d")

	since, until := ageRange(app.filterState.Age)
	assert.NotEmpty(t, since, "startup should produce a since value from default 7d age")
	assert.NotEmpty(t, until, "startup should produce an until value from default 7d age")
}

func TestIncidentsLoadedMsg_ErrorShowsFlash(t *testing.T) {
	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)
	msg := incidentsLoadedMsg{err: errors.New("API rate limited")}
	result, cmd := app.Update(msg)
	a := result.(App)
	assert.False(t, a.loading)
	assert.NotNil(t, cmd)
}

func TestAgeRangeMapping(t *testing.T) {
	tests := []struct {
		age       string
		wantSince bool
		wantUntil bool
		wantDur   time.Duration
	}{
		{"7d", true, true, 7 * 24 * time.Hour},
		{"30d", true, true, 30 * 24 * time.Hour},
		{"60d", true, true, 60 * 24 * time.Hour},
		{"90d", true, true, 90 * 24 * time.Hour},
		{"all", false, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.age, func(t *testing.T) {
			since, until := ageRange(tt.age)

			if tt.wantSince {
				require.NotEmpty(t, since, "expected since to be set for age %q", tt.age)
				parsed, err := time.Parse(time.RFC3339, since)
				require.NoError(t, err)
				expected := time.Now().Add(-tt.wantDur)
				diff := expected.Sub(parsed).Abs()
				assert.Less(t, diff, 5*time.Second, "since value should be within 5s of expected")
			} else {
				assert.Empty(t, since, "expected since to be empty for age %q", tt.age)
			}

			if tt.wantUntil {
				require.NotEmpty(t, until, "expected until to be set for age %q", tt.age)
				parsed, err := time.Parse(time.RFC3339, until)
				require.NoError(t, err)
				diff := time.Since(parsed).Abs()
				assert.Less(t, diff, 5*time.Second, "until value should be within 5s of now")
			} else {
				assert.Empty(t, until, "expected until to be empty for age %q", tt.age)
			}
		})
	}
}

func TestNewApp_HasTabBar(t *testing.T) {
	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)

	require.Len(t, app.tabs, 1)
	assert.Equal(t, "Incidents", app.tabs[0].label)
	assert.Equal(t, 0, app.activeTab)
}

func TestTabIndexFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want int
	}{
		{"1", 0},
		{"2", 1},
		{"9", 8},
		{"0", -1},
		{"a", -1},
		{"", -1},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, tabIndexFromKey(tt.key))
		})
	}
}
