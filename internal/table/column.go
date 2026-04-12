package table

import "charm.land/lipgloss/v2"

// Column describes a single table column. Use [Col] to create one,
// then chain builder methods to configure styling, formatting and
// link behaviour. Columns are value types; every builder returns a
// new copy.
type Column struct {
	header  string
	flex    bool
	normal  bool
	bold    bool
	timeAgo bool
	styleFn func(string) lipgloss.Style
	linkFn  func(string) string
}

// Col creates a column with the given header text. Chain builder
// methods to configure display behaviour.
func Col(header string) Column { return Column{header: header} }

// Flex marks the column as flexible-width. Flex columns expand to
// fill remaining horizontal space after fixed columns are measured.
func (c Column) Flex() Column { c.flex = true; return c }

// Normal marks the column as unstyled. Normal columns are not
// wrapped in the dim style that is applied by default.
func (c Column) Normal() Column { c.normal = true; return c }

// Bold renders the column value in bold.
func (c Column) Bold() Column { c.bold = true; return c }

// TimeAgo treats the cell value as an RFC 3339 timestamp and
// formats it as a compact relative time string (e.g. "15m ago").
// Non-parseable values pass through unchanged.
func (c Column) TimeAgo() Column { c.timeAgo = true; return c }

// StyleMap returns a column that applies a lipgloss style looked
// up from m by the raw cell value. Unmatched values fall back to
// the theme's dim style.
func (c Column) StyleMap(m map[string]lipgloss.Style) Column {
	c.styleFn = func(v string) lipgloss.Style {
		if s, ok := m[v]; ok {
			return s
		}
		return lipgloss.Style{}
	}
	return c
}

// Style returns a column that calls fn with the raw cell value to
// obtain a per-cell lipgloss style. When fn returns a style with
// [lipgloss.NoColor] foreground, the cell falls back to the
// theme's dim style.
func (c Column) Style(fn func(string) lipgloss.Style) Column {
	c.styleFn = fn
	return c
}

// Link returns a column that wraps each cell in an OSC 8
// hyperlink. fn receives the raw cell value and returns the URL;
// return an empty string to skip the link for that cell.
func (c Column) Link(fn func(string) string) Column {
	c.linkFn = fn
	return c
}
