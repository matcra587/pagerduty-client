package components

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// RenderOverlay wraps content in the standard overlay style with correct
// background rendering. It applies PersistBg to each line so the overlay
// background persists through ANSI SGR resets emitted by lipgloss Render.
func RenderOverlay(content string, minWidth int) string {
	bgEsc := ColorToANSIBg(theme.ColorOverlayBg)
	if bgEsc != "" {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = PersistBg(line, bgEsc)
		}
		content = strings.Join(lines, "\n")
	}

	contentW := lipgloss.Width(content)
	if contentW < minWidth {
		return theme.HelpOverlay.Width(minWidth).Render(content)
	}
	// Don't set Width when content is wider than minWidth - let the
	// overlay box size itself to fit. Setting Width to the content
	// width causes wrapping because HelpOverlay's Padding(1, 2)
	// reduces the inner area.
	return theme.HelpOverlay.Render(content)
}
