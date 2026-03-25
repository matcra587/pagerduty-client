package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// FilterState holds the currently applied filter values.
type FilterState struct {
	Status   string // "open", "triggered", "acked", "resolved", "all"
	Urgency  string // "high", "low", "all"
	Priority string // "P1", "P2", "P3", "P4", "all"
	Assigned string // "assigned", "unassigned", "all"
	Age      string // "7d", "30d", "60d", "90d", "all"
}

// DefaultFilterState returns the default filter values.
func DefaultFilterState() FilterState {
	return FilterState{Status: "open", Urgency: "all", Priority: "all", Assigned: "all", Age: "7d"}
}

// ActiveCount returns how many filters differ from their defaults.
func (fs FilterState) ActiveCount() int {
	defaults := DefaultFilterState()
	var count int
	if fs.Status != defaults.Status {
		count++
	}
	if fs.Urgency != defaults.Urgency {
		count++
	}
	if fs.Priority != defaults.Priority {
		count++
	}
	if fs.Assigned != defaults.Assigned {
		count++
	}
	if fs.Age != defaults.Age {
		count++
	}
	return count
}

// ChipSummary returns a space-separated string of "key:value" pairs for
// each filter that differs from its default.
func (fs FilterState) ChipSummary() string {
	defaults := DefaultFilterState()
	var chips []string
	if fs.Status != defaults.Status {
		chips = append(chips, "status:"+fs.Status)
	}
	if fs.Urgency != defaults.Urgency {
		chips = append(chips, "urgency:"+fs.Urgency)
	}
	if fs.Priority != defaults.Priority {
		chips = append(chips, "priority:"+fs.Priority)
	}
	if fs.Assigned != defaults.Assigned {
		chips = append(chips, "assigned:"+fs.Assigned)
	}
	if fs.Age != defaults.Age {
		chips = append(chips, "age:"+fs.Age)
	}
	return strings.Join(chips, " ")
}

// FilterApplied is sent when the user confirms the filter selection.
type FilterApplied struct{ State FilterState }

// FilterClosed is sent when the user dismisses the filter overlay without applying.
type FilterClosed struct{}

// filterRow describes a single filter option row with a label and ordered choices.
type filterRow struct {
	label   string
	choices []string
	current int // index into choices
}

// FilterOptions is a Bubble Tea overlay component that lets the user configure
// incident filter criteria. State is preserved between openings.
type FilterOptions struct {
	Visible bool
	state   FilterState
	cursor  int
	rows    []filterRow
}

// NewFilterOptions returns a FilterOptions with all choices set to "all".
func NewFilterOptions() FilterOptions {
	return FilterOptions{
		rows: []filterRow{
			{label: "Status", choices: []string{"open", "triggered", "acked", "resolved", "all"}, current: 0},
			{label: "Urgency", choices: []string{"high", "low", "all"}, current: 2},
			{label: "Priority", choices: []string{"P1", "P2", "P3", "P4", "P5", "all"}, current: 5},
			{label: "Assigned", choices: []string{"assigned", "unassigned", "all"}, current: 2},
			{label: "Age", choices: []string{"7d", "30d", "60d", "90d", "all"}, current: 0},
		},
	}
}

func (f FilterOptions) choiceByLabel(label string) string {
	for _, row := range f.rows {
		if row.label == label {
			return row.choices[row.current]
		}
	}
	return ""
}

// State returns the current FilterState matching the UI defaults.
func (f FilterOptions) State() FilterState {
	return FilterState{
		Status:   f.choiceByLabel("Status"),
		Urgency:  f.choiceByLabel("Urgency"),
		Priority: f.choiceByLabel("Priority"),
		Assigned: f.choiceByLabel("Assigned"),
		Age:      f.choiceByLabel("Age"),
	}
}

// Show sets Visible to true and returns the FilterOptions. Previously selected
// filter values are preserved.
func (f FilterOptions) Show() FilterOptions {
	f.Visible = true
	return f
}

// Init implements tea.Model.
func (f FilterOptions) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (f FilterOptions) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !f.Visible {
		return f, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, nil
	}

	switch key.String() {
	case "j", "down":
		if f.cursor < len(f.rows)-1 {
			f.cursor++
		}

	case "k", "up":
		if f.cursor > 0 {
			f.cursor--
		}

	case "space", "l", "right":
		row := &f.rows[f.cursor]
		row.current = (row.current + 1) % len(row.choices)

	case "h", "left":
		row := &f.rows[f.cursor]
		row.current = (row.current - 1 + len(row.choices)) % len(row.choices)

	case "backspace":
		row := &f.rows[f.cursor]
		row.current = len(row.choices) - 1

	case "enter":
		f.state = f.State()
		f.Visible = false
		state := f.state
		return f, func() tea.Msg { return FilterApplied{State: state} }

	case "esc":
		f.Visible = false
		return f, func() tea.Msg { return FilterClosed{} }
	}

	return f, nil
}

// View implements tea.Model. Returns an empty string when not visible.
func (f FilterOptions) View() tea.View {
	if !f.Visible {
		return tea.NewView("")
	}

	var sb strings.Builder
	sb.WriteString(theme.Title.Render("Filter Options"))
	sb.WriteString("\n\n")

	maxLabel := 0
	for _, row := range f.rows {
		if w := len(row.label) + 1; w > maxLabel {
			maxLabel = w
		}
	}

	dim := lipgloss.NewStyle().Foreground(theme.ColorOverlayBorder)

	for i, row := range f.rows {
		cursor := "  "
		if i == f.cursor {
			cursor = "> "
		}

		label := theme.HelpDesc.Render(fmt.Sprintf("%-*s", maxLabel, row.label+":"))

		var choices []string
		for j, c := range row.choices {
			if j == row.current {
				choices = append(choices, theme.HelpKey.Render(c))
			} else {
				choices = append(choices, dim.Render(c))
			}
		}

		line := cursor + label + "  " + strings.Join(choices, dim.Render(" | "))
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpDesc.Render("↑↓ navigate  ←→ select  enter apply  esc close  ⌫ reset"))

	return tea.NewView(RenderOverlay(sb.String(), 52))
}
