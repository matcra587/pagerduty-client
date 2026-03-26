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
- Use `stretchr/testify` - `require` for guards/preconditions, `assert` for verifications
- Argument order is always `(expected, actual)` - swapping produces backwards diffs
- Use `assert.ErrorIs`/`require.ErrorIs` to check errors, not `Equal` (fails on wrapped errors)
- When using mocks, always call `AssertExpectations(t)` - without it expectations pass silently
- `assert.Equal` on pointers compares addresses, not values - dereference first
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

## Integration Tests

Gate behind `//go:build integration`. Run with `task test:integration`.

- Hit the PagerDuty Stoplight mock (needs network)
- Use `WithBaseURL` + `WithExtraHeaders` for mock headers
- Assert structure (non-empty ID, non-empty lists), not specific values
  (mock returns random data)
- Skip gracefully if mock is unreachable

### Non-Paginated Endpoints

These endpoints return a plain JSON object without pagination fields.
Use simple GET + decode, NOT `paginate()`:

- `GET /users/{id}/contact_methods`
- `GET /schedules/{id}/overrides`

### Mock Response Accuracy

httptest mock responses must match the PagerDuty OpenAPI spec. Verify
envelope keys, pagination structure and field names against
`docs/superpowers/reference/pagerduty_api_optimized_bundle.json`.

## Go 1.25+ Testing Features

- **`testing/synctest`**: Virtualised time for testing concurrent code
  with timers, tickers or channel timeouts.
- **`testing.ArtifactDir()`** (Go 1.26): Structured output directory
  for test files (pass `-artifacts` flag).
