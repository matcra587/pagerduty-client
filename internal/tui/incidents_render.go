package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/x/ansi"
	"github.com/matcra587/pagerduty-client/internal/integration"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// badgeFgStyle returns a foreground style for a badge value based on
// common monitoring state names. Matching is case-insensitive.
func badgeFgStyle(value string) lipgloss.Style {
	switch strings.ToLower(value) {
	case "triggered", "alert", "alarm":
		return lipgloss.NewStyle().Foreground(theme.Theme.Red.GetForeground()).Bold(true)
	case "ok", "resolved", "closed":
		return lipgloss.NewStyle().Foreground(theme.Theme.Green.GetForeground()).Bold(true)
	case "warning", "warn":
		return lipgloss.NewStyle().Foreground(theme.Theme.Yellow.GetForeground()).Bold(true)
	case "open", "incident", "incident_opened", "incident_closed":
		return lipgloss.NewStyle().Foreground(theme.Theme.Orange.GetForeground()).Bold(true)
	case "no data", "insufficient_data":
		return lipgloss.NewStyle().Foreground(theme.Theme.Orange.GetForeground())
	case "muted":
		return lipgloss.NewStyle().Faint(true)
	default:
		return lipgloss.NewStyle().Faint(true)
	}
}

// renderBadge renders a single field as a coloured foreground pill.
func renderBadge(f integration.Field) string {
	if f.Value == "" {
		return ""
	}
	return badgeFgStyle(f.Value).Padding(0, 1).Render(f.Value)
}

// renderHeaderRow composes the source label, badge pills and clickable
// links into a single header line. Links use OSC 8 hyperlinks when
// a is non-nil; otherwise they render as styled label text.
func renderHeaderRow(source string, badges []integration.Field, links []integration.Link, a *ansi.ANSI) string {
	parts := []string{theme.DetailHeader.Render(source)}
	for _, b := range badges {
		if pill := renderBadge(b); pill != "" {
			parts = append(parts, pill)
		}
	}
	linkStyle := lipgloss.NewStyle().
		Foreground(theme.Theme.Blue.GetForeground()).
		Underline(true)
	for _, l := range links {
		label := l.Label
		if label == "" {
			label = "Link"
		}
		styled := linkStyle.Render(label)
		if a != nil {
			styled = a.Hyperlink(l.URL, styled)
		}
		parts = append(parts, styled)
	}
	return strings.Join(parts, "  ")
}

// renderCodeBlock renders a field as a left-bordered highlighted block
// with the label as a dim header and the value in a code-friendly colour.
func renderCodeBlock(f integration.Field, width int) string {
	if f.Value == "" {
		return ""
	}
	label := theme.DetailDim.Render(f.Label)
	value := lipgloss.NewStyle().
		Foreground(theme.Theme.Green.GetForeground()).
		Render(f.Value)

	content := label + "\n" + value

	blockWidth := max(width-4, 10)
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderTop(false).
		BorderRight(false).
		BorderBottom(false).
		BorderLeftForeground(theme.DetailHeader.GetForeground()).
		Padding(0, 1).
		Width(blockWidth).
		Render(content)
}

// renderMarkdownField renders a field's value through glamour markdown
// rendering with no label prefix.
func renderMarkdownField(f integration.Field, width int) string {
	if f.Value == "" {
		return ""
	}
	return renderMarkdown(f.Value, width)
}

// renderTagPills splits a comma-separated tag string into individually
// styled pills and wraps them across lines when they exceed width.
func renderTagPills(f integration.Field, width int) string {
	if f.Value == "" {
		return ""
	}

	pillStyle := lipgloss.NewStyle().
		Foreground(theme.Theme.Blue.GetForeground()).
		Bold(true).
		Padding(0, 1)

	tags := strings.Split(f.Value, ",")
	var lines []string
	var line []string
	lineWidth := 0

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		pill := pillStyle.Render(tag)
		pillWidth := lipgloss.Width(pill) + 1
		if lineWidth > 0 && lineWidth+pillWidth > width {
			lines = append(lines, strings.Join(line, " "))
			line = nil
			lineWidth = 0
		}
		line = append(line, pill)
		lineWidth += pillWidth
	}
	if len(line) > 0 {
		lines = append(lines, strings.Join(line, " "))
	}

	return strings.Join(lines, "\n")
}

// groupFieldsByType partitions fields into type buckets, preserving
// emission order within each bucket.
func groupFieldsByType(fields []integration.Field) map[integration.FieldType][]integration.Field {
	groups := make(map[integration.FieldType][]integration.Field)
	for _, f := range fields {
		groups[f.Type] = append(groups[f.Type], f)
	}
	return groups
}
