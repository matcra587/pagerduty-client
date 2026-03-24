---
description: >
  Testing conventions: TDD workflow, testify usage, httptest patterns,
  naming, and what to cover. Loaded for test files and API code.
paths:
  - "**/*_test.go"
  - "internal/api/**/*.go"
---

# Testing

- TDD: write failing test first, then minimal implementation to pass
- Use `stretchr/testify` - `require` for fatal, `assert` for non-fatal
- Use `net/http/httptest` for API client tests
- Never import or use go-pagerduty's HTTP client in tests

## API Client Test Pattern

```go
func TestClient_ListIncidents(t *testing.T) {
    mux := http.NewServeMux()
    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)

    mux.HandleFunc("/incidents", func(w http.ResponseWriter, r *http.Request) {
        require.Equal(t, http.MethodGet, r.Method)
        require.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"incidents": [{"id": "P123"}], "more": false}`))
    })

    client := NewClient("test-token", WithBaseURL(server.URL))
    incidents, err := client.ListIncidents(context.Background(), ListIncidentsOpts{})

    require.NoError(t, err)
    assert.Len(t, incidents, 1)
    assert.Equal(t, "P123", incidents[0].ID)
}
```

## Naming

- `internal/api/incident.go` → `internal/api/incident_test.go`
- `internal/config/config.go` → `internal/config/config_test.go`
- Test functions: `TestTypeName_MethodName` or `TestFunctionName`
- Table-driven tests for multiple cases

## What to Test

- API client: correct HTTP method, path, headers, query params, response parsing
- Config: precedence chain, env var overrides, TOML parsing, defaults
- Agent detection: env var sniffing, flag override
- Output: JSON envelope structure, table formatting
- Do NOT test Cobra command wiring or TUI rendering in unit tests

## Go 1.25+ Testing Features

- **`testing/synctest`**: Virtualised time for testing concurrent code
  with timers, tickers or channel timeouts.
- **`testing.ArtifactDir()`** (Go 1.26): Structured output directory
  for test files (pass `-artifacts` flag).
