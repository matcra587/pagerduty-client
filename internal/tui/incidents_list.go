package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/tui/components"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// incidentList is a Bubble Tea model that renders a table of incidents.
type incidentList struct {
	ctx          context.Context //nolint:containedctx // Bubble Tea models are value-typed; context must travel with the model.
	incidents    []pagerduty.Incident
	cursor       int
	scrollOffset int
	selections   map[string]bool
	width        int
	height       int
	client       *api.Client
	fromEmail    string
	hideService  bool
	filterInput  textinput.Model
	filterActive bool
	filterState  components.FilterState
}

func (m incidentList) readOnly() bool {
	return m.client == nil
}

func newIncidentList(ctx context.Context, client *api.Client, fromEmail string, hideService bool) incidentList {
	fi := textinput.New()
	fi.Prompt = ""
	fi.CharLimit = 80
	return incidentList{
		ctx:         ctx,
		client:      client,
		fromEmail:   fromEmail,
		hideService: hideService,
		selections:  make(map[string]bool),
		filterInput: fi,
	}
}

// Init implements tea.Model.
func (m incidentList) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m incidentList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if m.filterActive {
		switch key.String() {
		case "esc":
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			m.filterActive = false
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil
		case "enter":
			m.filterInput.Blur()
			m.filterActive = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(key)
			vis := m.visibleIncidents()
			if m.cursor >= len(vis) {
				m.cursor = max(0, len(vis)-1)
			}
			if m.scrollOffset > m.cursor {
				m.scrollOffset = m.cursor
			}
			return m, cmd
		}
	}

	vis := m.visibleIncidents()

	switch key.String() {
	case "/":
		m.filterActive = true
		cmd := m.filterInput.Focus()
		return m, tea.Batch(cmd, textinput.Blink)
	case "j", "down":
		if m.cursor < len(vis)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "shift+down":
		if len(vis) > 0 {
			m.selections[vis[m.cursor].ID] = true
			if m.cursor < len(vis)-1 {
				m.cursor++
			}
		}
	case "shift+up":
		if len(vis) > 0 {
			m.selections[vis[m.cursor].ID] = true
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case "space":
		if len(vis) > 0 {
			m.selections[vis[m.cursor].ID] = !m.selections[vis[m.cursor].ID]
			if !m.selections[vis[m.cursor].ID] {
				delete(m.selections, vis[m.cursor].ID)
			}
		}
	case "ctrl+a":
		for _, inc := range vis {
			m.selections[inc.ID] = true
		}
	case "esc":
		if len(m.selections) > 0 {
			m.selections = make(map[string]bool)
			return m, nil
		}
	case "enter":
		if len(vis) > 0 {
			inc := vis[m.cursor]
			return m, func() tea.Msg {
				return IncidentSelected{Incident: inc}
			}
		}
	case "a":
		if len(vis) == 0 {
			return m, nil
		}
		inc := vis[m.cursor]
		pending := func() tea.Msg {
			return incidentActionPendingMsg{op: "ack", id: inc.ID}
		}
		return m, tea.Batch(pending, m.ackCmd())
	case "s":
		if len(vis) > 0 {
			id := vis[m.cursor].ID
			return m, func() tea.Msg {
				return showInputMsg{
					action:      "snooze",
					incidentID:  id,
					prompt:      "Snooze duration (e.g. 4h, 30m):",
					placeholder: "4h",
				}
			}
		}
	case "n":
		if len(vis) > 0 {
			id := vis[m.cursor].ID
			return m, func() tea.Msg {
				return showInputMsg{
					action:     "note",
					incidentID: id,
					prompt:     "Add note:",
				}
			}
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(vis) > 0 {
			m.cursor = len(vis) - 1
		}
	}

	maxRows := m.viewportRows()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxRows {
		m.scrollOffset = m.cursor - maxRows + 1
	}

	return m, nil
}

// viewportRows returns how many incident data rows fit in the available height.
func (m incidentList) viewportRows() int {
	rows := m.height - 1 // column header
	if m.filterActive || m.filterInput.Value() != "" {
		rows-- // filter bar
	}
	return max(rows, 1)
}

// View implements tea.Model.
func (m incidentList) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	var sb strings.Builder

	showFilter := m.filterActive || m.filterInput.Value() != ""
	if showFilter {
		if m.filterActive {
			sb.WriteString("  / " + m.filterInput.View())
		} else {
			sb.WriteString("  filter: " + m.filterInput.Value())
		}
		sb.WriteString("\n")
	}

	vis := m.visibleIncidents()

	if len(vis) == 0 {
		msg := "🎉 No active incidents"
		if showFilter {
			msg = "🔍 No matching incidents"
		}
		sb.WriteString(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Faint(true).Render(msg)))
		return tea.NewView(sb.String())
	}

	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")

	maxRows := m.viewportRows()

	start := min(m.scrollOffset, max(0, len(vis)-maxRows))

	for i := start; i < len(vis) && (i-start) < maxRows; i++ {
		sb.WriteString(m.renderRowFromIncident(vis[i], i == m.cursor))
		sb.WriteString("\n")
	}

	return tea.NewView(sb.String())
}

// SetIncidents replaces the incident list and clamps the cursor.
func (m *incidentList) SetIncidents(incidents []pagerduty.Incident) {
	m.incidents = incidents
	vis := m.visibleIncidents()
	if m.cursor >= len(vis) {
		m.cursor = max(0, len(vis)-1)
	}
	if m.scrollOffset > m.cursor {
		m.scrollOffset = m.cursor
	}
}

// FilterActive reports whether the filter input is focused.
func (m incidentList) FilterActive() bool {
	return m.filterActive
}

// ClearFilter resets the filter value and deactivates the input.
func (m *incidentList) ClearFilter() {
	m.filterInput.SetValue("")
	m.filterInput.Blur()
	m.filterActive = false
	vis := m.visibleIncidents()
	if m.cursor >= len(vis) {
		m.cursor = max(0, len(vis)-1)
	}
	if m.scrollOffset > m.cursor {
		m.scrollOffset = m.cursor
	}
}

// visibleIncidents returns the subset of incidents matching the current text
// filter and any structured filter criteria.
func (m incidentList) visibleIncidents() []pagerduty.Incident {
	query := strings.ToLower(m.filterInput.Value())
	hasStructured := !isDefaultFilter(m.filterState)

	if query == "" && !hasStructured {
		return m.incidents
	}

	var result []pagerduty.Incident
	for _, inc := range m.incidents {
		if query != "" && !matchesFilter(inc, query) {
			continue
		}
		if hasStructured && !matchesStructuredFilter(inc, m.filterState) {
			continue
		}
		result = append(result, inc)
	}
	return result
}

func isDefaultFilter(fs components.FilterState) bool {
	return (fs.Priority == "" || fs.Priority == "all") &&
		(fs.Assigned == "" || fs.Assigned == "all")
}

func matchesStructuredFilter(inc pagerduty.Incident, fs components.FilterState) bool {
	if fs.Priority != "" && fs.Priority != "all" {
		name := ""
		if inc.Priority != nil {
			name = inc.Priority.Name
		}
		if name != fs.Priority {
			return false
		}
	}

	if fs.Assigned == "assigned" && len(inc.Assignments) == 0 {
		return false
	}
	if fs.Assigned == "unassigned" && len(inc.Assignments) > 0 {
		return false
	}

	return true
}

func matchesFilter(inc pagerduty.Incident, query string) bool {
	if strings.Contains(strings.ToLower(inc.ID), query) {
		return true
	}
	if strings.Contains(strings.ToLower(inc.Title), query) {
		return true
	}
	if strings.Contains(strings.ToLower(inc.Service.Summary), query) {
		return true
	}
	for _, a := range inc.Assignments {
		if strings.Contains(strings.ToLower(a.Assignee.Summary), query) {
			return true
		}
	}
	return false
}

func (m incidentList) hiddenColumns() []int {
	if m.hideService {
		return []int{colService}
	}
	return nil
}

func (m incidentList) renderHeader() string {
	widths := layoutColumns(m.width, incidentColumns, m.hiddenColumns()...)
	var parts []string
	for i, col := range incidentColumns {
		w := widths[i]
		if w == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%-*s", w, col.header))
	}
	return theme.TableHeader.Render(strings.Join(parts, " "))
}

func (m incidentList) renderRowFromIncident(inc pagerduty.Incident, isCursor bool) string {
	widths := layoutColumns(m.width, incidentColumns, m.hiddenColumns()...)

	isSelected := m.selections[inc.ID]

	style := incidentStyle(inc)
	assigneeStyle := theme.EntityColor(assigneeNames(inc.Assignments))
	dimStyle := theme.DetailDim

	prefix := "  "
	if isCursor {
		prefix = "> "
	} else if isSelected {
		prefix = "* "
	}

	type cell struct {
		text  string
		style lipgloss.Style
	}
	cells := []cell{
		{prefix, lipgloss.NewStyle()},
		{severityLabel(inc), style},
		{inc.Title, style},
		{inc.Service.Summary, theme.EntityColor(inc.Service.Summary)},
		{assigneeNames(inc.Assignments), assigneeStyle},
		{renderTimePlain(inc.CreatedAt), dimStyle},
		{renderTimePlain(inc.LastStatusChangeAt), dimStyle},
	}

	var parts []string
	for i, w := range widths {
		if w == 0 {
			continue
		}
		truncated := truncate(cells[i].text, w)
		padded := fmt.Sprintf("%-*s", w, truncated)
		parts = append(parts, cells[i].style.Render(padded))
	}

	row := strings.Join(parts, " ")

	if isCursor {
		return components.PersistBg(row, components.ColorToANSIBg(theme.ColorHighlightBg))
	}
	if isSelected {
		return components.PersistBg(row, components.ColorToANSIBg(theme.ColorSelectedBg))
	}

	return row
}

func (m incidentList) ackCmd() tea.Cmd {
	if m.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	vis := m.visibleIncidents()
	if len(vis) == 0 {
		return nil
	}
	inc := vis[m.cursor]
	client := m.client
	listCtx := m.ctx
	from := m.fromEmail
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(listCtx, 15*time.Second)
		defer cancel()
		if err := client.AckIncident(ctx, inc.ID, from); err != nil {
			return incidentErrMsg{op: "ack", err: err}
		}
		return IncidentAcked{ID: inc.ID}
	}
}

func (m incidentList) resolveCmd() tea.Cmd {
	if m.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	vis := m.visibleIncidents()
	if len(vis) == 0 {
		return nil
	}
	inc := vis[m.cursor]
	client := m.client
	listCtx := m.ctx
	from := m.fromEmail
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(listCtx, 15*time.Second)
		defer cancel()
		if err := client.ResolveIncident(ctx, inc.ID, from); err != nil {
			return incidentErrMsg{op: "resolve", err: err}
		}
		return IncidentResolved{ID: inc.ID}
	}
}

func (m incidentList) selectedIncidents() []pagerduty.Incident {
	var result []pagerduty.Incident
	for _, inc := range m.visibleIncidents() {
		if m.selections[inc.ID] {
			result = append(result, inc)
		}
	}
	return result
}

// batchCmd returns a tea.Cmd that runs fn on each selected incident
// concurrently with a bounded semaphore. Uses sync.WaitGroup.Go (Go 1.25+).
func (m incidentList) batchCmd(op string, fn func(ctx context.Context, id, from string) error) tea.Cmd {
	if m.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	incidents := m.selectedIncidents()
	listCtx := m.ctx
	from := m.fromEmail
	return func() tea.Msg {
		sem := make(chan struct{}, 5)
		var mu sync.Mutex
		var success, failures int
		var firstErr error
		var wg sync.WaitGroup
		for _, inc := range incidents {
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()
				ctx, cancel := context.WithTimeout(listCtx, 15*time.Second)
				defer cancel()
				if err := fn(ctx, inc.ID, from); err != nil {
					mu.Lock()
					failures++
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", inc.ID, err)
					}
					mu.Unlock()
					return
				}
				mu.Lock()
				success++
				mu.Unlock()
			})
		}
		wg.Wait()
		return batchResultMsg{op: op, success: success, failures: failures, firstErr: firstErr}
	}
}

func (m incidentList) batchAckCmd() tea.Cmd {
	return m.batchCmd("ack", func(ctx context.Context, id, from string) error {
		return m.client.AckIncident(ctx, id, from)
	})
}

func (m incidentList) batchResolveCmd() tea.Cmd {
	return m.batchCmd("resolve", func(ctx context.Context, id, from string) error {
		return m.client.ResolveIncident(ctx, id, from)
	})
}
