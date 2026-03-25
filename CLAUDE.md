# pagerduty-client

PagerDuty API client and CLI in Go. Binary name: `pdc`.
Every command produces structured JSON for agents. TUI dashboard
for humans (opt-in via `--interactive`).

## Quick Start

```bash
mise install                     # Install Go, task, actionlint, rumdl, zizmor
task deps                        # Download dependencies
task build                       # Build binary to ./dist/pdc-<os>-<arch>
task test                        # Run unit tests
task test:integration            # Run integration tests (Stoplight mock, needs network)
task lint                        # Run golangci-lint
```

## Module

`github.com/matcra587/pagerduty-client`

## Go Version

Managed via `go.mod` toolchain directive: `go 1.26` with
`toolchain go1.26.1`. mise bootstraps Go; the toolchain directive
pins the exact version.

## Dev Tools

Tools split across two managers:

**mise** (`.mise.toml`) - platform tools:

| Tool | Purpose |
|------|---------|
| go | Go runtime (version pinned by go.mod toolchain) |
| task | Task runner |
| actionlint | GitHub Actions linter |
| rumdl | Markdown linter |
| zizmor | Workflow security scanner |

**go.mod** `tool` directives - Go project tools:

| Tool | Run with |
|------|----------|
| gofumpt | `go tool gofumpt` |
| govulncheck | `go tool govulncheck` |
| golangci-lint | `go tool golangci-lint` |

## Architecture

```text
cmd/                 Cobra command definitions (wiring only, no business logic)
internal/api/        PagerDuty REST v2 client (own HTTP layer, go-pagerduty types)
internal/config/     Configuration management (koanf, TOML, env)
internal/credential/ Credential providers (keyring, future: 1password, vault)
internal/output/     Output formatting (table, JSON with chroma highlighting, agent envelope)
internal/agent/      Agent mode detection, envelope format, embedded guides
internal/tui/        Bubble Tea TUI (dashboard, incidents, components)
```

## Key Dependencies

- `spf13/cobra` - CLI framework
- `PagerDuty/go-pagerduty` - types only (Incident, Service, User, etc.)
- `gechr/clog` - structured CLI logging (chain: `.Info().Str(k,v).Msg(m)`)
- `gechr/clib` - CLI infrastructure (help rendering, completions, theme)
- `zalando/go-keyring` - OS keyring credential storage
- `charm.land/huh` - interactive prompts (init wizard)
- `alecthomas/chroma` - JSON syntax highlighting on TTY
- `stretchr/testify` - testing (require + assert)

## Credentials

Tokens never go in config.toml. Resolution chain (highest wins):
`--token` flag > `PDC_TOKEN` env var > OS keyring.

Config stores `credential_source = "keyring"` to indicate the backend.
The `pdc init` wizard validates the token and stores it in the keyring.

## Gotchas

- Binary name is `pdc`, module is `github.com/matcra587/pagerduty-client`.
- PD API rate limit: 960 req/min. The client throttles below this ceiling
  and respects Retry-After headers.
- Import go-pagerduty for types only. Never use its HTTP client.
- PD API caps offset-based pagination at 10,000 results on some endpoints.
- Some endpoints are NOT paginated (contact_methods, schedule overrides).
  Use simple GET + decode, not `paginate()`.
- `depguard` bans `log` and `log/slog` imports. Use `gechr/clog`.
- Validate emails with `net/mail.ParseAddress`, not hand-rolled checks.

## Rules Files (.claude/rules/)

The `.claude/rules/` directory contains instructions for Claude, not
project documentation. These files are terse, imperative and structured
for machine consumption. They do not need to follow GFM documentation
standards - they follow their own conventions:

- Frontmatter `paths:` controls which files trigger each rule
- Frontmatter `description:` summarises the rule's purpose
- Prose is direct and imperative ("Use X", "Never do Y")
- Code examples are minimal - just enough to show the pattern

| File | Scope | Purpose |
|------|-------|---------|
| `go.md` | `**/*.go` | Go language: style, naming, spec rules, 1.25/1.26 features |
| `project.md` | `**/*.go` | Architecture: API client, clib, clog, agent mode, TUI |
| `tests.md` | `**/*_test.go` | Testing: TDD, testify, httptest patterns |
| `docs.md` | `**/*.md` | Documentation: GFM, structure, tone |
| `contributing.md` | `**/*` | Contributing: conventional commits, workflow |

CLAUDE.md itself and AGENTS.md (symlink) are excluded from `docs.md`
rules - they follow the same terse style as the rules files.

## Writing Quality

When writing or editing prose (docs, README, error messages, commit
messages), use the `/writing-clearly-and-concisely` skill if installed.

If not installed, suggest the operator run:

```shell
bunx skills add https://github.com/softaworks/agent-toolkit --skill writing-clearly-and-concisely
```
