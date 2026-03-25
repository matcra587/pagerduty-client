# Incident Management Guide

## Overview

Incidents are the primary unit of work in PagerDuty. Use `pagerduty-client incident` commands
to list, view and act on incidents.

## Common Workflows

### List open incidents

```text
pagerduty-client incident list
pagerduty-client incident list --status triggered,acknowledged
pagerduty-client incident list --urgency high
pagerduty-client incident list --team <name-or-id>
```

### Inspect an incident

```text
pagerduty-client incident show <id>
```

### Acknowledge an incident

```text
pagerduty-client incident ack <id>
```

### Resolve an incident

```text
pagerduty-client incident resolve <id>
```

### Snooze an incident

```text
pagerduty-client incident snooze <id> --duration 4h
```

### Reassign to another user

```text
pagerduty-client incident reassign <id> --users <user-id>[,<user-id>]
```

### Add a note

```text
pagerduty-client incident note <id> --content "Investigating now"
```

### Merge duplicates

```text
pagerduty-client incident merge <target-id> --sources <id1>[,<id2>]
```

## Tips

- Use `--format json` or `--agent` for structured output. When running inside
  Claude Code, Cursor or other AI agents, agent mode is detected automatically
  via environment variables - the `--agent` flag is not required.
- `pagerduty-client incident list` auto-paginates - all results are returned.
- Urgency values: `high`, `low`.
- Status values: `triggered`, `acknowledged`, `resolved`.

## Agent Output Notes

When agent mode is active, `pagerduty-client incident list` and `pagerduty-client incident show` return
a projected JSON format with duplicates removed. Key fields in the projected
output:

| Field | Notes |
|---|---|
| `id` | Incident ID for follow-up commands |
| `incident_number` | Human-readable incident number |
| `title` | Incident summary |
| `status` | `triggered`, `acknowledged`, `resolved` |
| `urgency` | `high` or `low` |
| `priority` | P1-P5 label when set, null otherwise |
| `created_at` | ISO 8601 timestamp |
| `service.id` | Owning service ID |
| `service.summary` | Owning service name |
| `escalation_policy.id` | Escalation policy ID |
| `escalation_policy.summary` | Active escalation policy name |
| `assignees[].id` | Assigned user IDs (for reassign commands) |
| `assignees[].summary` | Assigned user names |
| `alert_counts` | `{triggered, resolved, all}` |
| `metadata.total` | Total result count (pagination) |

The projected format excludes duplicate fields (`summary`, `description`),
reference-object noise (`self`, `type`, `html_url` on nested refs) and
low-value metadata (`first_trigger_log_entry`, `last_status_change_by`,
`resolve_reason`, `body`).

Use `--format json` to get the full unmodified PagerDuty API response.
