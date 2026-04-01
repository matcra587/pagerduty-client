# PagerDuty API Coverage

Which PagerDuty REST API v2 endpoints `pdc` covers and which it does not.

> [!NOTE]
> Endpoints that require PagerDuty AIOps or Event Intelligence add-ons
> are excluded from this document. pdc does not currently target these
> features.

## Summary

pdc focuses on day-to-day incident response and operational
visibility: incidents, services, users, teams, schedules,
escalation policies, maintenance windows and on-call rosters.
Administrative areas (automation, orchestration, business
services, custom fields, workflows) remain uncovered.

| Category | Implemented | Total | Coverage |
|---|---|---|---|
| Incidents | 8 | 23 | 35% |
| Incident Alerts | 3 | 4 | 75% |
| Services | 2 | 20 | 10% |
| Users | 4 | 37 | 11% |
| Teams | 3 | 14 | 21% |
| Schedules | 4 | 11 | 36% |
| On-Calls | 1 | 1 | 100% |
| Escalation Policies | 2 | 6 | 33% |
| Maintenance Windows | 2 | 5 | 40% |
| Priorities | 1 | 1 | 100% |
| Abilities | 2 | 2 | 100% |
| Log Entries | 0 | 3 | 0% |
| All other resources | 0 | 283 | 0% |

---

## Incidents

Covers the core incident lifecycle: list, view, acknowledge, resolve, snooze, reassign, merge, set urgency, set title and add notes.

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/incidents` | Implemented | `pdc incident list`, TUI |
| POST | `/incidents` | Not implemented | - |
| PUT | `/incidents` | Implemented | `pdc incident ack`, `resolve`, `reassign`, `urgency`, `title`, TUI |
| GET | `/incidents/{id}` | Implemented | `pdc incident show`, TUI |
| PUT | `/incidents/{id}` | Not implemented | - |
| GET | `/incidents/{id}/business_services/impacts` | Not implemented | - |
| PUT | `/incidents/{id}/business_services/{business_service_id}/impacts` | Not implemented | - |
| GET | `/incidents/{id}/custom_fields/values` | Not implemented | - |
| PUT | `/incidents/{id}/custom_fields/values` | Not implemented | - |
| GET | `/incidents/{id}/log_entries` | Implemented | `pdc incident log`, TUI |
| PUT | `/incidents/{id}/merge` | Implemented | `pdc incident merge`, TUI |
| GET | `/incidents/{id}/notes` | Implemented | `pdc incident note list`, TUI |
| POST | `/incidents/{id}/notes` | Implemented | `pdc incident note add`, `pdc incident resolve --note`, TUI |
| DELETE | `/incidents/{id}/notes/{note_id}` | Not implemented | - |
| PUT | `/incidents/{id}/notes/{note_id}` | Not implemented | - |
| GET | `/incidents/{id}/related_change_events` | Not implemented | - |
| POST | `/incidents/{id}/responder_requests` | Not implemented | - |
| PUT | `/incidents/{id}/responder_requests/cancel` | Not implemented | - |
| POST | `/incidents/{id}/snooze` | Implemented | `pdc incident snooze`, TUI |
| POST | `/incidents/{id}/status_updates` | Not implemented | - |
| GET | `/incidents/{id}/status_updates/subscribers` | Not implemented | - |
| POST | `/incidents/{id}/status_updates/subscribers` | Not implemented | - |
| POST | `/incidents/{id}/status_updates/unsubscribe` | Not implemented | - |

## Incident Alerts

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/incidents/{id}/alerts` | Implemented | `pdc incident show --alerts`, `pdc incident show --payload`, TUI |
| PUT | `/incidents/{id}/alerts` | Implemented | `pdc incident resolve-alert` |
| GET | `/incidents/{id}/alerts/{alert_id}` | Implemented | API only |
| PUT | `/incidents/{id}/alerts/{alert_id}` | Not implemented | - |

## Services

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/services` | Implemented | `pdc service list`, `pdc config init` |
| POST | `/services` | Not implemented | - |
| DELETE | `/services/{id}` | Not implemented | - |
| GET | `/services/{id}` | Implemented | `pdc service show` |
| PUT | `/services/{id}` | Not implemented | - |
| GET | `/services/{id}/audit/records` | Not implemented | - |
| GET | `/services/{id}/change_events` | Not implemented | - |
| GET | `/services/{id}/custom_fields/values` | Not implemented | - |
| PUT | `/services/{id}/custom_fields/values` | Not implemented | - |
| GET | `/services/{id}/enablements` | Not implemented | - |
| PUT | `/services/{id}/enablements/{feature_name}` | Not implemented | - |
| POST | `/services/{id}/integrations` | Not implemented | - |
| GET | `/services/{id}/integrations/{integration_id}` | Not implemented | - |
| PUT | `/services/{id}/integrations/{integration_id}` | Not implemented | - |
| GET | `/services/{id}/rules` | Not implemented | - |
| POST | `/services/{id}/rules` | Not implemented | - |
| POST | `/services/{id}/rules/convert` | Not implemented | - |
| DELETE | `/services/{id}/rules/{rule_id}` | Not implemented | - |
| GET | `/services/{id}/rules/{rule_id}` | Not implemented | - |
| PUT | `/services/{id}/rules/{rule_id}` | Not implemented | - |

The Service Custom Fields endpoints (10 endpoints under `/services/custom_fields/`) are also not implemented.

## Users

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/users` | Implemented | `pdc user list` |
| POST | `/users` | Not implemented | - |
| GET | `/users/me` | Implemented | `pdc user me`, `pdc config init` |
| DELETE | `/users/{id}` | Not implemented | - |
| GET | `/users/{id}` | Implemented | `pdc user show` |
| PUT | `/users/{id}` | Not implemented | - |
| GET | `/users/{id}/audit/records` | Not implemented | - |
| GET | `/users/{id}/contact_methods` | Implemented | API only |
| POST | `/users/{id}/contact_methods` | Not implemented | - |
| DELETE | `/users/{id}/contact_methods/{contact_method_id}` | Not implemented | - |
| GET | `/users/{id}/contact_methods/{contact_method_id}` | Not implemented | - |
| PUT | `/users/{id}/contact_methods/{contact_method_id}` | Not implemented | - |
| GET | `/users/{id}/license` | Not implemented | - |
| GET | `/users/{id}/notification_rules` | Not implemented | - |
| POST | `/users/{id}/notification_rules` | Not implemented | - |
| DELETE | `/users/{id}/notification_rules/{notification_rule_id}` | Not implemented | - |
| GET | `/users/{id}/notification_rules/{notification_rule_id}` | Not implemented | - |
| PUT | `/users/{id}/notification_rules/{notification_rule_id}` | Not implemented | - |
| GET | `/users/{id}/notification_subscriptions` | Not implemented | - |
| POST | `/users/{id}/notification_subscriptions` | Not implemented | - |
| POST | `/users/{id}/notification_subscriptions/unsubscribe` | Not implemented | - |
| GET | `/users/{id}/oauth_delegations` | Not implemented | - |
| GET | `/users/{id}/oauth_delegations/{delegation_id}` | Not implemented | - |
| GET | `/users/{id}/oncall_handoff_notification_rules` | Not implemented | - |
| POST | `/users/{id}/oncall_handoff_notification_rules` | Not implemented | - |
| DELETE | `/users/{id}/oncall_handoff_notification_rules/{oncall_handoff_notification_rule_id}` | Not implemented | - |
| GET | `/users/{id}/oncall_handoff_notification_rules/{oncall_handoff_notification_rule_id}` | Not implemented | - |
| PUT | `/users/{id}/oncall_handoff_notification_rules/{oncall_handoff_notification_rule_id}` | Not implemented | - |
| DELETE | `/users/{id}/sessions` | Not implemented | - |
| GET | `/users/{id}/sessions` | Not implemented | - |
| DELETE | `/users/{id}/sessions/{type}/{session_id}` | Not implemented | - |
| GET | `/users/{id}/sessions/{type}/{session_id}` | Not implemented | - |
| GET | `/users/{id}/status_update_notification_rules` | Not implemented | - |
| POST | `/users/{id}/status_update_notification_rules` | Not implemented | - |
| DELETE | `/users/{id}/status_update_notification_rules/{status_update_notification_rule_id}` | Not implemented | - |
| GET | `/users/{id}/status_update_notification_rules/{status_update_notification_rule_id}` | Not implemented | - |
| PUT | `/users/{id}/status_update_notification_rules/{status_update_notification_rule_id}` | Not implemented | - |

## Teams

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/teams` | Implemented | `pdc team list`, `pdc config init`, TUI |
| POST | `/teams` | Not implemented | - |
| DELETE | `/teams/{id}` | Not implemented | - |
| GET | `/teams/{id}` | Implemented | `pdc team show` |
| PUT | `/teams/{id}` | Not implemented | - |
| GET | `/teams/{id}/audit/records` | Not implemented | - |
| DELETE | `/teams/{id}/escalation_policies/{escalation_policy_id}` | Not implemented | - |
| PUT | `/teams/{id}/escalation_policies/{escalation_policy_id}` | Not implemented | - |
| GET | `/teams/{id}/members` | Implemented | `pdc oncall --team` |
| GET | `/teams/{id}/notification_subscriptions` | Not implemented | - |
| POST | `/teams/{id}/notification_subscriptions` | Not implemented | - |
| POST | `/teams/{id}/notification_subscriptions/unsubscribe` | Not implemented | - |
| DELETE | `/teams/{id}/users/{user_id}` | Not implemented | - |
| PUT | `/teams/{id}/users/{user_id}` | Not implemented | - |

## Schedules

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/schedules` | Implemented | `pdc schedule list` |
| POST | `/schedules` | Not implemented | - |
| POST | `/schedules/preview` | Not implemented | - |
| DELETE | `/schedules/{id}` | Not implemented | - |
| GET | `/schedules/{id}` | Implemented | `pdc schedule show` |
| PUT | `/schedules/{id}` | Not implemented | - |
| GET | `/schedules/{id}/audit/records` | Not implemented | - |
| GET | `/schedules/{id}/overrides` | Implemented | API only |
| POST | `/schedules/{id}/overrides` | Implemented | `pdc schedule override` |
| DELETE | `/schedules/{id}/overrides/{override_id}` | Not implemented | - |
| GET | `/schedules/{id}/users` | Not implemented | - |

## On-Calls

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/oncalls` | Implemented | `pdc oncall`, `pdc incident list --schedule` |

## Escalation Policies

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/escalation_policies` | Implemented | `pdc escalation-policy list`, TUI |
| POST | `/escalation_policies` | Not implemented | - |
| DELETE | `/escalation_policies/{id}` | Not implemented | - |
| GET | `/escalation_policies/{id}` | Implemented | `pdc escalation-policy show`, TUI |
| PUT | `/escalation_policies/{id}` | Not implemented | - |
| GET | `/escalation_policies/{id}/audit/records` | Not implemented | - |

## Priorities

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/priorities` | Implemented | TUI |

## Abilities

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/abilities` | Implemented | `pdc ability list`, `pdc config init` |
| GET | `/abilities/{id}` | Implemented | `pdc ability test` |

## Maintenance Windows

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/maintenance_windows` | Implemented | `pdc maintenance-window list` |
| POST | `/maintenance_windows` | Not implemented | - |
| DELETE | `/maintenance_windows/{id}` | Not implemented | - |
| GET | `/maintenance_windows/{id}` | Implemented | `pdc maintenance-window show` |
| PUT | `/maintenance_windows/{id}` | Not implemented | - |

## Log Entries (top-level)

| Method | Path | Status | Used by |
|---|---|---|---|
| GET | `/log_entries` | Not implemented | - |
| GET | `/log_entries/{id}` | Not implemented | - |
| PUT | `/log_entries/{id}/channel` | Not implemented | - |

## Unimplemented Resource Groups

The following resource groups have no coverage.
Endpoints marked with a plan tier note require that tier or add-on.

### Add-ons

5 endpoints. CRUD for UI add-ons.

### Analytics

16 endpoints. Requires Advanced Analytics (Professional plan or above).

### Audit Records

1 endpoint. `GET /audit/records` lists account-level records.

### Automation Actions

25 endpoints. Requires PagerDuty Automation add-on.

### Business Services

16 endpoints.

### Change Events

4 endpoints.

### Event Orchestrations

39 endpoints.

### Extension Schemas

2 endpoints (read-only).

### Extensions

6 endpoints.

### Incident Custom Fields

9 endpoints (under `/incidents/custom_fields/`).

### Incident Types

14 endpoints (under `/incidents/types/`).

### Incident Workflows

15 endpoints.

### Licenses

2 endpoints (read-only). Includes `GET /licenses` and `GET /license_allocations`.

### Notifications

1 endpoint (read-only).

### OAuth Delegations

2 endpoints.

### Rulesets (deprecated)

10 endpoints. Deprecated in favour of Event Orchestrations.

### Service Custom Fields

10 endpoints (under `/services/custom_fields/`).

### Service Dependencies

4 endpoints.

### Session Configurations

3 endpoints.

### SRE Agent

3 endpoints.

### Standards

4 endpoints.

### Status Dashboards

5 endpoints (read-only).

### Status Pages

27 endpoints.

### Tags

7 endpoints. Includes 2 generic `/{entity_type}/{id}` endpoints for tagging any resource.

### Templates

7 endpoints.

### Vendors

2 endpoints (read-only).

### Webhooks

12 endpoints.

### Workflow Integrations

8 endpoints.

### v3 Schedules

24 endpoints. New schedule model with rotations, custom shifts and events.
