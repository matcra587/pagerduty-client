# Incident Triage

Step-by-step workflow for handling PagerDuty incidents. Designed for
AI agents performing on-call response.

## 1. Assess

List open incidents and check who is on call:

```text
pdc incident list --status triggered,acknowledged --team <team>
pdc oncall --team <team>
```

## 2. Inspect

Get full detail on a specific incident:

```text
pdc incident show <id>
```

## 3. Acknowledge

Stop escalation while you investigate:

```text
pdc incident ack <id>
```

## 4. Check alerts

List the alerts attached to the incident:

```text
pdc alert list --incident <incident-id>
```

## 5. Escalate or reassign

Find team members, then reassign:

```text
pdc user list --team <team>
pdc incident reassign <id> --user <user-id>
```

## 6. Document

Add a note with your findings:

```text
pdc incident note <id> --content "Root cause: connection pool exhaustion"
```

## 7. Resolve

Once the problem is mitigated:

```text
pdc incident resolve <id>
```

## 8. Merge duplicates

If several incidents describe the same problem:

```text
pdc incident merge <target-id> --source <dup1> --source <dup2>
```

## Tips

- Address high-urgency incidents before low-urgency ones.
- Snoozed incidents re-trigger when the snooze expires.
- All list commands auto-paginate and return every result.
