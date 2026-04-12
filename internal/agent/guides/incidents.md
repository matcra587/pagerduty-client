# Incidents

Complete reference for incident operations via `pdc`.

## Listing incidents

List open incidents (default: triggered and acknowledged):

```text
pdc incident list
```

Filter by status.
Pass `--status` once per value:

```text
pdc incident list --status triggered --status acknowledged
pdc incident list --status resolved --since 2026-03-29T00:00:00Z
```

Filter by urgency:

```text
pdc incident list --urgency high
pdc incident list --urgency low
```

Filter by team, service, schedule or user:

```text
pdc incident list --team PTEAMID
pdc incident list --service PSVCID
pdc incident list --schedule PSCHEDID
pdc incident list --user PUSERID
```

Combine filters freely.
Results auto-paginate up to the PagerDuty offset cap of 10,000 rows.

Include resolved incidents with `--all`:

```text
pdc incident list --all
```

Sort results.
The PagerDuty API accepts `incident_number`, `created_at`, `resolved_at` and `urgency`, each with `:asc` or `:desc`:

```text
pdc incident list --sort created_at:desc
pdc incident list --sort incident_number:desc
```

Time-bound queries:

```text
pdc incident list --since 2026-03-01T00:00:00Z --until 2026-03-30T00:00:00Z
```

## Show incident details

```text
pdc incident show PINCID
```

List alerts grouped under the incident:

```text
pdc incident show PINCID --alerts
```

Show all integration-detected fields, including verbose ones normally hidden (Summary, Body, Tags, etc.):

```text
pdc incident show PINCID --detailed
```

Open the PagerDuty incident URL in the default browser:

```text
pdc incident show PINCID --open
```

Open the integration-detected external link (e.g. a Datadog monitor, Grafana dashboard) in the default browser.
Warns and exits if no external link is configured:

```text
pdc incident show PINCID --open-external
```

Both `--open` flags are no-ops in agent mode.

## Timeline

Show the incident log — acknowledge, escalate, notify and other events:

```text
pdc incident log PINCID
```

Filter to recent entries:

```text
pdc incident log PINCID --since 2026-03-30T00:00:00Z
```

Show the summary overview only:

```text
pdc incident log PINCID --overview
```

## Acknowledge

Stop escalation while investigating.
Does not resolve:

```text
pdc incident ack PINCID
```

## Escalate

Escalate to the next level in the incident's escalation policy:

```text
pdc incident escalate PINCID
```

Errors when the incident is already at the highest level, has no escalation policy, or the next level has no targets.

## Resolve

```text
pdc incident resolve PINCID
```

Attach a resolution note.
On an interactive terminal without `--note`, `pdc` prompts for an optional note; pass `--note` to skip the prompt or run non-interactively:

```text
pdc incident resolve PINCID --note "Deployed hotfix v2.3.1"
```

## Resolve specific alerts

Resolve individual alerts without resolving the incident:

```text
pdc incident resolve-alert PINCID PALERTID1 PALERTID2
```

## Snooze

Pause notifications.
The incident re-triggers when the duration expires:

```text
pdc incident snooze PINCID --duration 4h
```

## Reassign

Reassign to one or more users:

```text
pdc incident reassign PINCID --user PUSERID
pdc incident reassign PINCID --user PUSER1 --user PUSER2
```

## Merge

Merge duplicate incidents into a single target:

```text
pdc incident merge PTARGETID --source PDUPID1 --source PDUPID2
```

## Notes

Add a note:

```text
pdc incident note add PINCID --content "Cause identified: memory leak in worker"
```

List existing notes:

```text
pdc incident note list PINCID
```

## Set urgency

```text
pdc incident urgency PINCID high
pdc incident urgency PINCID low
```

## Set title

```text
pdc incident title PINCID "Updated incident title"
```

## Set priority

```text
pdc incident priority PINCID P1
```

Clear the priority (case-insensitive):

```text
pdc incident priority PINCID none
```

The priority name must match a priority configured on the PagerDuty account, case-insensitive.
`pdc` errors with the list of valid priority names when the account has priorities but the name does not match.

## Common patterns

**Find stale acknowledged incidents.**
List acknowledged incidents, inspect each `created_at`, then escalate or resolve:

```text
pdc incident list --status acknowledged
```

**Bulk action across a team.**
List triggered incidents for a team, then act on each:

```text
pdc incident list --status triggered --team PTEAMID
pdc incident ack PINCID1
pdc incident ack PINCID2
```

**Investigate an incident.**
Combine show, alerts, timeline and notes:

```text
pdc incident show PINCID
pdc incident show PINCID --alerts
pdc incident log PINCID
pdc incident note list PINCID
```

Jump to the source system:

```text
pdc incident show PINCID --open-external
```

## Agent output

In agent mode, `incident list` and `incident show` return projected JSON.
Compaction strips reference noise (`self` and `type` on every nested object) and incident metadata of low agent value (`assigned_via`, `first_trigger_log_entry`, `last_status_change_by`, `conference_bridge`, `incident_responders`, `responder_requests`, `pending_actions`, `is_mergeable`, `occurrence`, `resolve_reason`).
Top-level `html_url`, `description` and `incident_key` are preserved but low-weighted — the field budget may drop them for large payloads.

Top-weighted fields (always preserved):
`id`, `title`, `status`, `urgency`, `priority`, `integration`.

Next tier:
`service`, `assignments`, `incident_number`, `created_at`, `body`, `escalation_policy`, `teams`, `acknowledgements`, `alert_counts`.

Use `--format json` for the full unmodified PagerDuty API response.

## Browser and terminal links

List and show tables render incident IDs as OSC 8 hyperlinks on terminals that support them.
Ctrl-click or cmd-click an ID to open it in the PagerDuty web UI.

`--open` and `--open-external` on `incident show` launch the default browser.
`--open-external` resolves the link in this order: configured custom fields with `display = "link"`, then integration-detected links from the alert body.
When no link is found, `pdc` logs a warning and exits without opening anything.

## Flag reference

For the exhaustive, up-to-date flag list run:

```text
pdc agent schema
```

This returns the command tree and all flags as JSON.
Use `--compact` for smaller output.
The schema is authoritative and always matches the installed binary.

## Rules

*   The default `pdc incident list` returns only triggered and acknowledged incidents.
  Pass `--all` to include resolved.
*   `--all` overrides `--since` and `--until` and clears the default status filter.
  An explicit `--status` survives: `--all --status resolved` keeps the status.
*   `--status` accepts one value per flag.
  Pass it multiple times: `--status triggered --status acknowledged`.
*   `--schedule` resolves the current on-call users for the schedule and filters by those user IDs.
  It does not filter by schedule directly.
*   `--from` is auto-detected from the API token.
  Set it only when acting on behalf of a different user.
*   There is no `pdc alert list` command.
  Alerts are sub-resources of incidents; use `pdc incident show --alerts`.
*   To see all parsed integration fields on `incident show`, pass `--detailed`.
  For the raw alert body, use `pdc incident show --alerts --format json`.
*   For integration-detection debugging, run with `--debug` (or
  `PDC_DEBUG=1`). `enrichIncident` emits the raw alert body, detected
  source and counts of extracted fields and links at debug level.
*   Merge is irreversible.
  Source incidents remain in PagerDuty but their activity consolidates into the target's timeline.
*   Snooze pauses notifications for the given duration.
  The incident re-triggers when the snooze expires; it is not a resolution.
*   `--duration` accepts Go duration strings: `30m`, `1h`, `4h`, `24h`.
*   `--since` and `--until` accept ISO 8601 timestamps.
  `--since` also accepts shorthands: `7d`, `30d`, `60d`, `90d`.
