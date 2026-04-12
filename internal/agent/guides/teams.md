# Teams

Operations for listing and inspecting PagerDuty teams. Teams group
users, services and escalation policies. Most commands accept `--team`
to filter results.

## List teams

List all teams:

```text
pdc team list
```

Search by name:

```text
pdc team list --query "platform"
```

## Show team details

```text
pdc team show PTEAMID
```

## Common patterns

**Find a team ID to use as a filter** - search by name, note the ID,
then use it with other commands:

```text
pdc team list --query "platform"
pdc incident list --team PTEAMID
pdc service list --team PTEAMID
pdc oncall --team PTEAMID
```

**List all resources for a team** - combine team filtering across
commands:

```text
pdc incident list --status triggered --status acknowledged --team PTEAMID
pdc service list --team PTEAMID
pdc oncall --team PTEAMID
pdc user list --team PTEAMID
```

## Flag reference

For the exhaustive, up-to-date flag list run:

```text
pdc agent schema
```

This returns the command tree and all flags as JSON. Use
`--compact` for smaller output. The schema is authoritative and
always matches the installed binary.

## Using --team elsewhere

The `--team` flag filters results across most commands. It accepts a
team name or ID. If a name matches more than one team, pdc returns an
error - use the ID instead.

```text
pdc incident list --team "Platform Engineering"
pdc incident list --team PTEAMID
pdc oncall --team PTEAMID
pdc service list --team PTEAMID
```

## Rules

*   Teams are read-only in pdc. You cannot create, modify or delete
  teams through the CLI.
*   `--team` accepts a name or ID. If the name matches multiple teams,
  pdc returns an error. Use the ID to avoid ambiguity.
*   The root `--team` (`-T`) persistent flag sets a default team for
  all commands. It can also be set via `PDC_TEAM` or in config.
*   `--query` searches by name substring, not ID.
