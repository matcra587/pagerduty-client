package cmd

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gechr/clib/complete"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureCompletion runs the handler and collects printed lines.
// Not safe to call in parallel - it redirects os.Stdout for the
// duration of the handler call.
func captureCompletion(t *testing.T, handler complete.Handler, kind string, args []string) []string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w
	handler("zsh", kind, args)
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
		_, _ = w.Write([]byte(`{"incidents": [{"id": "P1"}, {"id": "P2"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /incidents/P1/alerts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"alerts": [{"id": "A1", "status": "triggered"}, {"id": "A2", "status": "resolved"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /teams", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"teams": [{"id": "T1"}, {"id": "T2"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /services", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"services": [{"id": "S1"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"users": [{"id": "U1"}, {"id": "U2"}], "limit": 100, "offset": 0, "more": false}`))
	})
	mux.HandleFunc("GET /schedules", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"schedules": [{"id": "SCH1"}], "limit": 100, "offset": 0, "more": false}`))
	})

	handler := completionHandler("test-token", api.WithBaseURL(server.URL))

	tests := []struct {
		name     string
		kind     string
		args     []string
		expected []string
	}{
		{name: "incident IDs", kind: "incident", expected: []string{"P1", "P2"}},
		{name: "alert IDs filters resolved", kind: "alert", args: []string{"P1"}, expected: []string{"A1"}},
		{name: "alert no args", kind: "alert"},
		{name: "team IDs", kind: "team", expected: []string{"T1", "T2"}},
		{name: "service IDs", kind: "service", expected: []string{"S1"}},
		{name: "user IDs", kind: "user", expected: []string{"U1", "U2"}},
		{name: "schedule IDs", kind: "schedule", expected: []string{"SCH1"}},
		{name: "urgency static", kind: "urgency", expected: []string{"high", "low"}},
		{name: "unknown kind", kind: "bogus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureCompletion(t, handler, tt.kind, tt.args)
			if tt.expected == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestCompletionHandler_NoToken(t *testing.T) {
	handler := completionHandler("")
	got := captureCompletion(t, handler, "incident", nil)
	assert.Empty(t, got)
}

func TestSetDynamicArgs(t *testing.T) {
	t.Parallel()
	subs := []complete.SubSpec{
		{
			Name: "incident",
			Subs: []complete.SubSpec{
				{Name: "list"},
				{Name: "show"},
				{Name: "ack"},
				{Name: "resolve"},
				{Name: "snooze"},
				{Name: "reassign"},
				{Name: "merge"},
				{Name: "log"},
				{Name: "urgency"},
				{Name: "title"},
				{Name: "resolve-alert"},
				{Name: "note", Subs: []complete.SubSpec{
					{Name: "add"},
					{Name: "list"},
				}},
			},
		},
		{Name: "service", Subs: []complete.SubSpec{{Name: "list"}, {Name: "show"}}},
		{Name: "user", Subs: []complete.SubSpec{{Name: "list"}, {Name: "show"}, {Name: "me"}}},
		{Name: "team", Subs: []complete.SubSpec{{Name: "list"}, {Name: "show"}}},
		{Name: "schedule", Subs: []complete.SubSpec{{Name: "list"}, {Name: "show"}, {Name: "override"}}},
		{Name: "oncall"},
		{Name: "version"},
	}

	setDynamicArgs(subs)

	inc := subs[0]
	find := func(name string) complete.SubSpec {
		t.Helper()
		for _, s := range inc.Subs {
			if s.Name == name {
				return s
			}
		}
		require.FailNowf(t, "subcommand not found", "%q", name)
		return complete.SubSpec{}
	}

	// No positional args.
	assert.Nil(t, find("list").DynamicArgs)

	// Single incident ID.
	for _, name := range []string{"show", "ack", "resolve", "snooze", "reassign", "merge", "log"} {
		assert.Equal(t, []string{"incident"}, find(name).DynamicArgs, name)
	}

	// Incident ID + free text.
	assert.Equal(t, []string{"incident"}, find("title").DynamicArgs)

	// Incident ID + urgency.
	assert.Equal(t, []string{"incident", "urgency"}, find("urgency").DynamicArgs)

	// Incident ID + alert IDs.
	assert.Equal(t, []string{"incident", "alert"}, find("resolve-alert").DynamicArgs)

	// Nested note subcommands.
	note := find("note")
	for _, sub := range note.Subs {
		assert.Equal(t, []string{"incident"}, sub.DynamicArgs, "note "+sub.Name)
	}

	// Service.
	assert.Nil(t, subs[1].Subs[0].DynamicArgs, "service list")
	assert.Equal(t, []string{"service"}, subs[1].Subs[1].DynamicArgs, "service show")

	// User.
	assert.Nil(t, subs[2].Subs[0].DynamicArgs, "user list")
	assert.Equal(t, []string{"user"}, subs[2].Subs[1].DynamicArgs, "user show")
	assert.Nil(t, subs[2].Subs[2].DynamicArgs, "user me")

	// Team.
	assert.Nil(t, subs[3].Subs[0].DynamicArgs, "team list")
	assert.Equal(t, []string{"team"}, subs[3].Subs[1].DynamicArgs, "team show")

	// Schedule.
	assert.Nil(t, subs[4].Subs[0].DynamicArgs, "schedule list")
	assert.Equal(t, []string{"schedule"}, subs[4].Subs[1].DynamicArgs, "schedule show")
	assert.Equal(t, []string{"schedule"}, subs[4].Subs[2].DynamicArgs, "schedule override")

	// No dynamic args on commands without positional args.
	assert.Nil(t, subs[5].DynamicArgs, "oncall")
	assert.Nil(t, subs[6].DynamicArgs, "version")
}
