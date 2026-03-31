# Incidents

Complete reference for incident operations.

## Listing incidents

List all open incidents (default: triggered and acknowledged):

```text
pdc incident list
```

Filter by status. Pass `--status` once per value:

```text
pdc incident list --status triggered --status acknowledged
pdc incident list --status resolved --since 2026-03-29T00:00:00Z
```

Filter by urgency:

```text
pdc incident list --urgency high
pdc incident list --urgency low
```

Filter by team, service, schedule or user:

```text
pdc incident list --team PTEAMID
pdc incident list --service PSVCID
pdc incident list --schedule PSCHEDID
pdc incident list --user PUSERID
```

Combine filters freely. All results auto-paginate.

Include resolved incidents with `--all`:

```text
pdc incident list --all
```

Sort results:

```text
pdc incident list --sort incident_number:desc
```

Time-bound queries:

```text
pdc incident list --since 2026-03-01T00:00:00Z --until 2026-03-30T00:00:00Z
```

## Show incident details

```text
pdc incident show PINCID
```

Include attached alerts:

```text
pdc incident show PINCID --alerts
```

Include full alert payloads (integration-specific data):

```text
pdc incident show PINCID --alerts --payload
```

## Timeline

Show the incident log (acknowledge, escalate, notify events):

```text
pdc incident log PINCID
```

Filter to recent entries:

```text
pdc incident log PINCID --since 2026-03-30T00:00:00Z
```

Show a summary overview:

```text
pdc incident log PINCID --overview
```

## Acknowledge

Stop escalation while investigating. Does not resolve:

```text
pdc incident ack PINCID
```

## Resolve

```text
pdc incident resolve PINCID
```

Attach a resolution note:

```text
pdc incident resolve PINCID --note "Deployed hotfix v2.3.1"
```

## Resolve specific alerts

Resolve individual alerts without resolving the entire incident:

```text
pdc incident resolve-alert PINCID PALERTID1 PALERTID2
```

## Snooze

Pause notifications. The incident re-triggers when the duration expires:

```text
pdc incident snooze PINCID --duration 4h
```

## Reassign

Reassign to one or more users:

```text
pdc incident reassign PINCID --user PUSERID
pdc incident reassign PINCID --user PUSER1 --user PUSER2
```

## Merge

Merge duplicate incidents into a single target:

```text
pdc incident merge PTARGETID --source PDUPID1 --source PDUPID2
```

## Notes

Add a note:

```text
pdc incident note add PINCID --content "Cause identified: memory leak in worker"
```

List existing notes:

```text
pdc incident note list PINCID
```

## Set urgency

```text
pdc incident urgency PINCID high
pdc incident urgency PINCID low
```

## Set title

```text
pdc incident title PINCID "Updated incident title"
```

## Common patterns

**Find stale acknowledged incidents** - list acknowledged incidents,
check how long each has been open, escalate or resolve:

```text
pdc incident list --status acknowledged
```

Inspect each result's `created_at`. If older than your threshold,
reassign or resolve.

**Bulk operations across a team** - list all triggered incidents for a
team, then acknowledge or resolve each:

```text
pdc incident list --status triggered --team PTEAMID
pdc incident ack PINCID1
pdc incident ack PINCID2
```

**Investigate an incident fully** - combine show, alerts, timeline and
notes:

```text
pdc incident show PINCID --alerts --payload
pdc incident log PINCID
pdc incident note list PINCID
```

## Agent output

In agent mode, `incident list` and `incident show` return projected
JSON. Duplicate fields (`summary`, `description`), reference noise
(`self`, `type`, `html_url` on nested objects) and low-value metadata
are stripped.

Key fields: `id`, `incident_number`, `title`, `status`, `urgency`,
`priority`, `created_at`, `service`, `escalation_policy`, `assignees`,
`alert_counts`.

Use `--format json` for the full unmodified PagerDuty API response.

## Flag reference

| Flag | Commands | Values |
|------|----------|--------|
| `--status` | list | triggered, acknowledged, resolved (repeatable) |
| `--urgency` | list | high, low (repeatable) |
| `--team` | list | team ID (repeatable) |
| `--service` | list | service ID (repeatable) |
| `--user` | list, reassign | user ID (repeatable) |
| `--schedule` | list | schedule ID (repeatable) |
| `--all` | list | include all statuses |
| `--sort` | list | sort order |
| `--since` | list, log | start time (ISO 8601) |
| `--until` | list, log | end time (ISO 8601) |
| `--overview` | log | summary view |
| `--alerts` | show | include attached alerts |
| `--payload` | show | include full alert payloads |
| `--from` | ack, resolve, snooze, reassign, merge, note add | acting user email |
| `--duration` | snooze | Go duration string (e.g. 4h, 30m) |
| `--content` | note add | note text |
| `--note` | resolve | resolution note |
| `--source` | merge | source incident ID (repeatable) |

## Rules

*   The default `pdc incident list` returns only triggered and
  acknowledged incidents. Pass `--all` to include resolved.
*   `--all` overrides `--since`/`--until` and default status filters.
  If you combine `--all` with explicit `--status`, the status filter
  is preserved.
*   `--status` accepts one value per flag. Pass it multiple times:
  `--status triggered --status acknowledged`.
*   `--schedule` resolves the current on-call users for that schedule
  and filters by those user IDs. It does not filter by schedule
  directly.
*   `--from` is auto-detected from the API token. Only set it when
  acting on behalf of a different user.
*   There is no `pdc alert list` command. Alerts are sub-resources of
  incidents. Use `pdc incident show --alerts`.
*   Merge is irreversible. Source incidents cannot be unmerged.
*   Snooze pauses notifications for the given duration. The incident
  re-triggers when the snooze expires. It is not a resolution.
*   `--duration` accepts Go duration strings: `30m`, `1h`, `4h`, `24h`.
*   `--since` and `--until` accept ISO 8601 timestamps. `--since` also
  accepts shorthands: `7d`, `30d`, `60d`, `90d`.
