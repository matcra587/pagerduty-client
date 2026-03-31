package output

import (
	"fmt"
	"io"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/theme"
)

// RenderColumns writes items in a multi-column flow layout similar to ls.
// Items fill down each column before moving to the next. When th is non-nil,
// items are styled with the theme's Orange colour.
func RenderColumns(w io.Writer, items []string, termWidth int, th *theme.Theme) error {
	if len(items) == 0 {
		return nil
	}

	if termWidth <= 0 {
		termWidth = 80
	}

	// Find the widest item.
	maxW := 0
	for _, item := range items {
		if itemW := xansi.StringWidth(item); itemW > maxW {
			maxW = itemW
		}
	}

	colWidth := maxW + colGap
	cols := max(termWidth/colWidth, 1)

	rows := (len(items) + cols - 1) / cols

	var sb strings.Builder
	for r := range rows {
		var line strings.Builder
		for c := range cols {
			idx := c*rows + r
			if idx >= len(items) {
				continue
			}
			item := items[idx]
			padded := fmt.Sprintf("%-*s", colWidth, item)

			if th != nil && th.Orange != nil {
				line.WriteString(th.Orange.Render(padded))
			} else {
				line.WriteString(padded)
			}
		}
		sb.WriteString(strings.TrimRight(line.String(), " "))
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}
