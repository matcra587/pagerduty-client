package components

import (
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// RenderOverlay wraps content in the standard overlay style (bordered,
// padded, no background). The overlay inherits the terminal's default
// background so it works with any colour scheme.
func RenderOverlay(content string, minWidth int) string {
	contentW := lipgloss.Width(content)
	if contentW < minWidth {
		return theme.HelpOverlay.Width(minWidth).Render(content)
	}
	return theme.HelpOverlay.Render(content)
}
