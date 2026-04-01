---
description: >
  Testing conventions: TDD workflow, testify usage, httptest patterns,
  naming, and what to cover. Loaded for test files and API code.
paths:
  - "**/*_test.go"
  - "internal/api/**/*.go"
---

# Testing

## Principles

Tests are executable specifications. They constrain behaviour and
make failures self-explanatory. Do not write tests to hit coverage
targets.

### Test behaviour, not wiring

Test observable inputs and outputs. Do not test that Cobra calls
RunE, that `clib.Extend` sets metadata, or that `fmt.Println`
prints. Those are framework guarantees.

### One test, one concern

Each test function or subtest verifies one behaviour. A test named
`TestListEscalationPolicies_WithTeamFilter` tests that team IDs
are passed as query params. Not the full response parsing.

### Extract testable helpers

When command logic gets complex, pull it into a pure function
(e.g. `escalationRuleRows`, `humanise`) and test that in
isolation with table-driven tests. Keep RunE thin.

### Do not duplicate coverage

Error paths handled centrally by `do()` (401, 403, 429 retry)
are tested once in `client_test.go`. Per-resource tests only
need the happy path plus resource-specific error codes (e.g.
402 for abilities, 404 for gets). Do not re-test retry logic
on every endpoint.

### Skip pointless tests

Do not test:
- Cobra command registration or flag binding
- TUI rendering (visual, test manually)
- Simple getters/setters with no logic
- Framework behaviour (arg validation, help rendering)

## Testify Usage

- Use `stretchr/testify` - `require` for guards, `assert` for
  verifications
- `require` when the next line would panic on nil (e.g. after
  `NoError`, `NotNil`, `Len`)
- `assert` for the actual verification (e.g. `Equal`, `Contains`)
- Do not use `require` and `assert` randomly. The choice signals
  intent: "stop here" vs "check this"
- Argument order is always `(expected, actual)` - swapping
  produces backwards diffs
- Use `ErrorIs`/`require.ErrorIs` for errors, not `Equal` (fails
  on wrapped errors). `ErrorIs` already implies `Error`, so do
  not call both
- `assert.Equal` on pointers compares addresses - dereference first
- When using mocks, always call `AssertExpectations(t)`
- Never import or use go-pagerduty's HTTP client in tests

## Test Structure

- `t.Parallel()` on all tests and subtests that do not share
  mutable state (e.g. `os.Stdout` redirection)
- Table-driven tests for multiple cases with named subtests
- Each test sets up its own `httptest.NewServer` and calls
  `t.Cleanup(server.Close)`
- Test naming: `TestFunctionName` or `TestFunctionName_Variant`
  (e.g. `TestListAbilities`, `TestListAbilities_Empty`)
- Keep the same naming style within a file
- Use `server` and `client` as variable names for httptest servers
  and API clients. Never `srv` or `c`.

## API Client Test Pattern

```go
func TestListIncidents(t *testing.T) {
    t.Parallel()
    mux := http.NewServeMux()
    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)

    mux.HandleFunc("GET /incidents", func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
        _, _ = w.Write([]byte(`{"incidents": [{"id": "P123"}], "limit": 100, "offset": 0, "more": false}`))
    })

    client := NewClient("test-token", WithBaseURL(server.URL))
    incidents, err := client.ListIncidents(context.Background(), ListIncidentsOpts{})

    require.NoError(t, err)
    require.Len(t, incidents, 1)
    assert.Equal(t, "P123", incidents[0].ID)
}
```

### What each API test should verify

- **Happy path**: correct response parsing (ID, key fields)
- **Query params**: dedicated test per filter flag (query, team,
  service, filter) verifying `r.URL.Query()`
- **Pagination**: multi-page test with `atomic.Int32` counter
- **Not found** (for Get methods): 404 returns `ErrNotFound`
- **Nil pointer fields**: test with field absent from response
  when the field is used in display or formatting logic where nil
  would panic

### What API tests should NOT verify

- 401/403 on every endpoint (tested centrally)
- 429 retry on every endpoint (tested in `client_test.go`)
- Response envelope key names (rely on the implementation)

## Command-Layer Tests

Only test extracted helper functions (e.g. `escalationRuleRows`,
`maintenanceWindowRows`). Use table-driven tests.

Do not test RunE, init(), flag parsing or Cobra wiring.

## Mock Response Accuracy

httptest mock responses must match the PagerDuty OpenAPI spec.
Verify envelope keys, pagination structure and field names against
`docs/superpowers/reference/pagerduty_api_optimized_bundle.json`.

## Integration Tests

Gate behind `//go:build integration`. Run with `task test:integration`.

- Hit the PagerDuty Stoplight mock (needs network)
- Use `WithBaseURL` + `WithExtraHeaders` for mock headers
- Assert structure (non-empty ID, non-empty lists), not specific
  values (mock returns random data)
- Skip gracefully if mock is unreachable

### Non-Paginated Endpoints

These endpoints return a plain JSON object without pagination
fields. Use simple GET + decode, NOT `paginate()`:

- `GET /users/{id}/contact_methods`
- `GET /schedules/{id}/overrides`
