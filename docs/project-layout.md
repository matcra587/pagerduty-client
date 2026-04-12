# Project Layout

`pdc` is a Go CLI for the PagerDuty REST v2 API.
The module is `github.com/matcra587/pagerduty-client`.

## Package Layout

| Package | Purpose |
|---------|---------|
| `cmd/` | Cobra command definitions - wiring only, no business logic |
| `internal/agent/` | Agent mode detection, envelope format, embedded guides |
| `internal/api/` | PagerDuty REST v2 client with its own HTTP layer |
| `internal/browser/` | Cross-platform browser launcher used by both CLI --open flags and TUI |
| `internal/compact/` | JSON projection with field weighting and token budgets for agent envelopes |
| `internal/config/` | Configuration management (koanf, TOML, environment variables) |
| `internal/credential/` | Credential providers - keyring and token resolution chain |
| `internal/dirs/` | Platform-aware directory helpers enforcing XDG paths on all platforms |
| `internal/integration/` | Detects and normalises alert payloads from CloudWatch, Datadog, GCP, Prometheus |
| `internal/output/` | Output formatting - tables, syntax-highlighted JSON, agent envelopes |
| `internal/resolve/` | Fuzzy ID and name resolvers for dynamic shell completion and CLI arguments |
| `internal/table/` | Builder-pattern table renderer with per-column styling, OSC 8 links and terminal-aware truncation |
| `internal/testutil/` | Shared golden JSON fixtures and typed loaders for test consumers |
| `internal/tui/` | Bubble Tea TUI - dashboard, incident list, detail view |
| `internal/update/` | Self-update logic - release detection, download, checksum verification |
| `internal/version/` | Build-time version information injected via ldflags |

## How the Packages Relate

`cmd/` wires flags, arguments and subcommands together.
Commands call into `internal/api/` to fetch data, then pass results to `internal/output/` to format them.

`internal/api/` owns all HTTP communication with PagerDuty.
It uses `go-pagerduty` for types only - `Incident`, `Service`, `User` and so on - but never its HTTP client.

`internal/output/` handles format detection, JSON syntax highlighting, agent envelopes and control character sanitisation.
Table rendering lives in `internal/table/`, which `cmd/` calls directly for list and detail views.

`internal/table/` provides a builder-pattern renderer where each column declares its own styling (colour by value, entity-coloured from a theme palette, OSC 8 hyperlinks, relative time formatting).
Flex columns fit remaining terminal width with configurable truncation.
See [output.md](output.md) for the user-facing behaviour.

`internal/browser/` is a three-line cross-platform browser launcher shared by the TUI's open-in-browser key binding and the CLI's `--open` / `--open-external` flags on `pdc incident show`.

`internal/tui/` runs when the user passes `--interactive`.
It reads from the same `internal/api/` client and renders the Bubble Tea application.

`cmd/root.go` calls `internal/config/` and `internal/credential/` at startup.
They resolve configuration from file, environment variables and the OS keyring before any command runs.

The incident detail view calls `internal/integration/` to parse monitoring alert payloads embedded in incident bodies and extract structured fields for display.

`internal/dirs/` is a thin helper for config and cache paths.
It overrides macOS `~/Library` paths to use `~/.config` and `~/.cache` on all platforms.

## Key Design Decisions

**Own HTTP layer.**
`pdc` does not use `go-pagerduty`'s HTTP client.
It owns its own `http.Client` with timeouts, rate limiting and `Retry-After` header handling.
This keeps control over backoff behaviour and request lifecycle.

**`go-pagerduty` for types only.**
The upstream library provides well-maintained Go structs for PagerDuty resources.
Importing it for types avoids duplicating struct definitions and keeps HTTP behaviour under our control.

**Agent mode auto-detection.**
Every command produces structured JSON.
`internal/agent/` detects agents by checking environment variables set by Claude Code, Cursor, Copilot, Codex, Aider, Cline, Windsurf, Amazon Q, Gemini Code Assist and Sourcegraph Cody.
When detected, output is wrapped in a JSON envelope.
The `--agent` flag forces agent mode manually; `--interactive` opts humans into the TUI.

**Credential resolution chain.**
Tokens never go in the config file.
The resolution chain is: `--token` / `--token-file` flag, then `PDC_TOKEN` environment variable, then the OS keyring.
`--token` is visible in `ps`; prefer `--token-file` for programmatic use.
