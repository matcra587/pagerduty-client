package cmd

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gechr/clib/complete"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureCompletion runs the handler and collects printed lines.
// Not safe to call in parallel - it redirects os.Stdout for the
// duration of the handler call.
func captureCompletion(t *testing.T, handler complete.Handler, shell, kind string, args []string) []string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w
	handler(shell, kind, args)
	os.Stdout = orig
	_ = w.Close()

	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func TestCompletionHandler(t *testing.T) {
	// Sequential: captureCompletion redirects os.Stdout.
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /incidents", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "title": "Disk full"}, {"id": "P2", "title": "CPU spike"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"alerts": [{"id": "A1", "status": "triggered", "summary": "Host unreachable"}, {"id": "A2", "status": "resolved", "summary": "Disk warning"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /teams", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"teams": [{"id": "T1", "name": "Platform"}, {"id": "T2", "name": "Mobile"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /services", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"services": [{"id": "S1", "name": "Auth API"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"users": [{"id": "U1", "name": "Alice"}, {"id": "U2", "name": "Bob"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /schedules", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"schedules": [{"id": "SCH1", "name": "Primary"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"escalation_policies": [{"id": "EP1", "name": "Platform"}, {"id": "EP2", "name": "Mobile"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"maintenance_windows": [{"id": "MW1", "description": "Deploy window"}, {"id": "MW2", "description": "DB migration"}], "limit": 100, "offset": 0, "more": false}`))
	})

	handler := completionHandler("test-token", nil, api.WithBaseURL(server.URL))

	tests := []struct {
		name     string
		shell    string
		kind     string
		args     []string
		expected []string
	}{
		{name: "incident IDs", shell: "zsh", kind: "incident", expected: []string{"P1", "P2"}},
		{name: "alert IDs filters resolved", shell: "zsh", kind: "alert", args: []string{"P1"}, expected: []string{"A1"}},
		{name: "alert no args", shell: "zsh", kind: "alert"},
		{name: "team IDs", shell: "zsh", kind: "team", expected: []string{"T1", "T2"}},
		{name: "service IDs", shell: "zsh", kind: "service", expected: []string{"S1"}},
		{name: "user IDs", shell: "zsh", kind: "user", expected: []string{"U1", "U2"}},
		{name: "schedule IDs", shell: "zsh", kind: "schedule", expected: []string{"SCH1"}},
		{name: "escalation policy IDs", shell: "zsh", kind: "escalation_policy", expected: []string{"EP1", "EP2"}},
		{name: "maintenance window IDs", shell: "zsh", kind: "maintenance_window", expected: []string{"MW1", "MW2"}},
		{name: "urgency static", shell: "zsh", kind: "urgency", expected: []string{"high", "low"}},
		{name: "unknown kind", shell: "zsh", kind: "bogus"},

		// Fish receives tab-separated ID\tDescription pairs.
		{name: "fish incident descriptions", shell: "fish", kind: "incident", expected: []string{"P1\tDisk full", "P2\tCPU spike"}},
		{name: "fish team descriptions", shell: "fish", kind: "team", expected: []string{"T1\tPlatform", "T2\tMobile"}},
		{name: "fish service descriptions", shell: "fish", kind: "service", expected: []string{"S1\tAuth API"}},
		{name: "fish user descriptions", shell: "fish", kind: "user", expected: []string{"U1\tAlice", "U2\tBob"}},
		{name: "fish schedule descriptions", shell: "fish", kind: "schedule", expected: []string{"SCH1\tPrimary"}},
		{name: "fish alert descriptions", shell: "fish", kind: "alert", args: []string{"P1"}, expected: []string{"A1\tHost unreachable"}},
		{name: "fish escalation policy descriptions", shell: "fish", kind: "escalation_policy", expected: []string{"EP1\tPlatform", "EP2\tMobile"}},
		{name: "fish maintenance window descriptions", shell: "fish", kind: "maintenance_window", expected: []string{"MW1\tDeploy window", "MW2\tDB migration"}},
		{name: "fish urgency no descriptions", shell: "fish", kind: "urgency", expected: []string{"high", "low"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureCompletion(t, handler, tt.shell, tt.kind, tt.args)
			if tt.expected == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestCompletionHandler_NoToken(t *testing.T) {
	handler := completionHandler("", nil)
	got := captureCompletion(t, handler, "zsh", "incident", nil)
	assert.Empty(t, got)
}

func TestCompletionHandler_ConfigFilters(t *testing.T) {
	// Sequential: captureCompletion redirects os.Stdout.
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /incidents", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		assert.Equal(t, []string{"S1"}, r.URL.Query()["service_ids[]"])
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1", "title": "Filtered"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /services", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{"services": [{"id": "S1", "name": "Auth API"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{"users": [{"id": "U1", "name": "Alice"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{"escalation_policies": [{"id": "EP1", "name": "Platform"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /maintenance_windows", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1"}, r.URL.Query()["team_ids[]"])
		assert.Equal(t, []string{"S1"}, r.URL.Query()["service_ids[]"])
		_, _ = w.Write([]byte(`{"maintenance_windows": [{"id": "MW1", "description": "Deploy window"}], "limit": 100, "offset": 0, "more": false}`))
	})

	cfg := &config.Config{Team: "T1", Service: "S1"}
	handler := completionHandler("test-token", cfg, api.WithBaseURL(server.URL))

	tests := []struct {
		name     string
		kind     string
		expected []string
	}{
		{name: "incidents filtered by team and service", kind: "incident", expected: []string{"P1"}},
		{name: "services filtered by team", kind: "service", expected: []string{"S1"}},
		{name: "users filtered by team", kind: "user", expected: []string{"U1"}},
		{name: "escalation policies filtered by team", kind: "escalation_policy", expected: []string{"EP1"}},
		{name: "maintenance windows filtered by team and service", kind: "maintenance_window", expected: []string{"MW1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureCompletion(t, handler, "zsh", tt.kind, nil)
			assert.Equal(t, tt.expected, got)
		})
	}
}
