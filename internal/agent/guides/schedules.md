# Schedules

Operations for listing, inspecting and overriding PagerDuty schedules.
Schedules define on-call rotations.

## List schedules

List all schedules:

```text
pdc schedule list
```

Search by name:

```text
pdc schedule list --query "primary"
```

## Show schedule details

```text
pdc schedule show PSCHEDID
```

## Create an override

Replace the scheduled on-call user for a specific time window.
All flags are required:

```text
pdc schedule override PSCHEDID --user PUSERID --start 2026-04-01T09:00:00Z --end 2026-04-01T17:00:00Z
```

## Common patterns

**Cover for a colleague going on leave** - find the schedule, look up
the covering user, then create overrides for each day:

```text
pdc schedule list --query "primary"
pdc user list --team PTEAMID
pdc schedule override PSCHEDID --user PCOVERID --start 2026-04-07T09:00:00Z --end 2026-04-11T09:00:00Z
```

For multi-day coverage, set `--start` to the beginning of the first
day and `--end` to the beginning of the day after the last day.

**Check who is currently on call for a schedule** - use the oncall
command filtered to the schedule:

```text
pdc oncall --schedule PSCHEDID
```

## Flag reference

For the exhaustive, up-to-date flag list run:

```text
pdc agent schema
```

This returns the command tree and all flags as JSON. Use
`--compact` for smaller output. The schema is authoritative and
always matches the installed binary.

## Rules

*   `--user`, `--start` and `--end` are all required for overrides.
  Omitting any of them produces an error.
*   Times must be ISO 8601 with a timezone offset (e.g.
  `2026-04-01T09:00:00Z`). Bare dates are rejected.
*   Schedules are read-only apart from overrides. You cannot create,
  modify or delete schedules through pdc.
*   `--from` is auto-detected from the API token. Only set it when
  acting on behalf of a different user.
*   To check who is on call for a schedule, use `pdc oncall --schedule`
  rather than `pdc schedule show`.
