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
	"github.com/matcra587/pagerduty-client/internal/output"
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

// logEntriesLoadedMsg carries the fetched log entries for the current incident.
type logEntriesLoadedMsg struct {
	incidentID string
	entries    []pagerduty.LogEntry
	err        error
}

// detailTab identifies which tab is active in the detail view.
type detailTab int

const (
	tabSummary detailTab = iota
	tabAlerts
	tabNotes
	tabTimeline
	tabCount
)

// incidentDetail is a Bubble Tea model rendering full incident information
// split across five tabs: Summary, Alerts, Notes, Timeline and Related.
type incidentDetail struct {
	ctx             context.Context //nolint:containedctx // Bubble Tea models are value-typed; context must travel with the model.
	incident        pagerduty.Incident
	alerts          []pagerduty.IncidentAlert
	notes           []pagerduty.IncidentNote
	logEntries      []pagerduty.LogEntry
	loading         bool
	notesLoading    bool
	timelineLoading bool
	err             error
	notesErr        error
	timelineErr     error
	width           int
	height          int
	client          *api.Client
	cfg             *config.Config
	ansi            *ansi.ANSI
	activeTab       detailTab
	viewports       [tabCount]viewport.Model
}

// setSize updates the detail model and all viewports to the given dimensions.
func (m *incidentDetail) setSize(width, height int) {
	m.width = width
	m.height = height
	vpHeight := max(height, 1)
	for i := range m.viewports {
		m.viewports[i].SetWidth(width)
		m.viewports[i].SetHeight(vpHeight)
	}
}

// scroll applies an accumulated mouse wheel delta to the active viewport.
// Positive delta scrolls down, negative scrolls up.
func (m *incidentDetail) scroll(delta int) {
	vp := &m.viewports[m.activeTab]
	if delta > 0 {
		vp.ScrollDown(delta)
	} else {
		vp.ScrollUp(-delta)
	}
}

func newIncidentDetail(ctx context.Context, client *api.Client, cfg *config.Config, a *ansi.ANSI, inc pagerduty.Incident) incidentDetail {
	var vps [tabCount]viewport.Model
	for i := range vps {
		vps[i] = viewport.New()
		vps[i].SoftWrap = true
	}
	return incidentDetail{
		ctx:             ctx,
		client:          client,
		cfg:             cfg,
		ansi:            a,
		incident:        inc,
		loading:         true,
		notesLoading:    true,
		timelineLoading: true,
		viewports:       vps,
	}
}

// Init implements tea.Model.
func (m incidentDetail) Init() tea.Cmd {
	return tea.Batch(m.fetchAlertsCmd(), m.fetchNotesCmd(), m.fetchLogEntriesCmd())
}

// Update implements tea.Model.
func (m incidentDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
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
				return detailEditMsg{incident: m.incident}
			}
		case "x":
			return m, func() tea.Msg {
				return detailEscalateMsg{id: m.incident.ID, confirm: true}
			}
		case "alt+x":
			return m, func() tea.Msg {
				return detailEscalateMsg{id: m.incident.ID, confirm: false}
			}
		case "a":
			return m, func() tea.Msg {
				return detailAckMsg{id: m.incident.ID}
			}
		case "n":
			return m, func() tea.Msg {
				return detailNoteMsg{id: m.incident.ID}
			}
		case "s":
			return m, func() tea.Msg {
				return detailSnoozeMsg{id: m.incident.ID}
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

	case logEntriesLoadedMsg:
		if msg.incidentID == m.incident.ID {
			m.timelineLoading = false
			m.logEntries = msg.entries
			m.timelineErr = msg.err
			m.syncContent()
		}
	}
	return m, nil
}

func (m *incidentDetail) syncContent() {
	m.viewports[tabSummary].SetContent(m.summaryView())
	m.viewports[tabAlerts].SetContent(m.alertsSection())
	m.viewports[tabNotes].SetContent(m.notesSection())
	m.viewports[tabTimeline].SetContent(m.timelineSection())
}

// View implements tea.Model.
func (m incidentDetail) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}
	return tea.NewView(m.viewports[m.activeTab].View())
}

// headerContent returns the detail tab bar with incident ID on the right.
func (m incidentDetail) headerContent() string {
	tabs := m.tabBar()
	id := m.incident.ID
	if m.ansi != nil {
		id = m.ansi.Hyperlink(m.incident.HTMLURL, id)
	}
	idW := lipgloss.Width(id)
	tabsW := lipgloss.Width(tabs)
	gap := max(m.width-tabsW-idW, 1)
	return tabs + fmt.Sprintf("%*s", gap, "") + id
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
		{"Timeline", len(m.logEntries)},
	}

	active := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorTitleFg).
		Underline(true).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(theme.ColorHeaderFg).
		Faint(true).
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
		return indent + lbl(name) + " " + theme.DetailValue.Render(output.Sanitize(value)) + "\n"
	}

	var sb strings.Builder

	sb.WriteString(theme.DetailHeader.Render("Overview"))
	sb.WriteString("\n\n")

	sb.WriteString(field("Title", inc.Title))
	sb.WriteString(indent + lbl("Status") + " " + statusText(inc.Status) + "\n")
	sb.WriteString(indent + lbl("Priority") + " " + styledPriorityLabel(inc) + "\n")
	sb.WriteString(field("Service", inc.Service.Summary))
	if inc.EscalationPolicy.Summary != "" {
		sb.WriteString(field("Escalation", inc.EscalationPolicy.Summary))
	}

	if inc.AlertCounts.All > 1 {
		alertSummary := fmt.Sprintf("%d total, %d triggered, %d resolved",
			inc.AlertCounts.All, inc.AlertCounts.Triggered, inc.AlertCounts.Resolved)
		sb.WriteString(field("Alerts", alertSummary))
	}

	names := assigneeNames(inc.Assignments)
	if names != "" {
		sanitized := output.Sanitize(names)
		sb.WriteString(indent + lbl("Assignees") + " " + theme.EntityColor(names).Render(sanitized) + "\n")
	}

	created := renderTimeAgo(inc.CreatedAt) + " " + theme.DetailDim.Render("("+formatTimeAbsolute(inc.CreatedAt)+")")
	updated := renderTimeAgo(inc.LastStatusChangeAt) + " " + theme.DetailDim.Render("("+formatTimeAbsolute(inc.LastStatusChangeAt)+")")
	timeline := indent + lbl("Timeline") + " created " + created + " " + theme.DetailDim.Render("-") + " updated " + updated
	sb.WriteString(timeline + "\n")

	if inc.IncidentKey != "" {
		sb.WriteString(field("Event key", inc.IncidentKey))
	}
	if inc.Description != "" && inc.Description != inc.Title {
		if strings.Contains(inc.Description, "\n") || len(inc.Description) > 80 {
			sb.WriteString("\n")
			sb.WriteString(theme.DetailHeader.Render("Description"))
			sb.WriteString("\n")
			sb.WriteString(renderMarkdown(output.Sanitize(inc.Description), m.width))
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

	// Incident body details (always rendered when present - normalisers
	// are best-effort and may miss data in Body.Details).
	if m.incident.Body.Details != "" {
		sb.WriteString(theme.DetailHeader.Render("Details"))
		sb.WriteString("\n")
		sb.WriteString(renderMarkdown(output.Sanitize(m.incident.Body.Details), m.width))
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

	// Title dedup: drop integration title when it matches the incident
	// title. Promote differing titles to code blocks for visibility.
	incTitle := m.incident.Title
	var filtered []integration.Field
	for _, f := range summary.Fields {
		if f.Value == "" {
			continue
		}
		if f.Label == "Title" {
			if strings.EqualFold(strings.TrimSpace(f.Value), strings.TrimSpace(incTitle)) {
				continue
			}
			f.Type = integration.FieldCode
		}
		filtered = append(filtered, f)
	}

	groups := groupFieldsByType(filtered)

	// 1. Header row: source + badges + links.
	badges := groups[integration.FieldBadge]
	if len(badges) > 0 {
		sb.WriteString(renderHeaderRow(summary.Source, badges, summary.Links, m.ansi))
		sb.WriteString("\n\n")
	} else if summary.Source != "Unknown" {
		sb.WriteString(theme.DetailHeader.Render(summary.Source))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(theme.DetailLabel.Render("Body:"))
		sb.WriteString("\n\n")
	}

	// 2. Text fields: standard label-value pairs.
	// When badges are absent, links render inline below text fields.
	textFields := groups[integration.FieldText]
	inlineLinks := len(badges) == 0
	if len(textFields) > 0 || inlineLinks {
		maxLabel := 0
		for _, f := range textFields {
			if w := len(f.Label) + 1; w > maxLabel {
				maxLabel = w
			}
		}
		if inlineLinks {
			for _, l := range summary.Links {
				if w := len(l.Label) + 1; w > maxLabel {
					maxLabel = w
				}
			}
		}
		for _, f := range textFields {
			label := fmt.Sprintf("%-*s", maxLabel, f.Label+":")
			val := output.Sanitize(f.Value)
			if strings.Contains(val, "\n") {
				fmt.Fprintf(&sb, "    %s\n", theme.DetailLabel.Render(label))
				sb.WriteString(renderMarkdown(val, m.width-4))
				sb.WriteString("\n")
			} else if len(val) > 80 {
				fmt.Fprintf(&sb, "    %s\n", theme.DetailLabel.Render(label))
				sb.WriteString(theme.DetailDim.Render(wordWrap(val, m.width-6, "      ")) + "\n")
			} else {
				fmt.Fprintf(&sb, "    %s %s\n",
					theme.DetailLabel.Render(label),
					theme.DetailValue.Render(val),
				)
			}
		}
		if inlineLinks {
			for _, l := range summary.Links {
				label := fmt.Sprintf("%-*s", maxLabel, l.Label+":")
				linkLabel := l.URL
				if m.ansi != nil {
					linkLabel = m.ansi.Hyperlink(l.URL, l.URL)
				}
				fmt.Fprintf(&sb, "    %s %s\n",
					theme.DetailLabel.Render(label),
					theme.DetailValue.Render(linkLabel),
				)
			}
		}
	}

	// 3. Code blocks.
	for _, f := range groups[integration.FieldCode] {
		f.Value = output.Sanitize(f.Value)
		sb.WriteString(renderCodeBlock(f, m.width))
		sb.WriteString("\n")
	}

	// 4. Markdown fields.
	for _, f := range groups[integration.FieldMarkdown] {
		f.Value = output.Sanitize(f.Value)
		sb.WriteString(renderMarkdownField(f, m.width))
		sb.WriteString("\n")
	}

	// 5. Tags (after separator).
	tagFields := groups[integration.FieldTags]
	if len(tagFields) > 0 {
		sb.WriteString(theme.DetailDim.Render(strings.Repeat("─", m.width)))
		sb.WriteString("\n")
		for _, f := range tagFields {
			f.Value = output.Sanitize(f.Value)
			sb.WriteString(renderTagPills(f, m.width))
			sb.WriteString("\n")
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
		v := output.Sanitize(integration.FormatValue(val))
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
		key := theme.DetailValue.Render(truncate(output.Sanitize(a.AlertKey), 30))
		summaryW := max(m.width-38, 0)
		summary := truncate(output.Sanitize(a.APIObject.Summary), summaryW)
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
		author := theme.HelpKey.Render(output.Sanitize(n.User.Summary))
		when := theme.DetailDim.Render(renderTimeAgo(n.CreatedAt))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", author, when))
		sb.WriteString(renderMarkdown(output.Sanitize(n.Content), m.width-2))
		sb.WriteString("\n")
		if i < len(m.notes)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m incidentDetail) fetchNotesCmd() tea.Cmd {
	incID := m.incident.ID
	client := m.client
	detailCtx := m.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(detailCtx, 15*time.Second)
		defer cancel()
		notes, err := client.ListIncidentNotes(ctx, incID)
		return notesLoadedMsg{incidentID: incID, notes: notes, err: err}
	}
}

func (m incidentDetail) timelineSection() string {
	if m.timelineLoading {
		return "\n" + theme.DetailDim.Render("  Loading timeline...") + "\n"
	}
	if m.timelineErr != nil {
		return "\n" + theme.DetailDim.Render(fmt.Sprintf("  Error loading timeline: %v", m.timelineErr)) + "\n"
	}
	if len(m.logEntries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(theme.DetailHeader.Render("Timeline"))
	sb.WriteString("\n\n")

	for _, e := range m.logEntries {
		entryType := strings.TrimSuffix(e.Type, "_log_entry")
		when := theme.DetailDim.Render(renderTimeAgo(e.CreatedAt))
		typeLabel := theme.HelpKey.Render(entryType)
		agentName := e.Agent.Summary
		if agentName == "" {
			agentName = "system"
		}
		agent := theme.DetailValue.Render(output.Sanitize(agentName))

		sb.WriteString(fmt.Sprintf("  %s  %s  %s", when, typeLabel, agent))

		if summary := logEntryChannelSummary(e); summary != "" {
			sb.WriteString("\n")
			sb.WriteString("    " + theme.DetailDim.Render(truncate(output.Sanitize(summary), m.width-6)))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func logEntryChannelSummary(e pagerduty.LogEntry) string {
	if s, ok := e.Channel.Raw["summary"].(string); ok && s != "" {
		return s
	}
	for _, v := range e.EventDetails {
		if v != "" {
			return v
		}
	}
	return ""
}

func (m incidentDetail) fetchAlertsCmd() tea.Cmd {
	incID := m.incident.ID
	client := m.client
	detailCtx := m.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(detailCtx, 15*time.Second)
		defer cancel()
		alerts, err := client.ListIncidentAlerts(ctx, incID)
		return alertsLoadedMsg{incidentID: incID, alerts: alerts, err: err}
	}
}

func (m incidentDetail) fetchLogEntriesCmd() tea.Cmd {
	incID := m.incident.ID
	client := m.client
	detailCtx := m.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(detailCtx, 15*time.Second)
		defer cancel()
		entries, err := client.ListIncidentLogEntries(ctx, incID, api.LogEntryOpts{})
		return logEntriesLoadedMsg{incidentID: incID, entries: entries, err: err}
	}
}
