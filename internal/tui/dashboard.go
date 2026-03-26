package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/api"
)

// Dashboard is the root view model. It owns the incident list panel.
type Dashboard struct {
	incidents incidentList
	width     int
	height    int
}

func newDashboard(ctx context.Context, client *api.Client, fromEmail string, hideService bool) Dashboard {
	return Dashboard{
		incidents: newIncidentList(ctx, client, fromEmail, hideService),
	}
}

// Init implements tea.Model.
func (d Dashboard) Init() tea.Cmd { return nil }

// SetIncidents updates the incident list with fresh data.
func (d *Dashboard) SetIncidents(incidents []pagerduty.Incident) {
	d.incidents.SetIncidents(incidents)
}

// FilterActive reports whether the incident list filter input is focused.
func (d Dashboard) FilterActive() bool {
	return d.incidents.FilterActive()
}

// Update implements tea.Model.
func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		d.width = ws.Width
		d.height = ws.Height
		d.incidents.width = ws.Width
		d.incidents.height = d.height
	}

	im, iCmd := d.incidents.Update(msg)
	d.incidents = im.(incidentList)

	return d, iCmd
}

// View implements tea.Model.
func (d Dashboard) View() tea.View {
	if d.width == 0 {
		return tea.NewView("")
	}
	return tea.NewView(d.incidents.View().Content)
}
