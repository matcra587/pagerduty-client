# Configuration

`pdc config init` creates `~/.config/pagerduty-client/config.toml` and
stores your API token in the OS keyring.

## Credential resolution

Tokens never go in the config file. Resolution order (highest wins):

`--token` | `--token-file` > `PDC_TOKEN` env var > OS keyring

Prefer `--token-file` for programmatic use. `--token` is visible in
process listings.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `PDC_BASE_URL` | API base URL (default: `https://api.pagerduty.com`) |
| `PDC_DEBUG` | Enable debug output (1 or true) |
| `PDC_EMAIL` | Acting user email for write operations |
| `PDC_FORMAT` | Output format: table or json |
| `PDC_INTERACTIVE` | Launch TUI by default (1 or true) |
| `PDC_LOG_LEVEL` | Log verbosity: trace, debug, info, warn, error (handled by clog) |
| `PDC_SERVICE` | Default service name or ID |
| `PDC_TEAM` | Default team name or ID |
| `PDC_THEME` | Colour theme: dracula, monokai, monochrome, catppuccin-latte/frappe/macchiato/mocha, default (overrides [ui] theme) |
| `PDC_TOKEN` | API token (overrides keyring) |
| `PDC_UPDATE_CHANNEL` | Update channel: stable or dev (overrides [defaults] update_channel) |

Environment variables override config file values.
Command-line flags override both.

## Config file

```toml
credential_source = "keyring"
# base_url = "https://api.pagerduty.com"  # default

[defaults]
format = "table"           # table or json
team = ""                  # default team name or ID
service = ""               # default service name or ID
email = ""                 # acting user email
interactive = false        # launch TUI by default
refresh_interval = 30      # polling interval in seconds

[ui]
theme = "dracula"          # see docs/output.md for preset list

[ui.tui]
show_resolved = false      # show resolved incidents in TUI
# page_size = 50           # incidents per page
# tabs = ["incidents", "escalation-policies", "services", "teams"]
```

## Output

Colour theme, formats, truncation behaviour and agent mode are
documented separately. See [docs/output.md](output.md).

## Precedence

Sources load in this order (later overrides earlier):

1.  Built-in defaults
2.  Config file (`~/.config/pagerduty-client/config.toml`)
3.  `.env` file in the working directory (if present)
4.  Environment variables (`PDC_*`)
5.  Command-line flags

## Custom fields

Map PagerDuty custom detail fields to display columns in the TUI
detail view:

```toml
[[custom_fields]]
label = "Severity"
path = "details.custom_details.severity"
display = "inline"     # inline (default) or block

[[custom_fields]]
label = "Runbook"
path = "details.custom_details.runbook"
display = "block"
```

`path` is a dot-separated key into the alert body JSON.
pdc checks both the top-level body and `cef_details`.
