# Teams Guide

## Overview

Teams group users, services and escalation policies in PagerDuty.
Use `pagerduty-client team` to list and inspect teams. Use `--team` on other commands
to filter results to a specific team.

## Common Workflows

### List all teams

```text
pagerduty-client team list
```

### Search for a team by name

```text
pagerduty-client team list --query "platform"
```

### View a team

```text
pagerduty-client team show <id>
```

### Use a team as a filter on other commands

```text
pagerduty-client incident list --team "Platform Engineering"
pagerduty-client oncall list --team PABCDEF
pagerduty-client service list --team "Platform Engineering"
```

## Tips

- `--team` accepts either a team name or a team ID (starting with `P`).
- Team names are resolved via the API and cached in memory during the session.
- Ambiguous team names (multiple matches) return an error - use the ID instead.
