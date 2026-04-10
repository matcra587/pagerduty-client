package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
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

// FilterAppliedMsg is sent when the user confirms the filter selection.
// Origin identifies which tab opened the overlay; Selections maps each
// row label to the chosen value.
type FilterAppliedMsg struct {
	Origin     string
	Selections map[string]string
}

// FilterClosed is sent when the user dismisses the filter overlay without applying.
type FilterClosed struct{}

// FilterRow describes a single filter option row with a label and ordered choices.
type FilterRow struct {
	Label   string
	Choices []string
	Current int // index into Choices
}

// FilterOptions is a Bubble Tea overlay component that lets the user configure
// filter criteria. It is data-driven: callers provide rows at show-time and
// the component renders whatever it receives without knowing about specific tabs.
type FilterOptions struct {
	Visible bool
	origin  string
	cursor  int
	rows    []FilterRow
}

// NewFilterOptions returns an empty FilterOptions ready to receive rows.
func NewFilterOptions() FilterOptions {
	return FilterOptions{}
}

// IncidentFilterRows returns the default filter rows for the incidents tab.
func IncidentFilterRows() []FilterRow {
	return []FilterRow{
		{Label: "Status", Choices: []string{"open", "triggered", "acked", "resolved", "all"}, Current: 0},
		{Label: "Urgency", Choices: []string{"high", "low", "all"}, Current: 2},
		{Label: "Priority", Choices: []string{"P1", "P2", "P3", "P4", "P5", "all"}, Current: 5},
		{Label: "Assigned", Choices: []string{"assigned", "unassigned", "all"}, Current: 2},
		{Label: "Age", Choices: []string{"7d", "30d", "60d", "90d", "all"}, Current: 0},
	}
}

// ServiceFilterRows returns the default filter rows for the services tab.
func ServiceFilterRows() []FilterRow {
	return []FilterRow{
		{Label: "Status", Choices: []string{"all", "active", "warning", "critical", "maintenance", "disabled"}, Current: 0},
	}
}

// Selections returns a map of label to currently selected value for each row.
func (f FilterOptions) Selections() map[string]string {
	m := make(map[string]string, len(f.rows))
	for _, row := range f.rows {
		m[row.Label] = row.Choices[row.Current]
	}
	return m
}

// Origin returns the tab identifier that opened the overlay.
func (f FilterOptions) Origin() string {
	return f.origin
}

// State returns the current FilterState by reading selections from the rows.
// This is a convenience for the incidents tab which uses FilterState.
func (f FilterOptions) State() FilterState {
	sel := f.Selections()
	return FilterState{
		Status:   sel["Status"],
		Urgency:  sel["Urgency"],
		Priority: sel["Priority"],
		Assigned: sel["Assigned"],
		Age:      sel["Age"],
	}
}

// ShowWithRows sets Visible to true, stores the origin and rows, and returns
// the updated FilterOptions. The cursor is reset to the first row.
func (f FilterOptions) ShowWithRows(origin string, rows []FilterRow) FilterOptions {
	f.Visible = true
	f.origin = origin
	f.rows = rows
	f.cursor = 0
	return f
}

// Show opens the overlay with the default incident filter rows for backwards
// compatibility. Prefer ShowWithRows for tab-specific rows.
func (f FilterOptions) Show() FilterOptions {
	return f.ShowWithRows("incidents", IncidentFilterRows())
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
		row.Current = (row.Current + 1) % len(row.Choices)

	case "h", "left":
		row := &f.rows[f.cursor]
		row.Current = (row.Current - 1 + len(row.Choices)) % len(row.Choices)

	case "backspace":
		row := &f.rows[f.cursor]
		row.Current = len(row.Choices) - 1

	case "enter":
		f.Visible = false
		origin := f.origin
		selections := f.Selections()
		return f, func() tea.Msg { return FilterAppliedMsg{Origin: origin, Selections: selections} }

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
		if w := len(row.Label) + 1; w > maxLabel {
			maxLabel = w
		}
	}

	dim := theme.HelpDesc

	for i, row := range f.rows {
		cursor := "  "
		if i == f.cursor {
			cursor = "> "
		}

		label := theme.HelpDesc.Render(fmt.Sprintf("%-*s", maxLabel, row.Label+":"))

		var choices []string
		for j, c := range row.Choices {
			if j == row.Current {
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
