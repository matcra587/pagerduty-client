package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/components"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// servicesLoadedMsg carries fetched services from the API.
type servicesLoadedMsg struct {
	services   []pagerduty.Service
	err        error
	generation uint64
}

// services is the model for the Services tab.
type services struct {
	services     []pagerduty.Service
	cursor       int
	expanded     map[string]bool
	statusFilter string // "all", "active", "warning", "critical", "maintenance", "disabled"
	width        int
	height       int
	scrollOffset int
	loading      bool
	loaded       bool
	err          error
}

func newServices() services {
	return services{
		expanded:     make(map[string]bool),
		statusFilter: "all",
	}
}

// visibleServices returns services matching the current status filter.
func (s services) visibleServices() []pagerduty.Service {
	if s.statusFilter == "all" {
		return s.services
	}
	var out []pagerduty.Service
	for _, svc := range s.services {
		if svc.Status == s.statusFilter {
			out = append(out, svc)
		}
	}
	return out
}

// Init implements tea.Model.
func (s services) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (s services) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case servicesLoadedMsg:
		s.loading = false
		s.loaded = true
		s.scrollOffset = 0
		if msg.err != nil {
			s.err = msg.err
			return s, nil
		}
		s.err = nil
		s.services = msg.services
		if s.cursor >= len(s.services) {
			s.cursor = max(len(s.services)-1, 0)
		}
		return s, nil

	case tea.KeyPressMsg:
		return s.updateKey(msg)
	}

	return s, nil
}

func (s services) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(s.services) == 0 {
		return s, nil
	}

	vis := s.visibleServices()

	switch msg.String() {
	case "j", "down":
		if s.cursor < len(vis)-1 {
			s.cursor++
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "enter":
		if len(vis) > 0 {
			id := vis[s.cursor].ID
			s.expanded[id] = !s.expanded[id]
		}
	case "g":
		s.cursor = 0
	case "G":
		s.cursor = len(vis) - 1
	}

	s.clampScroll()
	return s, nil
}

func (s services) viewportRows() int {
	return max(s.height-2, 1) // header text + BorderBottom
}

func (s services) linesForItem(vis []pagerduty.Service, idx int) int {
	lines := 1
	if s.expanded[vis[idx].ID] {
		details := s.renderDetails(vis[idx])
		lines += strings.Count(details, "\n")
	}
	return lines
}

func (s *services) clampScroll() {
	vis := s.visibleServices()
	if s.cursor >= len(vis) {
		s.cursor = max(len(vis)-1, 0)
	}
	maxLines := s.viewportRows()

	if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
	}

	for {
		lines := 0
		for i := s.scrollOffset; i <= s.cursor && i < len(vis); i++ {
			lines += s.linesForItem(vis, i)
		}
		if lines <= maxLines {
			break
		}
		s.scrollOffset++
		if s.scrollOffset >= len(vis) {
			break
		}
	}
}

// View implements tea.Model.
func (s services) View() tea.View {
	if s.width == 0 {
		return tea.NewView("")
	}

	if s.loading {
		return tea.NewView(lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("Loading services...")))
	}

	errStyle := lipgloss.NewStyle().Foreground(theme.Theme.Red.GetForeground()).Bold(true)

	if s.err != nil {
		msg := fmt.Sprintf("Error: %s\n\nPress R to retry", s.err)
		return tea.NewView(lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
			errStyle.Render(msg)))
	}

	if len(s.services) == 0 {
		return tea.NewView(lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("No services found")))
	}

	return tea.NewView(s.renderList())
}

func (s services) renderList() string {
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Theme.Green.GetForeground())
	vis := s.visibleServices()

	var sb strings.Builder

	// Header row with active filter.
	header := fmt.Sprintf("  %-12s %-40s %-12s [f: %s]", "ID", "Name", "Status", s.statusFilter)
	sb.WriteString(theme.TableHeader.Render(fmt.Sprintf("%-*s", s.width, header)))
	sb.WriteString("\n")

	if len(vis) == 0 {
		sb.WriteString(theme.DetailDim.Render(fmt.Sprintf("  No services with status %q", s.statusFilter)))
		sb.WriteString("\n")
		return lipgloss.NewStyle().Width(s.width).Height(s.height).MaxHeight(s.height).
			Render(sb.String())
	}

	maxLines := s.viewportRows()
	linesRendered := 0

	for i := s.scrollOffset; i < len(vis) && linesRendered < maxLines; i++ {
		svc := vis[i]
		prefix := "  "
		if i == s.cursor {
			prefix = cursorStyle.Render("> ")
		}

		status := s.renderStatus(svc.Status)

		row := fmt.Sprintf("%s%-12s %-40s %s",
			prefix,
			svc.ID,
			truncate(svc.Name, 40),
			status,
		)

		if i == s.cursor {
			sb.WriteString(components.PersistBgFull(row, theme.CursorBg, s.width))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
		linesRendered++

		// Expanded details.
		if s.expanded[svc.ID] {
			details := s.renderDetails(svc)
			sb.WriteString(details)
			linesRendered += strings.Count(details, "\n")
		}
	}

	return lipgloss.NewStyle().Width(s.width).Height(s.height).MaxHeight(s.height).
		Render(sb.String())
}

func (s services) renderStatus(status string) string {
	switch status {
	case "active":
		return lipgloss.NewStyle().Foreground(theme.Theme.Green.GetForeground()).Render(status)
	case "disabled":
		return lipgloss.NewStyle().Foreground(theme.Theme.Red.GetForeground()).Render(status)
	default:
		return theme.DetailDim.Render(status)
	}
}

func (s services) renderDetails(svc pagerduty.Service) string {
	var sb strings.Builder
	accent := lipgloss.NewStyle().Foreground(theme.Theme.Orange.GetForeground())

	type detail struct {
		label string
		value string
	}

	var details []detail

	// Escalation policy.
	epName := svc.EscalationPolicy.Summary
	if epName == "" {
		epName = svc.EscalationPolicy.ID
	}
	if epName != "" {
		details = append(details, detail{label: "Escalation Policy", value: epName})
	}

	// Teams.
	if len(svc.Teams) > 0 {
		names := make([]string, len(svc.Teams))
		for i, t := range svc.Teams {
			names[i] = t.Name
		}
		details = append(details, detail{label: "Teams", value: strings.Join(names, ", ")})
	}

	// Description.
	if svc.Description != "" {
		details = append(details, detail{label: "Description", value: svc.Description})
	}

	for i, d := range details {
		connector := "├─"
		if i == len(details)-1 {
			connector = "└─"
		}

		label := fmt.Sprintf("    %s %s  ",
			accent.Render(connector),
			theme.DetailLabel.Render(d.label+":"),
		)
		labelW := lipgloss.Width(label)
		valueW := max(s.width-labelW-1, 20)

		// Wrap value text then indent continuation lines.
		wrapped := lipgloss.NewStyle().Width(valueW).Render(d.value)
		indent := strings.Repeat(" ", labelW)
		lines := strings.Split(wrapped, "\n")

		for j, l := range lines {
			if j == 0 {
				sb.WriteString(label + theme.DetailDim.Render(l))
			} else {
				sb.WriteString(indent + theme.DetailDim.Render(l))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
