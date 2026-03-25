# Services

Services represent components or systems that generate incidents.
Each service has an escalation policy that controls who gets notified.

## List services

```text
pdc service list
pdc service list --team <name-or-id>
pdc service list --query "api"
pdc service list --sort name:asc
```

## Show a service

```text
pdc service show <id>
```

## Reference

| Flag | Purpose |
|------|---------|
| `--team` | Filter by team name or ID |
| `--query` | Filter by service name |
| `--sort` | Sort order: name, name:asc, name:desc |

## Output fields

| Field | Notes |
|-------|-------|
| ID | Service ID (starts with `P`) |
| Name | Service name |
| Status | active, warning, critical, maintenance, disabled |
| Escalation Policy | Associated policy name |

A status of `maintenance` means an active maintenance window is open.
Use `--format json` for full detail including integrations.
