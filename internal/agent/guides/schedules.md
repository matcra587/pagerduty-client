# Schedules Guide

## Overview

Schedules define on-call rotations. Each schedule contains layers of users
who rotate being on call. Use `pagerduty-client schedule` to list, view and manage overrides.

## Common Workflows

### List all schedules

```text
pagerduty-client schedule list
```

### View a schedule

```text
pagerduty-client schedule show <id>
```

### List overrides for a schedule

```text
pagerduty-client schedule overrides <id>
pagerduty-client schedule overrides <id> --since 2026-01-01T00:00:00Z --until 2026-01-31T23:59:59Z
```

### Create an override

```text
pagerduty-client schedule override <id> --user <user-id> --start 2026-01-15T09:00:00Z --end 2026-01-15T17:00:00Z
```

## Tips

- Schedule IDs start with `P` followed by alphanumeric characters.
- Overrides replace the scheduled on-call user for a specific time window.
- Times must be in RFC3339 / ISO 8601 format with timezone offset.
