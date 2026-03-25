# On-Call Guide

## Overview

On-call entries show who is currently on call for each schedule and escalation
policy. Use `pagerduty-client oncall` to query the current on-call roster.

## Common Workflows

### List all current on-call entries

```text
pagerduty-client oncall list
```

### Filter by team

```text
pagerduty-client oncall list --team <name-or-id>
```

### Filter by user

```text
pagerduty-client oncall list --user <user-id>
```

### Filter by escalation policy

```text
pagerduty-client oncall list --escalation-policy <policy-id>
```

### Filter by schedule

```text
pagerduty-client oncall list --schedule <schedule-id>
```

## Output Fields

| Field | Description |
|-------|-------------|
| User | Name and ID of the on-call user |
| Schedule | Schedule name |
| Escalation Policy | Policy name |
| Level | Escalation level (1 = primary) |
| Start | When the on-call shift starts |
| End | When the on-call shift ends |

## Tips

- Use `--format json` for structured output suitable for scripting.
- Results include all active on-call entries across all escalation levels.
