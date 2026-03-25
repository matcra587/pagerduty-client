# Services Guide

## Overview

Services represent a component or system that generates incidents.
Use `pagerduty-client service` to list and inspect services.

## Common Workflows

### List all services

```text
pagerduty-client service list
```

### Filter services by team

```text
pagerduty-client service list --team <name-or-id>
```

### View a service

```text
pagerduty-client service show <id>
```

## Output Fields

| Field | Description |
|-------|-------------|
| ID | PagerDuty service ID |
| Name | Service name |
| Status | active, warning, critical, maintenance, disabled |
| Escalation Policy | Associated escalation policy |

## Tips

- Service IDs start with `P`.
- Use `--format json` for full service detail including integrations.
- Status `maintenance` means the service has an active maintenance window.
