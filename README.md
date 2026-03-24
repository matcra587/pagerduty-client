# pagerduty-client

**Work in progress. No PagerDuty API endpoints are implemented yet - this repo is project scaffolding only (tooling, CI, linting). See the roadmap below.**

PagerDuty from the command line - built for AI agents, usable by humans.

## Background

I couldn't find a modern PagerDuty CLI that did what I wanted.
The official [go-pagerduty](https://github.com/PagerDuty/go-pagerduty)
library hasn't seen much activity, and nothing out there treats AI
agents as a first-class interface. I built this to scratch that itch -
partly to improve my own observability workflows, partly to see how
far I could push building software in collaboration with Claude.

The result is a CLI that detects when an LLM is driving the session
and automatically switches to compact, token-efficient JSON output
that agents can parse and act on without wasting context window.
It recognises Claude Code, Cursor, Copilot and a dozen other agents
out of the box - no flags needed. Commands expose their schema for
tool discovery and serve embedded guides so agents understand what
they can do. For humans, there's a Bubble Tea TUI with live polling,
keyboard-driven actions and team switching.

> [!NOTE]
> Within the first few uses it surfaced acknowledged incidents from
> over a year ago that had been forgotten about and never resolved -
> the kind of thing that slips through when you're clicking around
> a web UI.

## Quick Start

```bash
mise install && task deps && task build

./dist/pagerduty-client                          # TUI dashboard
./dist/pagerduty-client incident list -f json    # Incidents as JSON
./dist/pagerduty-client incident ack P000001     # Acknowledge
./dist/pagerduty-client oncall                   # Who is on call
```

## Installation

[mise](https://mise.jdx.dev/) handles Go and all tooling:

```bash
mise install
task build
task install                     # Optional: install to GOPATH/bin
```

Or build from source if you already have Go 1.26+:

```bash
go install github.com/matcra587/pagerduty-client@latest
```

## Development

```bash
task test                        # Run tests
task lint                        # Lint with golangci-lint
task fmt                         # Format with gofumpt
task vet                         # Run go vet
task security                    # Run govulncheck
```

> [!TIP]
> mise manages platform tools (Go, task, actionlint, rumdl, zizmor).
> `go.mod` [tool directives][tools] manage Go project tools
> (golangci-lint, gofumpt, govulncheck).

[tools]: https://go.dev/doc/modules/managing-dependencies#tools

## API Coverage

None yet. Planned resources:

| Resource | Status |
|----------|--------|
| Incidents | Not started |
| Services | Not started |
| Users | Not started |
| Teams | Not started |
| Schedules | Not started |
| On-call | Not started |
| Escalation policies | Not started |

## Contributing

1.  Fork and branch.
1.  `mise install` to set up tooling.
1.  Write a failing test, then the implementation.
1.  `task lint && task test` before pushing.
1.  Open a pull request.

## Acknowledgements

Agent-first CLI design inspired by
[pup](https://github.com/datadog-labs/pup).

## Licence

[MIT](LICENSE)
