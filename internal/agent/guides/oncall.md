# On-Call

Operations for finding who is on call.

## Who is on call right now

List all current on-call entries:

```text
pdc oncall
```

## Filter by team

```text
pdc oncall --team PTEAMID
```

Pass `--team` multiple times to check several teams:

```text
pdc oncall --team PTEAM1 --team PTEAM2
```

## Filter by schedule

```text
pdc oncall --schedule PSCHEDID
```

## Filter by escalation policy

```text
pdc oncall --escalation-policy PPOLICYID
```

## Check a specific time window

Find who was (or will be) on call during a period:

```text
pdc oncall --since 2026-03-29T00:00:00Z --until 2026-03-29T12:00:00Z
```

## Common patterns

**Find who was on call when an incident fired** - get the incident's
`created_at` timestamp from `pdc incident show`, then query on-call
for that time:

```text
pdc incident show PINCID
pdc oncall --escalation-policy PPOLICYID --since 2026-03-29T02:00:00Z --until 2026-03-29T02:01:00Z
```

Use a narrow time window (one minute around the incident creation time)
to get the exact on-call person.

**Find primary on-call for a team** - filter results by escalation
level. Level 1 is the primary responder:

```text
pdc oncall --team PTEAMID
```

Check the `level` field in the output. Level 1 = primary.

## Flag reference

| Flag | Purpose |
|------|---------|
| `--team` | Filter by team ID (repeatable) |
| `--schedule` | Filter by schedule ID (repeatable) |
| `--escalation-policy` | Filter by escalation policy ID (repeatable) |
| `--since` | Start of time window (ISO 8601) |
| `--until` | End of time window (ISO 8601) |

## Output fields

| Field | Notes |
|-------|-------|
| User | Name and ID of the on-call person |
| Schedule | Schedule name |
| Escalation Policy | Policy name |
| Level | Escalation level (1 = primary) |
| Start / End | On-call shift window |

## Rules

*   `pdc oncall` returns all on-call entries, not just the primary.
  Check the `level` field to find the primary responder (level 1).
*   `--team` filters by team membership. It does not filter by the
  team's schedules specifically.
*   `--since` and `--until` accept ISO 8601 timestamps only (no
  shorthands like `7d`).
*   Use a narrow time window (one minute) when checking who was on
  call at a specific moment. Wide windows return all shifts that
  overlap the window.
