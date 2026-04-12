# Output

pdc renders human-friendly tables on a TTY and machine-friendly text when piped.
Agents get a structured JSON envelope automatically.

## Formats

The default format is a table on a TTY and plain text when piped.

Use `--format json` for structured output.
JSON renders with chroma syntax highlighting on a TTY and as plain JSON when piped.

The agent envelope auto-detects supported AI runtimes via environment variables.
Force it with `--agent`.

## Themes

Set the theme in `config.toml` under `ui.theme` or with the `PDC_THEME` environment variable.
The environment variable overrides the file.

| Preset |
|--------|
| `default` |
| `dracula` |
| `monokai` |
| `monochrome` |
| `catppuccin-latte` |
| `catppuccin-frappe` |
| `catppuccin-macchiato` |
| `catppuccin-mocha` |

The theme controls status colours, urgency colours, priority colours, dim cells, the entity palette, help text styling, markdown rendering and JSON syntax highlighting.

## Colour and styling

Table output colours columns by meaning.

Status column:

*   `triggered` renders red
*   `acknowledged` renders yellow
*   `resolved` renders green

Urgency column:

*   `high` renders red
*   `low` renders yellow

Service Status (in `service list`):

*   `active` renders green
*   `warning` renders yellow
*   `critical` renders red
*   `maintenance` renders blue
*   `disabled` renders faint

Priority (in `incident show`):

*   P1 and P2 render red
*   P3 renders yellow
*   P4 and P5 render dim

Entity names in Service, User, Assignee and Agent columns hash to one of 20 palette colours.
The same name always renders the same colour.

Column headers render bold.
Default cells outside status and flex columns render dim.

## Hyperlinks

Incident IDs render as OSC 8 hyperlinks in `incident list` and `incident show` tables.
Ctrl-click or cmd-click to open them in a browser.

The `service`, `team`, `user`, `schedule`, `escalation-policy` and `maintenance-window` list commands also linkify IDs.

`pdc incident show --open` opens the PagerDuty incident URL.

`pdc incident show --open-external` opens the integration-detected link.
This covers Datadog monitors, Grafana dashboards and similar sources.
The command warns when no external link is found.

## Width and truncation

Fixed columns truncate at 60 characters.
One column per table is marked flex (usually Title, Description or Value).

On a TTY, flex columns take the remaining terminal width.
Content truncates with `...` to fit.
Multiple flex columns share the remaining width equally.

Piped output skips flex truncation entirely and renders full content.

`--full` disables flex truncation even on a TTY.

Terminal resize is not dynamic.
Rerun the command to re-fit.

## Relative timestamps

Created, Start, End and similar timestamps render as `5m ago`, `2h ago` or `3d ago` on a TTY.
The theme applies an age-based colour gradient where available.

`--format json` and agent mode keep the original ISO 8601 timestamps.

## Agent mode

Output becomes a single JSON line wrapped in an envelope:

```json
{"success": true, "command": "...", "data": {}, "metadata": {}, "hints": [], "error": null}
```

Compaction strips reference noise such as `self` and `type` on nested objects, plus low-value metadata.
Per-resource field weights determine which fields survive the token budget.

See [agent-mode.md](agent-mode.md) for the envelope format and detection rules.

## Debugging output

`--debug` or `PDC_DEBUG=1` emits integration detection and rendering info to stderr via clog.

On `incident show`, debug mode logs the raw alert body, the detected source and the field and link counts.

Debug logs go to stderr.
Stdout stays clean for data consumers.
