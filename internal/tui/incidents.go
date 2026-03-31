package tui

import "github.com/PagerDuty/go-pagerduty"

// IncidentAcked is sent when an acknowledge action completes.
type IncidentAcked struct{ ID string }

// IncidentResolved is sent when a resolve action completes.
type IncidentResolved struct{ ID string }

// IncidentEscalated is sent when an escalate action completes.
type IncidentEscalated struct{ ID string }

// IncidentMerged is sent when a merge action completes.
type IncidentMerged struct{ TargetID string }

// IncidentPriorityUpdated is sent when a priority update completes.
type IncidentPriorityUpdated struct{ ID string }

// IncidentUpdated is sent when a multi-field edit completes.
type IncidentUpdated struct{ ID string }

// IncidentNoteAdded is sent after a note is successfully added.
type IncidentNoteAdded struct{ ID string }

// IncidentReassigned is sent after an incident is successfully reassigned.
type IncidentReassigned struct{ ID string }

// IncidentSnoozed is sent after an incident is successfully snoozed.
type IncidentSnoozed struct{ ID string }

// IncidentSelected is sent when the user opens an incident detail view.
type IncidentSelected struct{ Incident pagerduty.Incident }

// incidentErrMsg carries an error from an async incident action.
type incidentErrMsg struct {
	op  string
	err error
}

// showInputMsg requests that the App show the text input overlay for an
// incident action.
type showInputMsg struct {
	action      string
	incidentID  string
	prompt      string
	placeholder string
}

// statusMsg is a simple feedback message displayed in the status bar.
type statusMsg string

// batchResultMsg carries the outcome of a batch ack/resolve operation.
type batchResultMsg struct {
	op       string
	success  int
	failures int
	firstErr error
}

// detailAckMsg is sent from the detail view when the user presses a.
type detailAckMsg struct {
	id string
}

// detailResolveMsg is sent from the detail view when the user presses r or alt+r.
type detailResolveMsg struct {
	id      string
	confirm bool
}

// detailEditMsg is sent from the detail view when the user presses e.
type detailEditMsg struct {
	incident pagerduty.Incident
}

// detailEscalateMsg is sent from the detail view when the user presses E or alt+e.
type detailEscalateMsg struct {
	id      string
	confirm bool
}

// detailSetPriorityMsg is sent from the detail view when the user presses p.
type detailSetPriorityMsg struct {
	id string
}

// detailNoteMsg is sent from the detail view when the user presses n.
type detailNoteMsg struct {
	id string
}

// detailSnoozeMsg is sent from the detail view when the user presses s.
type detailSnoozeMsg struct {
	id string
}
