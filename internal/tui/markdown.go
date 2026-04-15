package tui

import (
	"strings"
	"sync"

	"charm.land/glamour/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// glamourStyle maps a clib theme preset name to a glamour style.
// Glamour has limited presets (no catppuccin), so dark themes map
// to "dracula" and light themes map to "light".
func glamourStyle(name string) string {
	switch name {
	case "dracula":
		return "dracula"
	case "catppuccin-latte":
		return "light"
	case "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha":
		return "dark"
	case "monochrome":
		return "notty"
	default:
		return "dracula"
	}
}

var (
	glamourCachedStyle    string
	glamourCachedWidth    int
	glamourCachedRenderer *glamour.TermRenderer
	glamourRenderMu       sync.Mutex
)

func cachedGlamourRenderer(width int) *glamour.TermRenderer {
	style := glamourStyle(theme.Theme.String())
	if width == glamourCachedWidth && style == glamourCachedStyle && glamourCachedRenderer != nil {
		return glamourCachedRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	glamourCachedStyle = style
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

	glamourRenderMu.Lock()
	defer glamourRenderMu.Unlock()

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
