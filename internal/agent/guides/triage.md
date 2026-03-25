# Incident Triage Guide

## Overview

This guide covers the end-to-end incident triage workflow using `pagerduty-client`.
It is designed for AI agents handling on-call response.

## Step 1 - Assess the situation

List all open (triggered + acknowledged) incidents for your team:

```text
pagerduty-client incident list --status triggered,acknowledged --team <team>
```

Check who is currently on call:

```text
pagerduty-client oncall list --team <team>
```

## Step 2 - Inspect a specific incident

```text
pagerduty-client incident show <id>
```

This returns full detail: title, status, urgency, service, alerts, notes.

## Step 3 - Acknowledge to stop escalation

```text
pagerduty-client incident ack <id>
```

## Step 4 - Investigate

List alerts attached to the incident:

```text
pagerduty-client alert list <incident-id>
```

## Step 5 - Escalate or reassign if needed

Find relevant users:

```text
pagerduty-client user list --team <team>
```

Reassign the incident:

```text
pagerduty-client incident reassign <id> --users <user-id>
```

## Step 6 - Add a note

Document your findings:

```text
pagerduty-client incident note <id> \
  --content "Root cause: database connection exhaustion."
```

## Step 7 - Resolve

Once mitigated:

```text
pagerduty-client incident resolve <id>
```

## Step 8 - Handle duplicates

If multiple incidents are duplicates, merge them:

```text
pagerduty-client incident merge <canonical-id> --sources <dup1>,<dup2>
```

## Tips

- All commands support `--agent` for structured JSON output.
- `pagerduty-client incident list` returns all results automatically (auto-paginated).
- High-urgency incidents should always be addressed before low-urgency ones.
- Snoozed incidents reappear as triggered after the snooze
  expires.
