# On-Call

Shows who is on call right now for each schedule and escalation policy.

## List on-call entries

```text
pdc oncall
pdc oncall --team <name-or-id>
pdc oncall --schedule <schedule-id>
pdc oncall --escalation-policy <policy-id>
```

## Filter by time window

```text
pdc oncall --since 2026-01-01T00:00:00Z --until 2026-01-02T00:00:00Z
```

## Reference

| Flag | Purpose |
|------|---------|
| `--team` | Filter by team name or ID |
| `--schedule` | Filter by schedule ID |
| `--escalation-policy` | Filter by escalation policy ID |
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
