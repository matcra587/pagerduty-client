# pagerduty-client

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

## Installation

### Homebrew

> [!NOTE]
> The Homebrew tap requires the repository to be public.
> Until then, use one of the methods below.

```bash
brew install matcra587/tap/pagerduty-client
```

### GitHub Releases

Download a pre-built binary from the
[releases page](https://github.com/matcra587/pagerduty-client/releases)
and place it on your `PATH`.

### Go

Requires Go `1.26+`.

```bash
git config --global url."git@github.com:matcra587/".insteadOf "https://github.com/matcra587/"
GOPRIVATE=github.com/matcra587 go install github.com/matcra587/pagerduty-client/cmd/pdc@latest
```

## Quick Start

```bash
pdc init                         # First-run setup (token, defaults)
pdc incident list                # List incidents (table on TTY, JSON for agents)
pdc incident ack P000001         # Acknowledge an incident
pdc oncall                       # Who is on call
pdc --interactive                # TUI dashboard
```

## API Coverage

| Resource | List | Get | Create | Update |
|----------|------|-----|--------|--------|
| Incidents | yes | yes | - | ack, resolve, snooze, merge, reassign, note |
| Services | yes | - | - | - |
| Users | yes | - | - | - |
| Teams | yes | - | - | - |
| Schedules | yes | - | - | overrides |
| On-call | yes | - | - | - |
| Alerts | yes | - | - | - |
| Escalation policies | - | - | - | - |

## Contributing

1.  Fork and branch.
1.  Set up tooling and build:

    ```bash
    mise install && task deps && task build
    ```

1.  Write a failing test, then the implementation.
1.  `task lint && task test` before pushing.
1.  Open a pull request.

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

## Acknowledgements

Agent-first CLI design inspired by
[pup](https://github.com/datadog-labs/pup).

## Licence

[MIT](LICENSE)
