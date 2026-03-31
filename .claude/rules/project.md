---
description: >
  Project architecture: PagerDuty API client, CLI framework (clib/clog),
  agent mode, output formatting, TUI, and Cobra patterns. Loaded for all Go files.
paths:
  - "**/*.go"
---

# Project Architecture

How this PagerDuty CLI is structured. Covers the API client, CLI
framework (clib/clog), agent mode, output formatting and TUI.

## Principles

- Keep the codebase clear, concise and easy to follow.
- Don't add files, packages or features before they're needed (YAGNI).
- Don't create abstractions for one-time operations.

---

## API Client (internal/api/)

- Use a custom HTTP transport layer; never use go-pagerduty's client
- Import go-pagerduty for entity types only (Incident, Service, User, etc.)
- Pass `context.Context` as first argument to every method
- Base URL: `https://api.pagerduty.com`
- Auth header: `Authorization: Token token=<api-token>`
- Accept header: `application/vnd.pagerduty+json;version=2`
- User-Agent: `pagerduty-client/<version>`
- HTTP timeout: 30s default

### Error Handling

- 429: respect Retry-After header, retry up to 3x with exponential backoff
- 5xx: retry up to 3x with backoff (1s, 2s, 4s)
- 4xx: no retry, return `*APIError` immediately
- Network errors: retry up to 3x with backoff
- Retries respect context cancellation

### Pagination

- Default page size: 100
- Auto-paginate by default, collecting all results
- PD hard limit: 10,000 offset-based results on some endpoints

### Adding a New Resource

Each resource file follows the same shape:

1. Options struct for list filtering
2. Client methods that call `c.do()` or `c.get()`/`c.post()`/`c.put()`
3. Corresponding test file with httptest mocks

---

## CLI Framework (gechr/clib)

`gechr/clib` provides Cobra extensions, themed help, shell completion,
terminal detection and human-readable formatting. Import sub-packages
directly; alias the Cobra integration as `clib`.

### Imports

```go
clib     "github.com/gechr/clib/cli/cobra"   // Cobra flag extensions, help, completion
         "github.com/gechr/clib/complete"     // Shell completion generator
         "github.com/gechr/clib/help"         // Help rendering
         "github.com/gechr/clib/terminal"     // TTY detection
         "github.com/gechr/clib/theme"        // Help theme
         "github.com/gechr/clib/human"        // Human-readable formatting
         "github.com/gechr/clib/ansi"         // ANSI escape utilities
clibtheme "github.com/gechr/clib/theme"       // When aliased in TUI theme code
```

### Flag Extension Pattern

Define Cobra flags first, then extend each with `clib.Extend` for
themed help, completion hints and validation metadata.

```go
pf := cmd.PersistentFlags()
pf.StringP("token", "t", "", "PagerDuty API token")

clib.Extend(pf.Lookup("token"), clib.FlagExtra{
    Group:       "Connection",
    Placeholder: "TOKEN",
    Terse:       "API token",
})
```

### FlagExtra Fields

| Field | Type | Purpose |
|-------|------|---------|
| Group | string | Category in help (Connection, Filters, Output, Action) |
| Placeholder | string | Value placeholder (TOKEN, NAME\|ID, PATH, DURATION) |
| Terse | string | One-line description |
| Enum | []string | Allowed values |
| EnumTerse | []string | Short descriptions for enum values (parallel to Enum) |
| EnumDefault | string | Default for display |
| Complete | string | Completion predicate (e.g. `"predictor=team"`) |
| Hint | string | Input hint (e.g. `"file"`) |

### Flag Group Categories

- **Connection** - token, config
- **Filters** - team, status, urgency, service, user, schedule, query, since, until
- **Output** - format, interactive, agent, debug, colour, sort, compact
- **Action** - from, duration, user (reassignment), source, content

### Command Structure

Root command uses `PersistentPreRunE` to run before every command:

1. Load config (file → env → flags)
2. Set clog verbosity and colour mode
3. Detect agent mode
4. Wire shell completion (exit if handled)
5. Create API client
6. Store config, client, agent detection and user email on context

Parent commands group related subcommands (no `RunE`):

```go
var incidentCmd = &cobra.Command{
    Use:     "incident",
    Short:   "Manage PagerDuty incidents",
    GroupID: "resources",
}
```

Leaf commands do the work. Register in `init()`:

```go
func init() {
    rootCmd.AddCommand(incidentCmd)
    incidentCmd.AddCommand(incidentListCmd)

    lf := incidentListCmd.Flags()
    lf.StringSlice("status", nil, "Filter by status")
    clib.Extend(lf.Lookup("status"), clib.FlagExtra{
        Group: "Filters",
        Enum:  []string{"triggered", "acknowledged", "resolved"},
        Terse: "status filter",
    })
}
```

Group subcommands in help output with `rootCmd.AddGroup(...)`.

### Context Helpers

Store values in `PersistentPreRunE`, retrieve in subcommands:

```go
func ConfigFromContext(cmd *cobra.Command) *config.Config {
    v, _ := cmd.Context().Value(configKey).(*config.Config)
    return v
}
```

Follow this pattern for: config, API client, agent detection, user email.

### Help Rendering

```go
th := theme.New(theme.WithEnumStyle(theme.EnumStyleHighlightBoth))
renderer := help.NewRenderer(th)
rootCmd.SetHelpFunc(clib.HelpFunc(renderer, clib.Sections,
    help.WithHelpFlags("Print help", "Print help with examples"),
    help.WithLongHelp(os.Args, help.Section{
        Title: "Examples",
        Content: []help.Content{
            help.Examples{
                {Comment: "Launch the TUI", Command: "pdc --interactive"},
                {Comment: "List incidents as JSON", Command: "pdc incident list -f json"},
            },
        },
    }),
))
```

### Shell Completion

Wire after all subcommands register:

```go
func setup() {
    comp = clib.NewCompletion(rootCmd)
}

// In PersistentPreRunE:
gen := complete.NewGenerator("pdc").FromFlags(clib.FlagMeta(cmd.Root()))
gen.Subs = clib.Subcommands(cmd.Root())
handled, err := comp.Handle(gen, completionHandler(token, opts...))
if handled {
    os.Exit(0)
}
```

All completion logic lives in `cmd/completion.go`:

- `completionHandler` - runtime handler that queries the PD API
  for resource IDs matching a completion kind.

Positional arg completion uses Cobra annotations. clib parses
`dynamic-args` from the `clib` annotation key on each command:

```go
var myCmd = &cobra.Command{
    Use:         "show <id>",
    Annotations: map[string]string{"clib": "dynamic-args='service'"},
}
```

When adding a new subcommand with positional args:

1. Add `Annotations: map[string]string{"clib": "dynamic-args='<kind>'"}`.
2. Add a handler `case` in `completionHandler` if the kind is new.
3. Add `Complete: "predictor=<kind>"` on any flag that takes a
   resource ID (via `clib.FlagExtra`).
4. Reinstall completions: `pdc --install-completion`.

### Terminal Detection

```go
isTTY := terminal.Is(os.Stdout)
```

Use to choose output format. Require interactive terminal for prompts.

---

## CLI Logging (gechr/clog)

`gechr/clog` is the only logger. `depguard` bans `log` and `log/slog`.

```go
"github.com/gechr/clog"
```

### Two Output Paths

- **clog** - operational feedback (token verified, incident acknowledged,
  debug timing). Human-readable, coloured, goes to terminal.
- **internal/output/** - command data output (incident list, service
  detail). Structured, parseable, goes to stdout. Never route data
  through clog.

### Levels

| Level | Use |
|-------|-----|
| `Trace` | Finest-grained, hidden by default |
| `Debug` | Verbose (API timing, token source, agent detection) |
| `Info` | Default - user feedback for write commands |
| `Warn` | Non-fatal warnings |
| `Error` | Errors via `Execute()` with `SilenceErrors: true` |
| `Fatal` | Calls `os.Exit` after logging |

### Field Methods

Use typed fields, not string interpolation:

```go
clog.Info().Msg("token verified")
clog.Error().Err(err).Send()                    // error IS the message
clog.Info().Link("incident", url, id).Msg("acknowledged")
clog.Info().Path("config", path).Msg("written")
clog.Info().Duration("duration", dur).Msg("snoozed")
clog.Info().Strs("users", userIDs).Msg("reassigned")
clog.Debug().Elapsed("duration").Int("count", n).Msg("listed incidents")
clog.Debug().Str("source", src).Bool("active", det.Active).Msg("agent detection")
```

| Method | Purpose |
|--------|---------|
| `.Err(err)` | Attach error. `.Send()` = error as message. `.Msg()` = error as field |
| `.Str(k, v)` / `.Strs(k, vs)` | String / string slice |
| `.Int(k, v)` / `.Bool(k, v)` | Typed numeric/boolean |
| `.Duration(k, v)` | time.Duration (styled) |
| `.Elapsed(k)` | Wall time from call to finalisation |
| `.Link(k, url, text)` | OSC8 clickable hyperlink |
| `.Path(k, v)` | File path hyperlink |
| `.JSON(k, v)` / `.RawJSON(k, b)` | Syntax-highlighted JSON field |

### Configuration

Set in `PersistentPreRunE`:

```go
clog.SetEnvPrefix("PDC")          // PDC_LOG_LEVEL=debug
clog.SetVerbose(cfg.Debug)
clog.SetColorMode(clog.ColorAuto) // or ColorAlways, ColorNever
```

### Error Handling Pattern

Cobra errors go through clog, not Cobra's default output:

```go
var rootCmd = &cobra.Command{
    SilenceErrors: true,
    SilenceUsage:  true,
}

func Execute() error {
    err := rootCmd.Execute()
    if err != nil {
        clog.Error().Err(err).Send()
    }
    return err
}
```

### Usage Rules

- Use clog for feedback, `internal/output/` for data.
- Use typed fields (`.Int()`, `.Duration()`, `.Strs()`) not `.Str()`
  with `fmt.Sprintf`.
- End every chain with `.Msg()` or `.Send()`. Unchained = no-op.
- Keep messages lowercase, brief and specific.
- `.Elapsed("duration")` on every API-calling read command for debug timing.

---

## Agent Mode (internal/agent/, internal/output/)

Every command MUST produce structured JSON output. JSON is the output contract.

### Output Priority

1. Agent mode detected → JSON with envelope (metadata, hints, errors)
2. `--format json` on TTY → syntax-highlighted JSON (chroma, monokai)
3. `--format json` piped → plain JSON
4. `--interactive` / `-i` → TUI (opt-in, configurable via config.toml)
5. TTY → table (styled)
6. No TTY / piped → table (plain, no colour)

### Agent Envelope

```json
{
  "success": true,
  "command": "incident list",
  "data": [],
  "metadata": {},
  "hints": []
}
```

Errors:

```json
{
  "success": false,
  "command": "incident ack",
  "error": {"code": 401, "message": "...", "suggestion": "..."}
}
```

### Detection

Any truthy env var activates agent mode:
CLAUDE_CODE, CLAUDECODE, CURSOR_AGENT, CODEX, OPENAI_CODEX, AIDER,
CLINE, WINDSURF_AGENT, GITHUB_COPILOT, AMAZON_Q, AWS_Q_DEVELOPER,
GEMINI_CODE_ASSIST, SRC_CODY, FORCE_AGENT_MODE

The `--agent` flag also activates it.

### Agent Subcommands

- `pdc agent schema` - walk Cobra tree, emit JSON schema of all commands
  (hidden flags are filtered out)
- `pdc agent schema --compact` - schema without descriptions
- `pdc agent guide <name>` - emit embedded markdown guide
- Embed guides via go:embed from internal/agent/guides/

---

## TUI (internal/tui/)

- Build with Bubble Tea (charmbracelet) and bubbles components
- Style with Lipgloss
- Keep TUI separate from cmd/
- Inject API client into the root model; components must not import it

### Theme

- Urgency colouring: high = red, low = yellow, resolved = dim
- Define styles in internal/tui/theme/
- Theme wraps `gechr/clib/theme` with presets: dark, light, high-contrast
- Configure with: `WithRed`, `WithGreen`, `WithYellow`, `WithBlue`,
  `WithMagenta`, `WithOrange`, `WithDim`, `WithBoldGreen`
- `.EntityColors` provides a 20-colour palette for consistent hashing
- Call `theme.Apply(preset())` once at startup before rendering
- `human.FormatTimeAgoCompact(t)` for plain-text relative time;
  `theme.RenderTimeAgoCompact(t, bool)` for coloured output

### Key Bindings - List View

j/k or arrows = navigate, g/G = jump top/bottom, enter = detail,
space = toggle selection, ctrl+a = select all, esc = back/deselect,
a = ack, r = resolve (confirm), alt+r = resolve (immediate),
e = edit, x = escalate (confirm), alt+x = escalate (immediate),
m = merge selected (confirm), alt+m = merge (immediate),
s = snooze, n = note, y = copy URL, o = open in browser,
alt+o = open external link,
/ = filter, O = filter options, R = toggle refresh,
t = team switcher, ? = help, ctrl+c/q = quit

### Key Bindings - Detail View

tab/shift+tab or left/right = switch tabs,
up/down = scroll, a = ack, r = resolve (confirm),
alt+r = resolve (immediate),
e = edit, x = escalate (confirm), alt+x = escalate (immediate),
p = set priority (picker),
y = copy URL, o = open in browser, alt+o = open external link,
? = help, esc = back, ctrl+c/q = quit

### Data Flow

- Poll on a configurable tick (default 30s); R toggles polling
- Switching team or user triggers a re-fetch
- Cache slow-changing data (teams, users, schedules) in memory
- Fetch alerts and notes on demand when opening detail views
