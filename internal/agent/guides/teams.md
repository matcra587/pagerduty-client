# Teams

Teams group users, services and escalation policies. Most commands
accept `--team` to filter results to a single team.

## List teams

```text
pdc team list
pdc team list --query "platform"
```

## Show a team

```text
pdc team show <id>
```

## Filter other commands by team

```text
pdc incident list --team "Platform Engineering"
pdc oncall --team PABCDEF
pdc service list --team "Platform Engineering"
```

`--team` accepts a name or an ID. Names are resolved via the PagerDuty
API. If a name matches more than one team, pdc returns an error - use
the ID instead.

## Reference

| Flag | Commands | Purpose |
|------|----------|---------|
| `--query` | list | Filter by team name |
