package tui

import (
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/testutil"
)

func testIncidents() []pagerduty.Incident {
	return testutil.MustLoadIncidents()
}
