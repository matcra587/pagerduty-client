package output

import (
	"io"
	"strings"
	"text/tabwriter"
)

// RenderTable writes a tab-aligned text table to w.
// colour is reserved for future styled output.
func RenderTable(w io.Writer, headers []string, rows [][]string, colour bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	if _, err := io.WriteString(tw, strings.Join(headers, "\t")+"\n"); err != nil {
		return err
	}

	for _, row := range rows {
		if _, err := io.WriteString(tw, strings.Join(row, "\t")+"\n"); err != nil {
			return err
		}
	}

	return tw.Flush()
}
