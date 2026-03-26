package tui

import (
	"strings"
	"time"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/gechr/clib/human"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

func statusText(status string) string {
	switch status {
	case "triggered":
		return theme.UrgencyHigh.Render("triggered")
	case "acknowledged":
		return theme.UrgencyLow.Render("acknowledged")
	case "resolved":
		return lipgloss.NewStyle().Foreground(theme.Theme.Green.GetForeground()).Faint(true).Render("resolved")
	default:
		return status
	}
}

func incidentStyle(inc pagerduty.Incident) lipgloss.Style {
	if inc.Status == "resolved" {
		return theme.UrgencyResolved
	}
	if inc.Priority != nil {
		if s, ok := theme.PriorityStyle(inc.Priority.Name); ok {
			return s
		}
	}
	if inc.Urgency == "high" {
		return theme.UrgencyHigh
	}
	return theme.UrgencyLow
}

func severityLabel(inc pagerduty.Incident) string {
	if inc.Priority != nil && inc.Priority.Name != "" {
		return inc.Priority.Name
	}
	if inc.Urgency == "high" {
		return "high"
	}
	return "low"
}

func styledPriorityLabel(inc pagerduty.Incident) string {
	if inc.Priority != nil && inc.Priority.Name != "" {
		return severityStyle(inc.Priority.Name).Render(inc.Priority.Name)
	}
	if inc.Urgency == "high" {
		return theme.UrgencyHigh.Render("high")
	}
	return theme.DetailDim.Render("low")
}

func severityStyle(name string) lipgloss.Style {
	if s, ok := theme.PriorityStyle(name); ok {
		return s
	}
	return theme.DetailDim
}

func assigneeNames(assignments []pagerduty.Assignment) string {
	names := make([]string, 0, len(assignments))
	for _, a := range assignments {
		if a.Assignee.Summary != "" {
			names = append(names, a.Assignee.Summary)
		}
	}
	return strings.Join(names, ", ")
}

func renderTimeAgo(raw string) string {
	if raw == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return theme.Theme.RenderTimeAgoCompact(t, true)
}

func renderTimePlain(raw string) string {
	if raw == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return human.FormatTimeAgoCompact(t)
}

func formatTimeAbsolute(raw string) string {
	if raw == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.UTC().Format("02/01 15:04 UTC")
}

func wordWrap(text string, width int, indent string) string {
	if width < 10 {
		width = 10
	}
	var sb strings.Builder
	for paragraph := range strings.SplitSeq(text, "\n") {
		if paragraph == "" {
			sb.WriteString(indent + "\n")
			continue
		}
		words := strings.Fields(paragraph)
		lineLen := 0
		for i, w := range words {
			wLen := len([]rune(w))
			if i == 0 {
				sb.WriteString(indent + w)
				lineLen = wLen
			} else if lineLen+1+wLen > width {
				sb.WriteString("\n" + indent + w)
				lineLen = wLen
			} else {
				sb.WriteString(" " + w)
				lineLen += 1 + wLen
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

var (
	glamourCachedWidth    int
	glamourCachedRenderer *glamour.TermRenderer
)

func cachedGlamourRenderer(width int) *glamour.TermRenderer {
	if width == glamourCachedWidth && glamourCachedRenderer != nil {
		return glamourCachedRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	glamourCachedWidth = width
	glamourCachedRenderer = r
	return r
}

func renderMarkdown(text string, width int) string {
	if text == "" {
		return ""
	}
	if width <= 0 {
		width = 80
	}
	r := cachedGlamourRenderer(width)
	if r == nil {
		return wordWrap(text, width-4, "    ")
	}
	rendered, err := r.Render(text)
	if err != nil || strings.TrimSpace(rendered) == "" {
		return wordWrap(text, width-4, "    ")
	}
	return strings.TrimRight(rendered, "\n")
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}

type column struct {
	header string
	width  int
	min    int
}

var incidentColumns = []column{
	{"", 2, 0},
	{"Sev.", 4, 0},
	{"Title", 0, 20},
	{"Service", 14, 0},
	{"Assignees", 0, 12},
	{"Created", 12, 0},
	{"Updated", 12, 0},
}

const colService = 3

func visibleColumnIndices(totalWidth int) []int {
	switch {
	case totalWidth < 80:
		return []int{1, 2, 5}
	case totalWidth < 120:
		return []int{1, 2, 4, 5}
	case totalWidth < 160:
		return []int{1, 2, 3, 4, 5, 6}
	default:
		return []int{0, 1, 2, 3, 4, 5, 6}
	}
}

func layoutColumns(totalWidth int, cols []column, hide ...int) []int {
	widths := make([]int, len(cols))
	visible := make([]bool, len(cols))

	visIdx := visibleColumnIndices(totalWidth)
	allowed := make(map[int]bool, len(visIdx))
	for _, idx := range visIdx {
		if idx < len(cols) {
			allowed[idx] = true
		}
	}
	for i := range cols {
		visible[i] = allowed[i]
	}

	for _, idx := range hide {
		if idx < len(cols) {
			visible[idx] = false
		}
	}

	var hideOrder []int
	for i := len(cols) - 1; i > 0; i-- {
		if cols[i].width > 0 && visible[i] {
			hideOrder = append(hideOrder, i)
		}
	}

	for attempt := 0; attempt <= len(hideOrder); attempt++ {
		for i, c := range cols {
			if !visible[i] {
				widths[i] = 0
				continue
			}
			widths[i] = c.width
		}

		fixedSum, flexCount, flexMinSum, visibleCount := 0, 0, 0, 0
		for i, c := range cols {
			if !visible[i] {
				continue
			}
			visibleCount++
			if c.width > 0 {
				fixedSum += c.width
			} else {
				flexCount++
				flexMinSum += c.min
			}
		}

		separators := max(0, visibleCount-1)
		remaining := totalWidth - fixedSum - separators

		if remaining >= flexMinSum {
			if flexCount == 0 {
				break
			}

			flexIdx := make([]int, 0, flexCount)
			for i, c := range cols {
				if visible[i] && c.width == 0 {
					flexIdx = append(flexIdx, i)
				}
			}

			if len(flexIdx) == 2 {
				firstMin := cols[flexIdx[0]].min
				secondMin := cols[flexIdx[1]].min
				firstW := remaining * 60 / 100
				firstW = max(firstW, firstMin)
				secondW := remaining - firstW
				if secondW < secondMin {
					secondW = secondMin
					firstW = max(remaining-secondW, firstMin)
				}
				widths[flexIdx[0]] = firstW
				widths[flexIdx[1]] = secondW
			} else {
				each := remaining / flexCount
				for _, idx := range flexIdx {
					widths[idx] = max(each, cols[idx].min)
				}
			}
			break
		}

		if attempt >= len(hideOrder) {
			for i, c := range cols {
				if visible[i] && c.width == 0 {
					widths[i] = c.min
				}
			}
			break
		}
		visible[hideOrder[attempt]] = false
	}

	for i := range cols {
		if !visible[i] {
			widths[i] = 0
		}
	}

	return widths
}
