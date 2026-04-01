package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// epLoadedMsg carries fetched escalation policies from the API.
type epLoadedMsg struct {
	policies []pagerduty.EscalationPolicy
	err      error
}

// escalationPolicies is the model for the Escalation Policies tab.
type escalationPolicies struct {
	policies []pagerduty.EscalationPolicy
	cursor   int
	expanded map[string]bool
	width    int
	height   int
	loading  bool
	loaded   bool
	err      error
}

func newEscalationPolicies() escalationPolicies {
	return escalationPolicies{
		expanded: make(map[string]bool),
	}
}

// Init implements tea.Model.
func (ep escalationPolicies) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (ep escalationPolicies) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ep.width = msg.Width
		ep.height = msg.Height
		return ep, nil

	case epLoadedMsg:
		ep.loading = false
		ep.loaded = true
		if msg.err != nil {
			ep.err = msg.err
			return ep, nil
		}
		ep.err = nil
		ep.policies = msg.policies
		if ep.cursor >= len(ep.policies) {
			ep.cursor = max(len(ep.policies)-1, 0)
		}
		return ep, nil

	case tea.KeyPressMsg:
		return ep.updateKey(msg)
	}

	return ep, nil
}

func (ep escalationPolicies) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(ep.policies) == 0 {
		return ep, nil
	}

	switch msg.String() {
	case "j", "down":
		if ep.cursor < len(ep.policies)-1 {
			ep.cursor++
		}
	case "k", "up":
		if ep.cursor > 0 {
			ep.cursor--
		}
	case "enter":
		id := ep.policies[ep.cursor].ID
		ep.expanded[id] = !ep.expanded[id]
	case "g":
		ep.cursor = 0
	case "G":
		ep.cursor = len(ep.policies) - 1
	}

	return ep, nil
}

// View implements tea.Model.
func (ep escalationPolicies) View() tea.View {
	if ep.width == 0 {
		return tea.NewView("")
	}

	if ep.loading {
		return tea.NewView(lipgloss.Place(ep.width, ep.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("Loading escalation policies...")))
	}

	errStyle := lipgloss.NewStyle().Foreground(theme.Theme.Red.GetForeground()).Bold(true)

	if ep.err != nil {
		msg := fmt.Sprintf("Error: %s\n\nPress R to retry", ep.err)
		return tea.NewView(lipgloss.Place(ep.width, ep.height, lipgloss.Center, lipgloss.Center,
			errStyle.Render(msg)))
	}

	if len(ep.policies) == 0 {
		return tea.NewView(lipgloss.Place(ep.width, ep.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("No escalation policies found")))
	}

	return tea.NewView(ep.renderList())
}

func (ep escalationPolicies) renderList() string {
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Theme.Green.GetForeground())

	var sb strings.Builder

	// Header row with full-width separator.
	header := fmt.Sprintf("  %-12s %-30s %s", "ID", "Name", "Loops")
	sb.WriteString(theme.TableHeader.Render(fmt.Sprintf("%-*s", ep.width, header)))
	sb.WriteString("\n")

	for i, p := range ep.policies {
		prefix := "  "
		if i == ep.cursor {
			prefix = cursorStyle.Render("> ")
		}

		row := fmt.Sprintf("%s%-12s %-30s %s",
			prefix,
			p.ID,
			truncate(p.Name, 30),
			strconv.FormatUint(uint64(p.NumLoops), 10),
		)

		if i == ep.cursor {
			sb.WriteString(theme.SelectedRow.Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")

		// Expanded rules.
		if ep.expanded[p.ID] {
			sb.WriteString(ep.renderRules(p.EscalationRules))
		}
	}

	return lipgloss.NewStyle().Width(ep.width).Height(ep.height).MaxHeight(ep.height).
		Render(sb.String())
}

func (ep escalationPolicies) renderRules(rules []pagerduty.EscalationRule) string {
	var sb strings.Builder
	accent := lipgloss.NewStyle().Foreground(theme.Theme.Orange.GetForeground())

	for i, r := range rules {
		connector := "├─"
		if i == len(rules)-1 {
			connector = "└─"
		}

		targets := formatTargets(r.Targets)
		line := fmt.Sprintf("    %s L%d  %d min  %s",
			accent.Render(connector),
			i+1,
			r.Delay,
			theme.DetailDim.Render(targets),
		)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatTargets(targets []pagerduty.APIObject) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		name := t.Summary
		if name == "" {
			name = t.ID
		}
		typeName := strings.TrimSuffix(t.Type, "_reference")
		parts[i] = fmt.Sprintf("%s (%s)", name, typeName)
	}
	return strings.Join(parts, ", ")
}
