package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// teamTabLoadedMsg carries fetched teams from the API.
// Named to avoid conflict with components.TeamsLoadedMsg used by the team switcher.
type teamTabLoadedMsg struct {
	teams []pagerduty.Team
	err   error
}

// teamsView is the model for the Teams tab.
type teamsView struct {
	teams        []pagerduty.Team
	cursor       int
	expanded     map[string]bool
	width        int
	height       int
	scrollOffset int
	loading      bool
	loaded       bool
	err          error
}

func newTeamsView() teamsView {
	return teamsView{
		expanded: make(map[string]bool),
	}
}

// Init implements tea.Model.
func (tv teamsView) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (tv teamsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tv.width = msg.Width
		tv.height = msg.Height
		return tv, nil

	case teamTabLoadedMsg:
		tv.loading = false
		tv.loaded = true
		tv.scrollOffset = 0
		if msg.err != nil {
			tv.err = msg.err
			return tv, nil
		}
		tv.err = nil
		tv.teams = msg.teams
		if tv.cursor >= len(tv.teams) {
			tv.cursor = max(len(tv.teams)-1, 0)
		}
		return tv, nil

	case tea.KeyPressMsg:
		return tv.updateKey(msg)
	}

	return tv, nil
}

func (tv teamsView) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(tv.teams) == 0 {
		return tv, nil
	}

	switch msg.String() {
	case "j", "down":
		if tv.cursor < len(tv.teams)-1 {
			tv.cursor++
		}
	case "k", "up":
		if tv.cursor > 0 {
			tv.cursor--
		}
	case "enter":
		id := tv.teams[tv.cursor].ID
		tv.expanded[id] = !tv.expanded[id]
	case "g":
		tv.cursor = 0
	case "G":
		tv.cursor = len(tv.teams) - 1
	}

	tv.clampScroll()
	return tv, nil
}

func (tv teamsView) viewportRows() int {
	return max(tv.height-2, 1) // header text + BorderBottom
}

func (tv teamsView) linesForItem(idx int) int {
	lines := 1
	if tv.expanded[tv.teams[idx].ID] {
		count := 0
		if tv.teams[idx].Description != "" {
			count++
		}
		if tv.teams[idx].Parent != nil {
			count++
		}
		if count == 0 {
			count = 1 // "(none)" placeholder
		}
		lines += count
	}
	return lines
}

func (tv *teamsView) clampScroll() {
	maxLines := tv.viewportRows()

	if tv.cursor < tv.scrollOffset {
		tv.scrollOffset = tv.cursor
	}

	for {
		lines := 0
		for i := tv.scrollOffset; i <= tv.cursor && i < len(tv.teams); i++ {
			lines += tv.linesForItem(i)
		}
		if lines <= maxLines {
			break
		}
		tv.scrollOffset++
		if tv.scrollOffset >= len(tv.teams) {
			break
		}
	}
}

// View implements tea.Model.
func (tv teamsView) View() tea.View {
	if tv.width == 0 {
		return tea.NewView("")
	}

	if tv.loading {
		return tea.NewView(lipgloss.Place(tv.width, tv.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("Loading teams...")))
	}

	errStyle := lipgloss.NewStyle().Foreground(theme.Theme.Red.GetForeground()).Bold(true)

	if tv.err != nil {
		msg := fmt.Sprintf("Error: %s\n\nPress R to retry", tv.err)
		return tea.NewView(lipgloss.Place(tv.width, tv.height, lipgloss.Center, lipgloss.Center,
			errStyle.Render(msg)))
	}

	if len(tv.teams) == 0 {
		return tea.NewView(lipgloss.Place(tv.width, tv.height, lipgloss.Center, lipgloss.Center,
			theme.DetailDim.Render("No teams found")))
	}

	return tea.NewView(tv.renderList())
}

func (tv teamsView) renderList() string {
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Theme.Green.GetForeground())

	var sb strings.Builder

	// Header row.
	header := fmt.Sprintf("  %-12s %s", "ID", "Name")
	sb.WriteString(theme.TableHeader.Render(fmt.Sprintf("%-*s", tv.width, header)))
	sb.WriteString("\n")

	maxLines := tv.viewportRows()
	linesRendered := 0

	for i := tv.scrollOffset; i < len(tv.teams) && linesRendered < maxLines; i++ {
		t := tv.teams[i]
		prefix := "  "
		if i == tv.cursor {
			prefix = cursorStyle.Render("> ")
		}

		row := fmt.Sprintf("%s%-12s %s",
			prefix,
			t.ID,
			truncate(t.Name, 40),
		)

		if i == tv.cursor {
			sb.WriteString(theme.SelectedRow.Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
		linesRendered++

		// Expanded detail.
		if tv.expanded[t.ID] {
			detail := tv.renderDetail(t)
			sb.WriteString(detail)
			linesRendered += strings.Count(detail, "\n")
		}
	}

	return lipgloss.NewStyle().Width(tv.width).Height(tv.height).MaxHeight(tv.height).
		Render(sb.String())
}

func (tv teamsView) renderDetail(t pagerduty.Team) string {
	var sb strings.Builder
	accent := lipgloss.NewStyle().Foreground(theme.Theme.Orange.GetForeground())

	type field struct {
		label string
		value string
	}

	var fields []field

	if t.Description != "" {
		fields = append(fields, field{label: "Description", value: t.Description})
	}

	if t.Parent != nil {
		name := t.Parent.Summary
		if name == "" {
			name = t.Parent.ID
		}
		fields = append(fields, field{label: "Parent", value: name})
	}

	if len(fields) == 0 {
		fields = append(fields, field{label: "Description", value: "(none)"})
	}

	for i, f := range fields {
		connector := "├─"
		if i == len(fields)-1 {
			connector = "└─"
		}

		label := fmt.Sprintf("    %s %s: ",
			accent.Render(connector),
			theme.DetailLabel.Render(f.label),
		)
		labelW := lipgloss.Width(label)
		valueW := max(tv.width-labelW-1, 20)

		wrapped := lipgloss.NewStyle().Width(valueW).Render(f.value)
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
