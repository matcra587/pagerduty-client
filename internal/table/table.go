// Package table provides a builder-pattern table renderer with
// declarative column definitions, per-cell styling, relative time
// formatting, and OSC 8 hyperlinks.
package table

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/ansi"
	"github.com/gechr/clib/human"
	"github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/output"
)

const (
	colGap   = 2  // spaces between columns
	maxCellW = 60 // max cell width before truncation
)

// Table is a builder-pattern table renderer. Create one with [New],
// add columns with [Table.AddCol], append rows with [Table.Row],
// and call [Table.Render] to write the output.
type Table struct {
	cols []Column
	rows [][]string
	th   *theme.Theme
	w    io.Writer
}

// New creates a table that writes to w. When th is non-nil, the
// output includes ANSI styling (bold headers, dim default cells).
func New(w io.Writer, th *theme.Theme) *Table {
	return &Table{w: w, th: th}
}

// AddCol appends a column definition to the table.
func (t *Table) AddCol(c Column) *Table { t.cols = append(t.cols, c); return t }

// Row appends a data row. Values correspond positionally to the
// columns added with [Table.AddCol]. Extra values are silently
// ignored; missing values are treated as empty strings.
func (t *Table) Row(values ...string) *Table { t.rows = append(t.rows, values); return t }

// Render writes the formatted table to the configured writer. The
// output includes a header row followed by data rows, with columns
// padded to align. Returns any write error.
func (t *Table) Render() error {
	if len(t.cols) == 0 {
		return nil
	}

	// Phase 1: sanitise, format TimeAgo, and truncate cells.
	for _, row := range t.rows {
		for j := range row {
			if j >= len(t.cols) {
				break
			}
			cell := output.Sanitize(row[j])
			if t.cols[j].timeAgo {
				if ts, err := time.Parse(time.RFC3339, cell); err == nil {
					cell = human.FormatTimeAgoCompact(ts)
				}
			}
			if xansi.StringWidth(cell) > maxCellW {
				cell = xansi.Truncate(cell, maxCellW-3, "...")
			}
			row[j] = cell
		}
	}

	// Phase 2: measure column widths from display width.
	widths := make([]int, len(t.cols))
	for i, c := range t.cols {
		widths[i] = xansi.StringWidth(c.header)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if w := xansi.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Phase 3: build styles.
	colour := t.th != nil
	var headerStyle, dimStyle lipgloss.Style
	if colour {
		headerStyle = lipgloss.NewStyle().Bold(true)
		dimStyle = *t.th.Dim
	}

	var a *ansi.ANSI
	if colour {
		a = ansi.Force()
	}

	// Phase 4: render header + data rows.
	var sb strings.Builder

	for i, c := range t.cols {
		if i > 0 {
			sb.WriteString(strings.Repeat(" ", colGap))
		}
		padded := fmt.Sprintf("%-*s", widths[i], c.header)
		if colour {
			sb.WriteString(headerStyle.Render(padded))
		} else {
			sb.WriteString(padded)
		}
	}
	sb.WriteString("\n")

	for _, row := range t.rows {
		for i, col := range t.cols {
			if i > 0 {
				sb.WriteString(strings.Repeat(" ", colGap))
			}

			var cell string
			if i < len(row) {
				cell = row[i]
			}

			padded := fmt.Sprintf("%-*s", widths[i], cell)
			raw := cell
			padded = t.styleCell(col, raw, padded, dimStyle)

			if a != nil && col.linkFn != nil {
				if url := col.linkFn(raw); url != "" {
					if colour {
						padded = lipgloss.NewStyle().Underline(true).Render(padded)
					}
					padded = a.Hyperlink(url, padded)
				}
			}

			sb.WriteString(padded)
		}
		sb.WriteString("\n")
	}

	_, err := io.WriteString(t.w, sb.String())
	return err
}

// styleCell applies the column's configured styling to a padded cell value.
// raw is the unpadded cell text used for style lookups; padded is the
// width-adjusted string that gets rendered.
func (t *Table) styleCell(col Column, raw, padded string, dimStyle lipgloss.Style) string {
	switch {
	case col.styleFn != nil:
		s := col.styleFn(raw)
		if _, noColor := s.GetForeground().(lipgloss.NoColor); !noColor {
			return s.Render(padded)
		}
		return dimStyle.Render(padded)
	case col.bold:
		return lipgloss.NewStyle().Bold(true).Render(padded)
	case col.normal:
		return padded
	case col.linkFn != nil:
		return padded // no dim — underline applied in Render
	default:
		return dimStyle.Render(padded)
	}
}
