# Incidents

Incidents represent problems that need attention. Each incident belongs
to a service and follows an escalation policy.

## List incidents

```text
pdc incident list
pdc incident list --status triggered,acknowledged
pdc incident list --urgency high
pdc incident list --team <name-or-id>
pdc incident list --service <id>
pdc incident list --schedule <id>
```

All results are returned (auto-paginated). Passing `--schedule` resolves
the current on-call users and filters by those user IDs.

## Show an incident

```text
pdc incident show <id>
```

## Acknowledge

Stops escalation. Does not resolve.

```text
pdc incident ack <id>
```

## Resolve

```text
pdc incident resolve <id>
```

## Snooze

Pauses notifications. The incident re-triggers when the duration expires.

```text
pdc incident snooze <id> --duration 4h
```

## Reassign

```text
pdc incident reassign <id> --user <user-id>
```

Pass `--user` multiple times to assign to several users.

## Add a note

```text
pdc incident note <id> --content "Root cause identified"
```

## Merge duplicates

Merges source incidents into a single target.

```text
pdc incident merge <target-id> --source <id1> --source <id2>
```

## Reference

| Flag | Commands | Values |
|------|----------|--------|
| `--status` | list | triggered, acknowledged, resolved |
| `--urgency` | list | high, low |
| `--team` | list | team ID |
| `--service` | list | service ID |
| `--user` | list, reassign | user ID |
| `--schedule` | list | schedule ID |
| `--from` | ack, resolve, snooze, reassign, merge, note | acting user email (auto-detected from token) |
| `--duration` | snooze | Go duration string (e.g. 4h, 30m) |
| `--content` | note | note text |
| `--source` | merge | source incident ID |

## Agent output

In agent mode, `incident list` and `incident show` return a projected
JSON format. Duplicate fields (`summary`, `description`), reference
noise (`self`, `type`, `html_url` on nested objects) and low-value
metadata are stripped.

Key fields: `id`, `incident_number`, `title`, `status`, `urgency`,
`priority`, `created_at`, `service`, `escalation_policy`, `assignees`,
`alert_counts`.

Use `--format json` to get the full unmodified PagerDuty API response.
