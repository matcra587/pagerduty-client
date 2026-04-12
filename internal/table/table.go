// Package table provides a builder-pattern table renderer with
// declarative column definitions, per-cell styling, relative time
// formatting, and OSC 8 hyperlinks.
package table

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/ansi"
	"github.com/gechr/clib/human"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/output"
)

const (
	colGap   = 2  // spaces between columns
	maxCellW = 60 // max cell width for fixed columns
	minFlexW = 10 // minimum width per flex column after truncation
)

// termWidth returns the current terminal width. Package-level
// variable so tests can override it without a real TTY.
var termWidth = func() int { return terminal.Width(os.Stdout) }

// Table is a builder-pattern table renderer. Create one with [New],
// add columns with [Table.AddCol], append rows with [Table.Row],
// and call [Table.Render] to write the output.
type Table struct {
	cols      []Column
	rows      [][]string
	th        *theme.Theme
	w         io.Writer
	unbounded bool
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

// Unbounded disables flex column truncation. Flex cells render
// at their full natural width regardless of terminal size.
func (t *Table) Unbounded() *Table { t.unbounded = true; return t }

// Render writes the formatted table to the configured writer. The
// output includes a header row followed by data rows, with columns
// padded to align. Returns any write error.
func (t *Table) Render() error {
	if len(t.cols) == 0 {
		return nil
	}

	// Phase 1: sanitise, collapse whitespace, format TimeAgo,
	// truncate fixed columns to maxCellW.
	for _, row := range t.rows {
		for j := range row {
			if j >= len(t.cols) {
				break
			}
			cell := output.Sanitize(row[j])
			cell = strings.Join(strings.Fields(cell), " ")
			if t.cols[j].timeAgo {
				if ts, err := time.Parse(time.RFC3339, cell); err == nil {
					if t.th != nil {
						cell = t.th.RenderTimeAgoCompact(ts, true)
					} else {
						cell = human.FormatTimeAgoCompact(ts)
					}
				}
			}
			if !t.cols[j].flex && xansi.StringWidth(cell) > maxCellW {
				cell = xansi.Truncate(cell, maxCellW-3, "...")
			}
			row[j] = cell
		}
	}

	// Phase 2a: measure fixed column widths.
	widths := make([]int, len(t.cols))
	flexCount := 0
	for i, c := range t.cols {
		widths[i] = xansi.StringWidth(c.header)
		if c.flex {
			flexCount++
		}
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) || t.cols[i].flex {
				continue
			}
			if w := xansi.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Phase 2b: distribute remaining terminal width across flex
	// columns. Skipped when Unbounded() or piped (termW == 0).
	if !t.unbounded && t.th != nil && flexCount > 0 {
		if termW := termWidth(); termW > 0 {
			fixedTotal := 0
			for i, c := range t.cols {
				if !c.flex {
					fixedTotal += widths[i]
				}
			}
			gaps := (len(t.cols) - 1) * colGap
			available := termW - fixedTotal - gaps
			perFlex := max(available/flexCount, minFlexW)
			for _, row := range t.rows {
				for j := range row {
					if j >= len(t.cols) || !t.cols[j].flex {
						continue
					}
					if xansi.StringWidth(row[j]) > perFlex {
						row[j] = xansi.Truncate(row[j], perFlex-3, "...")
					}
				}
			}
		}
	}

	// Phase 2c: measure flex column widths from (possibly truncated) cells.
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) || !t.cols[i].flex {
				continue
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
