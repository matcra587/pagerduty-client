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
pdc config init                  # First-run setup (token, defaults)
pdc incident list                # List incidents (table on TTY, JSON for agents)
pdc incident ack P000001         # Acknowledge an incident
pdc oncall                       # Who is on call
pdc -i                           # TUI dashboard
```

## Configuration

`pdc config init` creates `~/.config/pagerduty-client/config.toml` and
stores your API token in the OS keyring. Tokens never go in the
config file.

See [docs/configuration.md](docs/configuration.md) for the full
reference: config file fields, environment variables, credential
resolution and custom field mapping.

## Agent Mode

pdc detects AI agents automatically and switches to structured JSON
output. No flags needed - it recognises Claude Code, Cursor, Copilot
and a dozen other agents out of the box.

See [docs/agent-mode.md](docs/agent-mode.md) for the envelope format,
schema discovery and embedded operational guides.

## API Coverage

| Resource | List | Get | Update |
|----------|------|-----|--------|
| Incidents | yes | yes | ack, resolve, snooze, merge, reassign, note, urgency, title, resolve-alert |
| Services | yes | yes | - |
| Users | yes | yes | - |
| Teams | yes | yes | - |
| Schedules | yes | yes | overrides |
| On-call | yes | - | - |
| Escalation Policies | yes | yes | - |
| Maintenance Windows | yes | yes | - |
| Abilities | yes | test | - |

## TUI

Launch with `pdc -i` or set `interactive = true` in config.

Four tabs: Incidents, Escalation Policies, Services and Teams.
Tabs are configurable via `[tui] tabs` in config.toml.

The Incidents tab polls for live data and supports keyboard-driven
actions: acknowledge, resolve, snooze, escalate, reassign, merge,
add notes, edit fields and set priority. The other tabs show
expandable rows with detail on enter.

Switch teams with `t`, filter with `shift+o`, search with `/`
and toggle refresh with `R`. Press `?` for the full keybinding
reference.

## Documentation

*   [Configuration](docs/configuration.md) - config file, env vars, credentials
*   [Agent mode](docs/agent-mode.md) - detection, envelope format, guides
*   [Project layout](docs/project-layout.md) - package structure and design decisions
*   [Releasing](docs/releasing.md) - version scheme, tagging, GoReleaser
*   [Contributing](CONTRIBUTING.md) - setup, workflow, commit conventions

## Security

pdc sanitises API responses before rendering to the terminal.
PagerDuty fields like incident titles can contain raw ASCII control
characters that terminals interpret as commands - clearing the screen,
changing the window title or injecting hyperlinks.

pdc uses the [go-gh asciisanitizer](https://github.com/cli/go-gh)
to replace C0 and C1 control characters with visible caret notation
(e.g. `^[` for ESC, `^G` for BEL). Tabs, newlines and carriage
returns are preserved. If you see caret sequences in output, the
original data contained terminal escape codes that were neutralised.

JSON and agent output are not sanitised - they preserve the original
data for machine consumers.

## Acknowledgements

Agent-first CLI design inspired by
[pup](https://github.com/datadog-labs/pup).
