# Schedules

Schedules define on-call rotations. Each schedule contains layers of
users who take turns being on call.

## List schedules

```text
pdc schedule list
pdc schedule list --query "primary"
```

## Show a schedule

```text
pdc schedule show <id>
```

## Create an override

Overrides replace the scheduled on-call user for a specific time window.

```text
pdc schedule override <schedule-id> --user <user-id> --start 2026-01-15T09:00:00Z --end 2026-01-15T17:00:00Z
```

## Reference

| Flag | Commands | Purpose |
|------|----------|---------|
| `--query` | list | Filter by schedule name |
| `--user` | override | User ID to put on call |
| `--start` | override | Override start time (ISO 8601, required) |
| `--end` | override | Override end time (ISO 8601, required) |

## Tips

- Schedule IDs start with `P` followed by alphanumeric characters.
- All times must include a timezone offset (RFC 3339).
