package tui

import (
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/testutil"
)

func testIncidents() []pagerduty.Incident {
	return testutil.MustLoadIncidents()
}

func testAlerts(incidentID string) []pagerduty.IncidentAlert {
	return testutil.MustLoadAlerts(incidentID)
}

func testNotes(incidentID string) []pagerduty.IncidentNote {
	return testutil.MustLoadNotes(incidentID)
}
