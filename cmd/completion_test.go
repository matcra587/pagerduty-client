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
