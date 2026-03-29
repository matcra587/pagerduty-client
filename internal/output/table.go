package output

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
)

// Column styles for coloured table output.
var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)

	statusStyles = map[string]lipgloss.Style{
		"triggered":    lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		"acknowledged": lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		"resolved":     lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		"high":         lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		"low":          lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
	}
)

const (
	colGap   = 2  // spaces between columns
	maxCellW = 60 // max cell width before truncation
)

// RenderTable writes a column-aligned text table to w.
// When colour is true, headers are bold and status/urgency values
// are coloured. Long cell values are truncated.
func RenderTable(w io.Writer, headers []string, rows [][]string, colour bool) error {
	if len(headers) == 0 {
		return nil
	}

	// Truncate long cells first (before measuring).
	for _, row := range rows {
		for j, cell := range row {
			if xansi.StringWidth(cell) > maxCellW {
				row[j] = xansi.Truncate(cell, maxCellW-3, "...")
			}
		}
	}

	// Measure column widths from display width (rune-aware).
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = xansi.StringWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := xansi.StringWidth(cell); i < len(widths) && w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Find status/urgency columns for colouring.
	statusCol, urgencyCol := -1, -1
	for i, h := range headers {
		switch h {
		case "Status":
			statusCol = i
		case "Urgency":
			urgencyCol = i
		}
	}

	// Render header row.
	var sb strings.Builder
	for i, h := range headers {
		if i > 0 {
			sb.WriteString(strings.Repeat(" ", colGap))
		}
		padded := fmt.Sprintf("%-*s", widths[i], h)
		if colour {
			sb.WriteString(headerStyle.Render(padded))
		} else {
			sb.WriteString(padded)
		}
	}
	sb.WriteString("\n")

	// Render data rows.
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				sb.WriteString(strings.Repeat(" ", colGap))
			}
			padded := fmt.Sprintf("%-*s", widths[i], cell)

			if colour && (i == statusCol || i == urgencyCol) {
				if s, ok := statusStyles[cell]; ok {
					sb.WriteString(s.Render(padded))
					continue
				}
			}
			if colour {
				sb.WriteString(dimStyle.Render(padded))
			} else {
				sb.WriteString(padded)
			}
		}
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}
