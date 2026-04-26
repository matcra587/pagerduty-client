package tui

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/tui/components"
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

	require.Len(t, app.tabs, 4)
	assert.Equal(t, "Incidents", app.tabs[0].label)
	assert.Equal(t, "Escalation Policies", app.tabs[1].label)
	assert.Equal(t, "Services", app.tabs[2].label)
	assert.Equal(t, "Teams", app.tabs[3].label)
	assert.Equal(t, 0, app.activeTab)
}

func TestAppFetchServicesCmd_UsesCacheHit(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	var hits atomic.Int32
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{
			"services": [{"id":"PSVC-FRESH","name":"Fresh Service","status":"active"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		config.Default(),
		"test@example.com",
	)
	app.tabCache.PutServices([]pagerduty.Service{
		{APIObject: pagerduty.APIObject{ID: "PSVC-CACHED"}, Name: "Cached Service", Status: "disabled"},
	})

	msg := app.fetchServicesCmd()()
	require.Zero(t, hits.Load())

	loaded, ok := msg.(servicesLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.services, 1)
	assert.Equal(t, "PSVC-CACHED", loaded.services[0].ID)
	assert.Equal(t, "Cached Service", loaded.services[0].Name)
}

func TestAppFetchEPCmd_UsesCacheHit(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	var hits atomic.Int32
	mux.HandleFunc("/escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{
			"escalation_policies": [{"id":"PEP-FRESH","name":"Fresh EP"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		config.Default(),
		"test@example.com",
	)
	app.tabCache.PutEscalationPolicies([]pagerduty.EscalationPolicy{
		{APIObject: pagerduty.APIObject{ID: "PEP-CACHED"}, Name: "Cached EP"},
	})

	msg := app.fetchEPCmd()()
	require.Zero(t, hits.Load())

	loaded, ok := msg.(epLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.policies, 1)
	assert.Equal(t, "PEP-CACHED", loaded.policies[0].ID)
	assert.Equal(t, "Cached EP", loaded.policies[0].Name)
}

func TestAppResetTeamScopedTabState_ClearsCacheAndLoadedFlags(t *testing.T) {
	t.Parallel()

	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)
	app.tabCache.PutServices([]pagerduty.Service{{APIObject: pagerduty.APIObject{ID: "PSVC1"}, Name: "Cached Service"}})
	app.tabCache.PutEscalationPolicies([]pagerduty.EscalationPolicy{{APIObject: pagerduty.APIObject{ID: "PEP1"}, Name: "Cached EP"}})
	app.svc.loaded = true
	app.svc.loading = true
	app.ep.loaded = true
	app.ep.loading = true

	app.resetTeamScopedTabState()

	services, ok := app.tabCache.Services()
	assert.False(t, ok)
	assert.Nil(t, services)

	policies, ok := app.tabCache.EscalationPolicies()
	assert.False(t, ok)
	assert.Nil(t, policies)

	assert.False(t, app.svc.loaded)
	assert.False(t, app.svc.loading)
	assert.False(t, app.ep.loaded)
	assert.False(t, app.ep.loading)
}

func TestAppFetchFreshServicesCmd_HitsAPI(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	var hits atomic.Int32
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{
			"services": [{"id":"PSVC-FRESH","name":"Fresh Service","status":"active"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	cfg := config.Default()
	cfg.Team = "T1"

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		cfg,
		"test@example.com",
	)
	app.tabCache.PutServices([]pagerduty.Service{{APIObject: pagerduty.APIObject{ID: "PSVC-CACHED"}, Name: "Cached Service", Status: "disabled"}})

	msg := app.fetchFreshServicesCmd()()
	require.Equal(t, int32(1), hits.Load())

	loaded, ok := msg.(servicesLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.services, 1)
	assert.Equal(t, "PSVC-FRESH", loaded.services[0].ID)
	assert.Equal(t, "Fresh Service", loaded.services[0].Name)

	result, _ := app.Update(loaded)
	updated := result.(App)
	services, ok := updated.tabCache.Services()
	require.True(t, ok)
	require.Len(t, services, 1)
	assert.Equal(t, "PSVC-FRESH", services[0].ID)
}

func TestAppRefreshActiveTeamScopedTabAfterTeamChange_FetchesFreshServices(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	var hits atomic.Int32
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		assert.Equal(t, []string{"T2"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{
			"services": [{"id":"PSVC-FRESH","name":"Fresh Service","status":"active"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	cfg := config.Default()
	cfg.Team = "T1"

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		cfg,
		"test@example.com",
	)
	app.activeTab = tabIndexByID(app.tabs, "services")
	require.GreaterOrEqual(t, app.activeTab, 0)
	app.tabCache.PutServices([]pagerduty.Service{{APIObject: pagerduty.APIObject{ID: "PSVC-OLD"}, Name: "Old Service", Status: "disabled"}})
	app.svc.loaded = true
	app.cfg.Team = "T2"

	cmd := app.refreshActiveTeamScopedTabAfterTeamChange()
	require.NotNil(t, cmd)
	assert.True(t, app.svc.loading)

	msg := cmd()
	loaded, ok := msg.(servicesLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.services, 1)
	assert.Equal(t, "PSVC-FRESH", loaded.services[0].ID)
	assert.Equal(t, int32(1), hits.Load())
}

func TestAppUpdate_IgnoresStaleIncidentsLoadedMsgAfterTeamChange(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
		teamIDs := r.URL.Query()["team_ids[]"]
		if !assert.Len(t, teamIDs, 1) {
			return
		}

		switch teamIDs[0] {
		case "T1":
			_, _ = w.Write([]byte(`{
				"incidents": [{"id":"POLD","title":"Old Team Incident","status":"triggered"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		case "T2":
			_, _ = w.Write([]byte(`{
				"incidents": [{"id":"PNEW","title":"New Team Incident","status":"triggered"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		default:
			t.Fatalf("unexpected team id %q", teamIDs[0])
		}
	})

	cfg := config.Default()
	cfg.Team = "T1"

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		cfg,
		"test@example.com",
	)

	staleMsg := app.fetchIncidentsCmd()()

	result, cmd := app.Update(components.TeamSelected{TeamID: "T2", TeamName: "Team Two"})
	require.NotNil(t, cmd)
	updated := result.(App)
	assert.Equal(t, "T2", updated.cfg.Team)
	assert.Equal(t, "Team Two", updated.statusBar.Team)
	assert.True(t, updated.loading)

	result, _ = updated.Update(staleMsg)
	updated = result.(App)

	assert.Empty(t, updated.dashboard.incidents.incidents)
	assert.Zero(t, updated.statusBar.Triggered)
	assert.Zero(t, updated.statusBar.Acknowledged)
	assert.Zero(t, updated.statusBar.Resolved)
}

func TestAppUpdate_IgnoresStaleIncidentsLoadedMsgAfterFilterChange(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	mux.HandleFunc("/incidents", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"incidents": [{"id":"POLD","title":"Old Filter Incident","status":"triggered"}],
			"limit": 100, "offset": 0, "more": false, "total": 1
		}`))
	})

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		config.Default(),
		"test@example.com",
	)

	staleMsg := app.fetchIncidentsCmd()()

	rows := components.IncidentFilterRows()
	rows[0].Current = 3 // resolved
	app.filterOpts = app.filterOpts.ShowWithRows("incidents", rows)

	result, cmd := app.Update(components.FilterAppliedMsg{Origin: "incidents"})
	require.NotNil(t, cmd)
	updated := result.(App)
	assert.Equal(t, "resolved", updated.filterState.Status)
	assert.True(t, updated.loading)

	result, _ = updated.Update(staleMsg)
	updated = result.(App)

	assert.Empty(t, updated.dashboard.incidents.incidents)
	assert.Zero(t, updated.statusBar.Triggered)
	assert.Zero(t, updated.statusBar.Acknowledged)
	assert.Zero(t, updated.statusBar.Resolved)
	assert.True(t, updated.loading)
}

func TestAppRefreshServicesKey_DoesNotClearEscalationPoliciesState(t *testing.T) {
	t.Parallel()

	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)
	app.current = viewDashboard
	app.activeTab = tabIndexByID(app.tabs, "services")
	require.GreaterOrEqual(t, app.activeTab, 0)
	app.tabCache.PutServices([]pagerduty.Service{{APIObject: pagerduty.APIObject{ID: "PSVC1"}, Name: "Cached Service"}})
	app.tabCache.PutEscalationPolicies([]pagerduty.EscalationPolicy{{APIObject: pagerduty.APIObject{ID: "PEP1"}, Name: "Cached EP"}})
	app.svc.loaded = true
	app.ep.loaded = true

	result, cmd := app.updateKeyPress(tea.KeyPressMsg{Code: -2, Text: "R"})
	require.NotNil(t, cmd)
	updated := result.(App)

	services, ok := updated.tabCache.Services()
	assert.False(t, ok)
	assert.Nil(t, services)

	policies, ok := updated.tabCache.EscalationPolicies()
	require.True(t, ok)
	require.Len(t, policies, 1)
	assert.Equal(t, "PEP1", policies[0].ID)
	assert.False(t, updated.svc.loaded)
	assert.True(t, updated.svc.loading)
	assert.True(t, updated.ep.loaded)
	assert.False(t, updated.ep.loading)
}

func TestAppRefreshEscalationPoliciesKey_DoesNotClearServicesState(t *testing.T) {
	t.Parallel()

	app := New(
		context.Background(),
		api.NewClient("test-token"),
		config.Default(),
		"test@example.com",
	)
	app.current = viewDashboard
	app.activeTab = tabIndexByID(app.tabs, "escalation-policies")
	require.GreaterOrEqual(t, app.activeTab, 0)
	app.tabCache.PutServices([]pagerduty.Service{{APIObject: pagerduty.APIObject{ID: "PSVC1"}, Name: "Cached Service"}})
	app.tabCache.PutEscalationPolicies([]pagerduty.EscalationPolicy{{APIObject: pagerduty.APIObject{ID: "PEP1"}, Name: "Cached EP"}})
	app.svc.loaded = true
	app.ep.loaded = true

	result, cmd := app.updateKeyPress(tea.KeyPressMsg{Code: -2, Text: "R"})
	require.NotNil(t, cmd)
	updated := result.(App)

	policies, ok := updated.tabCache.EscalationPolicies()
	assert.False(t, ok)
	assert.Nil(t, policies)

	services, ok := updated.tabCache.Services()
	require.True(t, ok)
	require.Len(t, services, 1)
	assert.Equal(t, "PSVC1", services[0].ID)
	assert.True(t, updated.svc.loaded)
	assert.False(t, updated.svc.loading)
	assert.False(t, updated.ep.loaded)
	assert.True(t, updated.ep.loading)
}

func TestAppUpdate_IgnoresStaleServicesLoadedMsgAfterTeamChange(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		teamIDs := r.URL.Query()["team_ids[]"]
		if !assert.Len(t, teamIDs, 1) {
			return
		}

		switch teamIDs[0] {
		case "T1":
			_, _ = w.Write([]byte(`{
				"services": [{"id":"PSVC-OLD","name":"Old Team Service","status":"active"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		case "T2":
			_, _ = w.Write([]byte(`{
				"services": [{"id":"PSVC-NEW","name":"New Team Service","status":"active"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		default:
			t.Fatalf("unexpected team id %q", teamIDs[0])
		}
	})

	cfg := config.Default()
	cfg.Team = "T1"

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		cfg,
		"test@example.com",
	)
	app.activeTab = tabIndexByID(app.tabs, "services")
	require.GreaterOrEqual(t, app.activeTab, 0)

	staleMsg := app.fetchFreshServicesCmd()()

	result, cmd := app.Update(components.TeamSelected{TeamID: "T2", TeamName: "Team Two"})
	require.NotNil(t, cmd)
	updated := result.(App)
	assert.Equal(t, "T2", updated.cfg.Team)
	assert.Equal(t, "Team Two", updated.statusBar.Team)
	assert.True(t, updated.loading)
	assert.True(t, updated.svc.loading)
	assert.False(t, updated.svc.loaded)

	result, _ = updated.Update(staleMsg)
	updated = result.(App)

	services, ok := updated.tabCache.Services()
	assert.False(t, ok)
	assert.Nil(t, services)
	assert.False(t, updated.svc.loaded)
	assert.True(t, updated.svc.loading)
}

func TestAppUpdate_IgnoresStaleEscalationPoliciesLoadedMsgAfterTeamChange(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := newLocalHTTPTestServer(t, mux)

	mux.HandleFunc("/escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		teamIDs := r.URL.Query()["team_ids[]"]
		if !assert.Len(t, teamIDs, 1) {
			return
		}

		switch teamIDs[0] {
		case "T1":
			_, _ = w.Write([]byte(`{
				"escalation_policies": [{"id":"PEP-OLD","name":"Old Team EP"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		case "T2":
			_, _ = w.Write([]byte(`{
				"escalation_policies": [{"id":"PEP-NEW","name":"New Team EP"}],
				"limit": 100, "offset": 0, "more": false, "total": 1
			}`))
		default:
			t.Fatalf("unexpected team id %q", teamIDs[0])
		}
	})

	cfg := config.Default()
	cfg.Team = "T1"

	app := New(
		context.Background(),
		api.NewClient("test-token", api.WithBaseURL(server.URL)),
		cfg,
		"test@example.com",
	)
	app.activeTab = tabIndexByID(app.tabs, "escalation-policies")
	require.GreaterOrEqual(t, app.activeTab, 0)

	staleMsg := app.fetchFreshEPCmd()()

	result, cmd := app.Update(components.TeamSelected{TeamID: "T2", TeamName: "Team Two"})
	require.NotNil(t, cmd)
	updated := result.(App)
	assert.Equal(t, "T2", updated.cfg.Team)
	assert.Equal(t, "Team Two", updated.statusBar.Team)
	assert.True(t, updated.loading)
	assert.True(t, updated.ep.loading)
	assert.False(t, updated.ep.loaded)

	result, _ = updated.Update(staleMsg)
	updated = result.(App)

	policies, ok := updated.tabCache.EscalationPolicies()
	assert.False(t, ok)
	assert.Nil(t, policies)
	assert.False(t, updated.ep.loaded)
	assert.True(t, updated.ep.loading)
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

func tabIndexByID(tabs []topTab, id string) int {
	for i, tab := range tabs {
		if tab.id == id {
			return i
		}
	}
	return -1
}

func newLocalHTTPTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	var lc net.ListenConfig
	l, err := lc.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	server := &httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second},
	}
	server.Start()
	t.Cleanup(server.Close)
	return server
}
