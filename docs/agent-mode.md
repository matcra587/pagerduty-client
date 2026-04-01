# Agent Mode

pdc detects AI agents and switches to structured JSON output automatically.
No flags needed.

## Supported agents

Claude Code, Cursor, Copilot, Codex, Aider, Cline, Windsurf,
Amazon Q, Gemini Code Assist and Sourcegraph Cody. Detection checks environment variables each agent's runtime sets.

The `--agent` flag forces agent mode manually.

## Output envelope

Every command wraps its response in an envelope:

```json
{
  "success": true,
  "command": "incident list",
  "data": [...],
  "metadata": {},
  "hints": ["Use 'pdc incident show <id>' for full details"]
}
```

Errors use the same envelope:

```json
{
  "success": false,
  "command": "incident ack",
  "error": {
    "code": 401,
    "message": "invalid token",
    "suggestion": "Run 'pdc config init' to configure your API token"
  }
}
```

## Schema discovery

Agents can discover all commands and their flags:

```bash
pdc agent schema              # Full schema with descriptions
pdc agent schema --compact    # Schema without descriptions
```

The output lists every command with its use pattern, flags, defaults and allowed values.

## Embedded guides

pdc ships markdown guides that give agents domain context:

```bash
pdc agent guide triage        # Incident triage workflow
pdc agent guide incidents     # Working with incidents
pdc agent guide oncall        # On-call operations
pdc agent guide services      # Service management
pdc agent guide schedules     # Schedule operations
pdc agent guide teams         # Team management
```

Guides explain PagerDuty concepts and suggest command sequences for
common workflows.
