package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/gechr/clib/ansi"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/integration"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// alertsLoadedMsg carries the fetched alerts for the current incident.
type alertsLoadedMsg struct {
	incidentID string
	alerts     []pagerduty.IncidentAlert
	err        error
}

// notesLoadedMsg carries the fetched notes for the current incident.
type notesLoadedMsg struct {
	incidentID string
	notes      []pagerduty.IncidentNote
	err        error
}

// detailTab identifies which tab is active in the detail view.
type detailTab int

const (
	tabSummary detailTab = iota
	tabAlerts
	tabNotes
	tabCount
)

// incidentDetail is a Bubble Tea model rendering full incident information
// split across three tabs: Summary, Alerts and Notes.
type incidentDetail struct {
	ctx          context.Context //nolint:containedctx // Bubble Tea models are value-typed; context must travel with the model.
	incident     pagerduty.Incident
	alerts       []pagerduty.IncidentAlert
	notes        []pagerduty.IncidentNote
	loading      bool
	notesLoading bool
	err          error
	notesErr     error
	width        int
	height       int
	client       *api.Client
	cfg          *config.Config
	ansi         *ansi.ANSI
	activeTab    detailTab
	viewports    [tabCount]viewport.Model
}

func newIncidentDetail(ctx context.Context, client *api.Client, cfg *config.Config, a *ansi.ANSI, inc pagerduty.Incident) incidentDetail {
	var vps [tabCount]viewport.Model
	for i := range vps {
		vps[i] = viewport.New()
		vps[i].SoftWrap = true
	}
	return incidentDetail{
		ctx:          ctx,
		client:       client,
		cfg:          cfg,
		ansi:         a,
		incident:     inc,
		loading:      true,
		notesLoading: true,
		viewports:    vps,
	}
}

// Init implements tea.Model.
func (m incidentDetail) Init() tea.Cmd {
	return tea.Batch(m.fetchAlertsCmd(), m.fetchNotesCmd())
}

// Update implements tea.Model.
func (m incidentDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := max(msg.Height-4, 1)
		for i := range m.viewports {
			m.viewports[i].SetWidth(msg.Width)
			m.viewports[i].SetHeight(vpHeight)
		}
		m.syncContent()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % tabCount
			return m, nil
		case "shift+tab", "left":
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			return m, nil
		case "r":
			return m, func() tea.Msg {
				return detailResolveMsg{id: m.incident.ID, confirm: true}
			}
		case "alt+r":
			return m, func() tea.Msg {
				return detailResolveMsg{id: m.incident.ID, confirm: false}
			}
		case "p":
			return m, func() tea.Msg {
				return detailSetPriorityMsg{id: m.incident.ID}
			}
		case "e":
			return m, func() tea.Msg {
				return detailEscalateMsg{id: m.incident.ID, confirm: true}
			}
		case "alt+e":
			return m, func() tea.Msg {
				return detailEscalateMsg{id: m.incident.ID, confirm: false}
			}
		case "a":
			return m, func() tea.Msg {
				return detailAckMsg{id: m.incident.ID}
			}
		default:
			var cmd tea.Cmd
			m.viewports[m.activeTab], cmd = m.viewports[m.activeTab].Update(msg)
			return m, cmd
		}

	case alertsLoadedMsg:
		if msg.incidentID == m.incident.ID {
			m.loading = false
			m.alerts = msg.alerts
			m.err = msg.err
			m.syncContent()
		}

	case notesLoadedMsg:
		if msg.incidentID == m.incident.ID {
			m.notesLoading = false
			m.notes = msg.notes
			m.notesErr = msg.err
			m.syncContent()
		}
	}
	return m, nil
}

func (m *incidentDetail) syncContent() {
	m.viewports[tabSummary].SetContent(m.summaryView())
	m.viewports[tabAlerts].SetContent(m.alertsSection())
	m.viewports[tabNotes].SetContent(m.notesSection())
}

// View implements tea.Model.
func (m incidentDetail) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	header := theme.Title.Render("Incident: " + m.incident.ID)
	return tea.NewView(header + "\n" + m.tabBar() + "\n" + m.viewports[m.activeTab].View())
}

func (m incidentDetail) tabBar() string {
	type tab struct {
		name  string
		count int
	}
	tabs := []tab{
		{"Summary", 0},
		{"Alerts", len(m.alerts)},
		{"Notes", len(m.notes)},
	}

	active := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorTitleFg).
		Background(theme.ColorHighlightBg).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(theme.ColorHeaderFg).
		Padding(0, 1)

	var parts []string
	for i, t := range tabs {
		label := t.name
		if t.count > 0 {
			label += fmt.Sprintf(" (%d)", t.count)
		}
		if detailTab(i) == m.activeTab {
			parts = append(parts, active.Render(label))
		} else {
			parts = append(parts, inactive.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

func (m incidentDetail) summaryView() string {
	inc := m.incident

	const indent = "  "
	const labelW = 13

	lbl := func(name string) string {
		return theme.DetailLabel.Render(fmt.Sprintf("%*s:", labelW, name))
	}
	field := func(name, value string) string {
		return indent + lbl(name) + " " + theme.DetailValue.Render(value) + "\n"
	}

	var sb strings.Builder

	sb.WriteString(theme.DetailHeader.Render("Overview"))
	sb.WriteString("\n\n")

	sb.WriteString(field("Title", inc.Title))
	sb.WriteString(field("Status", statusText(inc.Status)))
	sb.WriteString(field("Priority", styledPriorityLabel(inc)))
	sb.WriteString(field("Service", inc.Service.Summary))

	if inc.AlertCounts.All > 1 {
		alertSummary := fmt.Sprintf("%d total, %d triggered, %d resolved",
			inc.AlertCounts.All, inc.AlertCounts.Triggered, inc.AlertCounts.Resolved)
		sb.WriteString(field("Alerts", alertSummary))
	}

	names := assigneeNames(inc.Assignments)
	if names != "" {
		sb.WriteString(indent + lbl("Assignees") + " " + theme.EntityColor(names).Render(names) + "\n")
	}

	created := renderTimeAgo(inc.CreatedAt) + " " + theme.DetailDim.Render("("+formatTimeAbsolute(inc.CreatedAt)+")")
	updated := renderTimeAgo(inc.LastStatusChangeAt) + " " + theme.DetailDim.Render("("+formatTimeAbsolute(inc.LastStatusChangeAt)+")")
	timeline := indent + lbl("Timeline") + " created " + created + " " + theme.DetailDim.Render("-") + " updated " + updated
	sb.WriteString(timeline + "\n")

	sb.WriteString(indent + lbl("ID") + " " + theme.DetailDim.Render(m.ansi.Hyperlink(inc.HTMLURL, inc.ID)) + "\n")

	if inc.IncidentKey != "" {
		sb.WriteString(field("Event key", inc.IncidentKey))
	}
	if inc.Description != "" && inc.Description != inc.Title {
		if strings.Contains(inc.Description, "\n") || len(inc.Description) > 80 {
			sb.WriteString("\n")
			sb.WriteString(theme.DetailHeader.Render("Description"))
			sb.WriteString("\n")
			sb.WriteString(renderMarkdown(inc.Description, m.width))
			sb.WriteString("\n")
		} else {
			sb.WriteString(field("Description", inc.Description))
		}
	}
	if inc.ConferenceBridge != nil && inc.ConferenceBridge.ConferenceURL != "" {
		sb.WriteString(field("Bridge URL", inc.ConferenceBridge.ConferenceURL))
	}
	if inc.ConferenceBridge != nil && inc.ConferenceBridge.ConferenceNumber != "" {
		sb.WriteString(field("Bridge num", inc.ConferenceBridge.ConferenceNumber))
	}

	sb.WriteString("\n")

	if body := m.bodySection(); body != "" {
		sb.WriteString(body)
	}

	return sb.String()
}

func (m incidentDetail) alertBody() map[string]any {
	if len(m.alerts) == 0 {
		return nil
	}
	return m.alerts[0].Body
}

func (m incidentDetail) bodySection() string {
	var sb strings.Builder

	if m.incident.Body.Details != "" {
		sb.WriteString(theme.DetailHeader.Render("Details"))
		sb.WriteString("\n")
		sb.WriteString(renderMarkdown(m.incident.Body.Details, m.width))
		sb.WriteString("\n")
	}

	body := m.alertBody()
	if body == nil {
		return sb.String()
	}

	if m.cfg != nil && len(m.cfg.CustomFields) > 0 {
		sb.WriteString(m.configuredFieldsView(body))
		return sb.String()
	}

	summary := integration.Detect(body)

	if len(summary.Fields) == 0 && len(summary.Links) == 0 {
		return sb.String()
	}

	if summary.Source != "Unknown" {
		sb.WriteString(theme.DetailHeader.Render(summary.Source))
	} else {
		sb.WriteString(theme.DetailLabel.Render("Body:"))
	}
	sb.WriteString("\n\n")

	maxLabel := 0
	for _, f := range summary.Fields {
		if w := len(f.Label) + 1; w > maxLabel {
			maxLabel = w
		}
	}
	for _, l := range summary.Links {
		if w := len(l.Label) + 1; w > maxLabel {
			maxLabel = w
		}
	}
	for _, f := range summary.Fields {
		label := fmt.Sprintf("%-*s", maxLabel, f.Label+":")
		if strings.Contains(f.Value, "\n") || len(f.Value) > 80 {
			fmt.Fprintf(&sb, "    %s\n", theme.DetailLabel.Render(label))
			sb.WriteString(renderMarkdown(f.Value, m.width-4))
			sb.WriteString("\n")
		} else {
			fmt.Fprintf(&sb, "    %s %s\n",
				theme.DetailLabel.Render(label),
				theme.DetailValue.Render(f.Value),
			)
		}
	}
	if len(summary.Links) > 0 {
		sb.WriteString("\n")
		for _, l := range summary.Links {
			label := fmt.Sprintf("%-*s", maxLabel, l.Label+":")
			fmt.Fprintf(&sb, "    %s %s\n",
				theme.DetailLabel.Render(label),
				theme.DetailValue.Render(l.URL),
			)
		}
	}

	return sb.String()
}

func (m incidentDetail) configuredFieldsView(body map[string]any) string {
	var sb strings.Builder
	wrote := false
	for _, f := range m.cfg.CustomFields {
		val, ok := resolveFieldPath(body, f.Path)
		if !ok || val == nil {
			continue
		}
		wrote = true
		v := integration.FormatValue(val)
		switch f.Display {
		case "block":
			sb.WriteString("\n")
			sb.WriteString(theme.DetailHeader.Render(f.Label))
			sb.WriteString("\n")
			sb.WriteString(renderMarkdown(v, m.width))
		default:
			sb.WriteString(fmt.Sprintf("    %s %s\n",
				theme.DetailLabel.Render(fmt.Sprintf("%-16s", f.Label+":")),
				theme.DetailValue.Render(v),
			))
		}
	}
	if !wrote {
		return ""
	}
	return sb.String()
}

func resolveFieldPath(body map[string]any, path string) (any, bool) {
	if val, ok := walkPath(body, path); ok {
		return val, true
	}
	if cef, ok := body["cef_details"].(map[string]any); ok {
		return walkPath(cef, path)
	}
	return nil, false
}

func walkPath(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = root
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func (m incidentDetail) alertsSection() string {
	if m.loading {
		return "\n" + theme.DetailDim.Render("  Loading alerts...") + "\n"
	}
	if m.err != nil {
		return "\n" + theme.DetailDim.Render(fmt.Sprintf("  Error loading alerts: %v", m.err)) + "\n"
	}
	if len(m.alerts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(theme.DetailHeader.Render("Alerts"))
	sb.WriteString("\n\n")

	for _, a := range m.alerts {
		status := statusText(a.Status)
		key := theme.DetailValue.Render(truncate(a.AlertKey, 30))
		summaryW := max(m.width-38, 0)
		summary := truncate(a.APIObject.Summary, summaryW)
		sb.WriteString(fmt.Sprintf("  %s %s  %s\n", status, key, summary))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m incidentDetail) notesSection() string {
	if m.notesLoading {
		return "\n" + theme.DetailDim.Render("  Loading notes...") + "\n"
	}
	if m.notesErr != nil {
		return "\n" + theme.DetailDim.Render(fmt.Sprintf("  Error loading notes: %v", m.notesErr)) + "\n"
	}
	if len(m.notes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(theme.DetailHeader.Render("Notes"))
	sb.WriteString("\n\n")

	for i, n := range m.notes {
		author := theme.HelpKey.Render(n.User.Summary)
		when := theme.DetailDim.Render(renderTimeAgo(n.CreatedAt))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", author, when))
		sb.WriteString(renderMarkdown(n.Content, m.width-2))
		sb.WriteString("\n")
		if i < len(m.notes)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m incidentDetail) fetchNotesCmd() tea.Cmd {
	incID := m.incident.ID
	if m.cfg != nil && m.cfg.TestMode {
		return func() tea.Msg {
			return notesLoadedMsg{incidentID: incID, notes: testNotes(incID)}
		}
	}
	client := m.client
	detailCtx := m.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(detailCtx, 15*time.Second)
		defer cancel()
		notes, err := client.ListIncidentNotes(ctx, incID)
		return notesLoadedMsg{incidentID: incID, notes: notes, err: err}
	}
}

func (m incidentDetail) fetchAlertsCmd() tea.Cmd {
	incID := m.incident.ID
	if m.cfg != nil && m.cfg.TestMode {
		return func() tea.Msg {
			return alertsLoadedMsg{incidentID: incID, alerts: testAlerts(incID)}
		}
	}
	client := m.client
	detailCtx := m.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(detailCtx, 15*time.Second)
		defer cancel()
		alerts, err := client.ListIncidentAlerts(ctx, incID)
		return alertsLoadedMsg{incidentID: incID, alerts: alerts, err: err}
	}
}
