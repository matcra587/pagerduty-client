# Incident Triage

Decision tree for triaging PagerDuty incidents.

## 1. List open incidents

```text
pdc incident list --status triggered --status acknowledged
```

If you know the team, scope the query:

```text
pdc incident list --status triggered --status acknowledged --team PTEAMID
```

Sort the results by urgency. Handle high-urgency incidents first.

## 2. Check who is on call

```text
pdc oncall --team PTEAMID
```

If the incident belongs to a specific escalation policy:

```text
pdc oncall --escalation-policy PPOLICYID
```

## 3. Acknowledge

Stop escalation while you investigate:

```text
pdc incident ack PINCID
```

## 4. Inspect the incident

Get full details and attached alerts:

```text
pdc incident show PINCID --alerts
```

If alert payloads contain useful diagnostic data:

```text
pdc incident show PINCID --alerts --payload
```

## 5. Check the timeline

```text
pdc incident log PINCID
```

Filter to recent entries if the timeline is long:

```text
pdc incident log PINCID --since 2026-03-30T00:00:00Z
```

## 6. Decide on action

Choose one:

*   **Resolve** if the problem is mitigated:

  ```text
  pdc incident resolve PINCID
  ```

*   **Escalate/reassign** if someone else should handle it:

  ```text
  pdc user list --team PTEAMID
  pdc incident reassign PINCID --user PUSERID
  ```

*   **Snooze** if action is needed later:

  ```text
  pdc incident snooze PINCID --duration 2h
  ```

*   **Change urgency** if severity was set incorrectly:

  ```text
  pdc incident urgency PINCID low
  ```

*   **Resolve specific alerts** if only some alerts within the incident are resolved:

  ```text
  pdc incident resolve-alert PINCID PALERTID1 PALERTID2
  ```

## 7. Document findings

Add a note with your analysis:

```text
pdc incident note add PINCID --content "Root cause: connection pool exhaustion"
```

Review existing notes:

```text
pdc incident note list PINCID
```

## 8. Merge duplicates

If multiple incidents describe the same problem, merge them into one:

```text
pdc incident merge PTARGETID --source PDUPID1 --source PDUPID2
```

## 9. Resolve

Once the problem is mitigated:

```text
pdc incident resolve PINCID
```

Optionally attach a resolution note:

```text
pdc incident resolve PINCID --note "Restarted worker pool, alerts cleared"
```

## Rules

*   Always acknowledge before investigating. Unacknowledged incidents
  continue to escalate.
*   High-urgency incidents page the on-call user. Low-urgency incidents
  do not. Address high-urgency first.
*   Snoozed incidents re-trigger when the duration expires. Snooze is
  not a resolution.
*   `--all` overrides `--since`/`--until` and default status filters.
  If you combine `--all` with `--status`, the explicit status is
  preserved.
*   `--from` is auto-detected from the API token. Only set it when
  acting on behalf of a different user.
*   Merge is irreversible. The source incidents are folded into the
  target and cannot be separated.
*   There is no `pdc alert list` command. Alerts are sub-resources of
  incidents. Use `pdc incident show --alerts` to view them.
*   `--status` accepts one value per flag. Pass it multiple times for
  multiple statuses: `--status triggered --status acknowledged`.
