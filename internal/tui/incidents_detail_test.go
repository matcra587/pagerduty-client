package tui

import (
	"context"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBodySection_UsesNormaliser_GCP(t *testing.T) {
	m := incidentDetail{
		incident: testIncidents()[0],
		alerts:   []pagerduty.IncidentAlert{{Body: testAlertBody()}},
		width: 80,
	}
	body := m.bodySection()
	assert.Contains(t, body, "Google Cloud Monitoring")
	assert.Contains(t, body, "Uptime failure")
	assert.Contains(t, body, "Check failed")
	assert.NotContains(t, body, "map[")
}

func TestIncidentDetailUpdate_AlertsLoadedSyncsSummaryAndAlertsOnly(t *testing.T) {
	m := newTestIncidentDetail()
	before := setViewportSentinels(&m)

	result, _ := m.Update(alertsLoadedMsg{
		incidentID: m.incident.ID,
		alerts: []pagerduty.IncidentAlert{{
			APIObject: pagerduty.APIObject{Summary: "CPU high"},
			AlertKey:  "alert-1",
			Status:    "triggered",
			Body:      testAlertBody(),
		}},
	})
	updated := result.(incidentDetail)

	after := viewportViews(updated)
	assert.NotEqual(t, before[tabSummary], after[tabSummary])
	assert.NotEqual(t, before[tabAlerts], after[tabAlerts])
	assert.Equal(t, before[tabNotes], after[tabNotes])
	assert.Equal(t, before[tabTimeline], after[tabTimeline])
}

func TestIncidentDetailUpdate_NotesLoadedSyncsNotesOnly(t *testing.T) {
	m := newTestIncidentDetail()
	before := setViewportSentinels(&m)

	result, _ := m.Update(notesLoadedMsg{
		incidentID: m.incident.ID,
		notes: []pagerduty.IncidentNote{{
			Content:   "Investigating",
			CreatedAt: time.Date(2026, 4, 14, 20, 0, 0, 0, time.UTC).Format(time.RFC3339),
			User:      pagerduty.APIObject{Summary: "Alice"},
		}},
	})
	updated := result.(incidentDetail)

	after := viewportViews(updated)
	assert.Equal(t, before[tabSummary], after[tabSummary])
	assert.Equal(t, before[tabAlerts], after[tabAlerts])
	assert.NotEqual(t, before[tabNotes], after[tabNotes])
	assert.Equal(t, before[tabTimeline], after[tabTimeline])
}

func TestIncidentDetailUpdate_LogEntriesLoadedSyncsTimelineOnly(t *testing.T) {
	m := newTestIncidentDetail()
	before := setViewportSentinels(&m)

	result, _ := m.Update(logEntriesLoadedMsg{
		incidentID: m.incident.ID,
		entries: []pagerduty.LogEntry{{
			CommonLogEntryField: pagerduty.CommonLogEntryField{
				APIObject: pagerduty.APIObject{Type: "annotate_log_entry"},
				CreatedAt: time.Date(2026, 4, 14, 20, 5, 0, 0, time.UTC).Format(time.RFC3339),
				Agent:     pagerduty.Agent{Summary: "system"},
			},
		}},
	})
	updated := result.(incidentDetail)

	after := viewportViews(updated)
	assert.Equal(t, before[tabSummary], after[tabSummary])
	assert.Equal(t, before[tabAlerts], after[tabAlerts])
	assert.Equal(t, before[tabNotes], after[tabNotes])
	assert.NotEqual(t, before[tabTimeline], after[tabTimeline])
}

func newTestIncidentDetail() incidentDetail {
	m := newIncidentDetail(context.Background(), nil, config.Default(), nil, testIncidents()[0])
	m.setSize(100, 20)
	return m
}

func setViewportSentinels(m *incidentDetail) [tabCount]string {
	m.viewports[tabSummary].SetContent("summary sentinel")
	m.viewports[tabAlerts].SetContent("alerts sentinel")
	m.viewports[tabNotes].SetContent("notes sentinel")
	m.viewports[tabTimeline].SetContent("timeline sentinel")
	return viewportViews(*m)
}

func viewportViews(m incidentDetail) [tabCount]string {
	return [tabCount]string{
		m.viewports[tabSummary].View(),
		m.viewports[tabAlerts].View(),
		m.viewports[tabNotes].View(),
		m.viewports[tabTimeline].View(),
	}
}

func testAlertBody() map[string]any {
	return map[string]any{
		"details": map[string]any{
			"custom_details": map[string]any{
				"incident": map[string]any{
					"policy_name":     "Uptime failure",
					"condition_name":  "Check failed",
					"condition":       map[string]any{},
					"observed_value":  "2.0",
					"threshold_value": "1",
					"state":           "open",
					"url":             "https://console.cloud.google.com/test",
				},
				"version": "1.2",
			},
		},
	}
}
