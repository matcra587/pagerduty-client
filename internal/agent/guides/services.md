# Services

Operations for listing and inspecting PagerDuty services. Services
represent components or systems that generate incidents.

## List services

List all services:

```text
pdc service list
```

Filter by team:

```text
pdc service list --team PTEAMID
```

Pass `--team` multiple times to include services from several teams:

```text
pdc service list --team PTEAM1 --team PTEAM2
```

Search by name:

```text
pdc service list --query "payments-api"
```

Sort results:

```text
pdc service list --sort name:asc
pdc service list --sort name:desc
```

## Show service details

```text
pdc service show PSVCID
```

Use `--format json` for the full API response including integrations.

## Common patterns

**Find all services for a team** - list services filtered to the team,
then inspect any that look relevant:

```text
pdc service list --team PTEAMID
pdc service show PSVCID
```

**Check a service by name** - use `--query` to search, then show
details for the match:

```text
pdc service list --query "checkout"
pdc service show PSVCID
```

**Find the service for an incident** - show the incident, read the
`service` field, then look up the service:

```text
pdc incident show PINCID
pdc service show PSVCID
```

## Flag reference

| Flag | Purpose |
|------|---------|
| `--team` | Filter by team ID (repeatable) |
| `--query` | Search by service name |
| `--sort` | Sort order: name, name:asc, name:desc |

## Output fields

| Field | Notes |
|-------|-------|
| ID | Service ID (starts with `P`) |
| Name | Service name |
| Status | active, warning, critical, maintenance, disabled |
| Escalation Policy | Associated policy name |

A status of `maintenance` means an active maintenance window is open.

## Rules

*   Services are read-only in pdc. You cannot create, update or delete
  services through the CLI.
*   `--query` searches by name, not ID. To look up a service by ID, use
  `pdc service show <id>`.
*   The `--service` persistent flag on root filters incidents by service
  ID. It does not affect `pdc service list`.
