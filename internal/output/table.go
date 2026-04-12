package output

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/ansi"
	"github.com/gechr/clib/theme"
)

const (
	colGap   = 2  // spaces between columns
	maxCellW = 60 // max cell width before truncation
)

// statusStylesFromTheme builds a style map for status and urgency
// values from the resolved theme.
func statusStylesFromTheme(th *theme.Theme) map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"triggered":    lipgloss.NewStyle().Foreground(th.Red.GetForeground()),
		"acknowledged": lipgloss.NewStyle().Foreground(th.Yellow.GetForeground()),
		"resolved":     lipgloss.NewStyle().Foreground(th.Green.GetForeground()),
		"high":         lipgloss.NewStyle().Foreground(th.Red.GetForeground()),
		"low":          lipgloss.NewStyle().Foreground(th.Yellow.GetForeground()),
	}
}

// TableOption configures optional RenderTable behaviour.
type TableOption func(*tableOpts)

type tableOpts struct {
	linkFn func(col int, value string) string
}

// WithLinkFunc adds OSC8 hyperlinks to table cells. The function
// receives the column index and the raw (sanitised) cell value.
// Return a URL to wrap the cell as a hyperlink, or empty string
// to skip.
func WithLinkFunc(fn func(col int, value string) string) TableOption {
	return func(o *tableOpts) { o.linkFn = fn }
}

// RenderTable writes a column-aligned text table to w.
// When th is non-nil, headers are bold and status/urgency values
// use theme colours. Cell values are sanitised and long values
// are truncated.
func RenderTable(w io.Writer, headers []string, rows [][]string, th *theme.Theme, opts ...TableOption) error {
	var cfg tableOpts
	for _, o := range opts {
		o(&cfg)
	}
	if len(headers) == 0 {
		return nil
	}

	// Sanitise and truncate cells (before measuring).
	for _, row := range rows {
		for j, cell := range row {
			cell = Sanitize(cell)
			if xansi.StringWidth(cell) > maxCellW {
				cell = xansi.Truncate(cell, maxCellW-3, "...")
			}
			row[j] = cell
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

	colour := th != nil
	var styles map[string]lipgloss.Style
	var headerStyle, dimStyle lipgloss.Style
	if colour {
		styles = statusStylesFromTheme(th)
		headerStyle = lipgloss.NewStyle().Bold(true)
		dimStyle = *th.Dim
	}

	var a *ansi.ANSI
	if colour && cfg.linkFn != nil {
		a = ansi.Force()
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

			var linkURL string
			if cfg.linkFn != nil {
				linkURL = cfg.linkFn(i, cell)
			}

			if colour && (i == statusCol || i == urgencyCol) {
				if s, ok := styles[cell]; ok {
					padded = s.Render(padded)
				} else {
					padded = dimStyle.Render(padded)
				}
			} else if colour && linkURL != "" {
				padded = lipgloss.NewStyle().Underline(true).Render(padded)
			} else if colour {
				padded = dimStyle.Render(padded)
			}
			if a != nil && linkURL != "" {
				padded = a.Hyperlink(linkURL, padded)
			}
			sb.WriteString(padded)
		}
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}
