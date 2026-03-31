// Package components contains reusable TUI sub-models.
package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// Column defines a table column with a header label and a width weight.
type Column struct {
	Header string
	Width  int
}

// RowSelected is sent when the user presses enter on a table row.
// Value contains the raw string slice for the selected row.
type RowSelected struct {
	Index int
	Row   []string
}

// Table is a generic, reusable Bubble Tea component that renders a scrollable
// table with configurable columns and keyboard navigation.
//
// The caller owns the row data ([][]string). Table does not fetch or transform
// data; it only handles rendering and navigation.
type Table struct {
	Columns  []Column
	Rows     [][]string
	Selected int
	Width    int
	Height   int
}

// Init implements tea.Model.
func (t Table) Init() tea.Cmd { return nil }

// Update implements tea.Model. Handles j/k/arrow navigation and enter.
func (t Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "j", "down":
			if t.Selected < len(t.Rows)-1 {
				t.Selected++
			}
		case "k", "up":
			if t.Selected > 0 {
				t.Selected--
			}
		case "enter":
			if len(t.Rows) > 0 {
				row := t.Rows[t.Selected]
				idx := t.Selected
				return t, func() tea.Msg {
					return RowSelected{Index: idx, Row: row}
				}
			}
		}
	}
	return t, nil
}

// View implements tea.Model. Renders header and visible rows.
func (t Table) View() tea.View {
	if len(t.Columns) == 0 {
		return tea.NewView("")
	}

	var sb strings.Builder

	sb.WriteString(theme.TableHeader.Render(t.headerRow()))
	sb.WriteString("\n")

	if len(t.Rows) == 0 {
		sb.WriteString("\n  (no data)\n")
		return tea.NewView(sb.String())
	}

	// Visible window: reserve 2 lines for header + newline.
	maxRows := max(t.Height-2, 1)

	start := 0
	if t.Selected >= maxRows {
		start = t.Selected - maxRows + 1
	}

	for i := start; i < len(t.Rows) && (i-start) < maxRows; i++ {
		sb.WriteString(t.renderRow(i))
		sb.WriteString("\n")
	}

	return tea.NewView(sb.String())
}

func (t Table) headerRow() string {
	parts := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		parts[i] = lipgloss.NewStyle().Width(col.Width).Render(col.Header)
	}
	return strings.Join(parts, " ")
}

func (t Table) renderRow(idx int) string {
	row := t.Rows[idx]
	cells := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		val := ""
		if i < len(row) {
			val = truncateStr(row[i], col.Width)
		}
		cells[i] = lipgloss.NewStyle().Width(col.Width).Render(val)
	}
	line := strings.Join(cells, " ")

	if idx == t.Selected {
		return theme.SelectedRow.Render(line)
	}
	return line
}

// truncateStr shortens s to at most n runes, appending an ellipsis if truncated.
func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}
