// Package theme defines lipgloss styles shared across TUI components.
package theme

import (
	"hash/fnv"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	clibtheme "github.com/gechr/clib/theme"
)

var applyOnce sync.Once

// presetNames lists the clib theme presets available for selection.
// Sorted alphabetically for completion output.
var presetNames = []string{
	"catppuccin-frappe",
	"catppuccin-latte",
	"catppuccin-macchiato",
	"catppuccin-mocha",
	"default",
	"dracula",
	"monochrome",
	"monokai",
}

// PresetNames returns the sorted list of available theme preset names.
func PresetNames() []string {
	return presetNames
}

// Resolve returns a clib theme for the given preset name. An empty or
// unrecognised name returns theme.Default() (which itself checks the
// PDC_THEME / CLIB_THEME env vars before falling back to the built-in
// default).
func Resolve(name string) *clibtheme.Theme {
	if name == "" {
		return clibtheme.Default()
	}
	var th clibtheme.Theme
	if err := th.UnmarshalText([]byte(name)); err != nil {
		return clibtheme.Default()
	}
	return &th
}

// Theme is the shared clib theme instance used as the colour foundation.
var Theme = clibtheme.Default()

// Urgency styles - derived from clib semantic colours.
var (
	UrgencyHigh     = lipgloss.NewStyle().Foreground(Theme.Red.GetForeground()).Bold(true)
	UrgencyLow      = lipgloss.NewStyle().Foreground(Theme.Yellow.GetForeground())
	UrgencyResolved = lipgloss.NewStyle().Foreground(Theme.Dim.GetForeground()).Faint(true)
)

// PriorityStyles maps PagerDuty priority names to lipgloss styles.
// Lookup via PriorityStyle which falls back to urgency-based styling.
var PriorityStyles = map[string]lipgloss.Style{
	"P1": lipgloss.NewStyle().Foreground(Theme.Red.GetForeground()).Bold(true),
	"P2": lipgloss.NewStyle().Foreground(Theme.Orange.GetForeground()).Bold(true),
	"P3": lipgloss.NewStyle().Foreground(Theme.Yellow.GetForeground()),
	"P4": lipgloss.NewStyle().Foreground(Theme.Blue.GetForeground()),
	"P5": lipgloss.NewStyle().Faint(true),
}

// PriorityStyle returns the style for a PagerDuty priority name.
// Returns the style and true if found, or zero style and false if not.
func PriorityStyle(name string) (lipgloss.Style, bool) {
	s, ok := PriorityStyles[name]
	return s, ok
}

// Status flash styles - for action feedback in the status bar.
var (
	// StatusOK styles success feedback in the status bar.
	StatusOK = lipgloss.NewStyle().Foreground(Theme.Green.GetForeground()).Bold(true)
	// StatusErr styles error feedback in the status bar.
	StatusErr = lipgloss.NewStyle().Foreground(Theme.Red.GetForeground()).Bold(true)
)

// Pill renders a compact label with horizontal padding. Use with a
// foreground or background colour for a coloured pill effect.
var Pill = lipgloss.NewStyle().Padding(0, 1)

// PillDanger is a pill for critical counts (triggered incidents).
var PillDanger = Pill.Foreground(Theme.Red.GetForeground()).Bold(true)

// PillWarning is a pill for warning counts (acknowledged incidents).
var PillWarning = Pill.Foreground(Theme.Yellow.GetForeground())

// PillDim is a pill for inactive or resolved counts.
var PillDim = Pill.Faint(true)

// UI chrome colours - project-specific, no clib equivalent.
var (
	// ColorStatusBarFg is the status bar foreground colour.
	ColorStatusBarFg = lipgloss.Color("#E0E0E0")
	// ColorTitleFg is the title text foreground colour.
	ColorTitleFg = lipgloss.Color("#FFFFFF")
	// ColorHighlightBg is the background colour for highlighted items.
	ColorHighlightBg = lipgloss.Color("#2D2D44")
	// ColorHighlightFg is the foreground colour for highlighted items.
	ColorHighlightFg = lipgloss.Color("#FFFFFF")
	// ColorHeaderFg is the header text foreground colour.
	ColorHeaderFg = lipgloss.Color("#A0A0C0")
	// ColorOverlayBg is the overlay background colour.
	ColorOverlayBg = lipgloss.Color("#222233")
	// ColorOverlayBorder is the overlay border colour.
	ColorOverlayBorder = lipgloss.Color("#7F849C")
	// ColorSelectedBg is the background colour for selected items.
	ColorSelectedBg = lipgloss.Color("#313244")
)

// TableHeader is the style for table column headers.
var TableHeader = lipgloss.NewStyle().
	Foreground(ColorHeaderFg).
	Bold(true).
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true)

// SelectedRow is the style for the active/selected table row.
var SelectedRow = lipgloss.NewStyle().
	Background(ColorHighlightBg).
	Foreground(ColorHighlightFg).
	Bold(true)

// Title is the style for section and panel titles.
var Title = lipgloss.NewStyle().
	Foreground(ColorTitleFg).
	Bold(true).
	Padding(0, 1)

// HelpOverlay is the outer container style for the help overlay.
var HelpOverlay = lipgloss.NewStyle().
	Background(ColorOverlayBg).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorOverlayBorder).
	Padding(1, 2)

// HelpKey is the style for keybinding key labels.
var HelpKey = lipgloss.NewStyle().Foreground(Theme.Yellow.GetForeground()).Bold(true)

// HelpDesc is the style for keybinding descriptions.
var HelpDesc = lipgloss.NewStyle().Foreground(ColorStatusBarFg).Faint(true)

// Detail view styles - derived from clib theme colours.
var (
	// DetailHeader styles section headers in the detail view.
	DetailHeader = lipgloss.NewStyle().Bold(true).Foreground(Theme.Magenta.GetForeground())
	// DetailLabel styles field labels in the detail view.
	DetailLabel = lipgloss.NewStyle().Bold(true).Foreground(Theme.Green.GetForeground())
	// DetailValue styles field values in the detail view.
	DetailValue = lipgloss.NewStyle().Foreground(Theme.MarkdownText.GetForeground())
	// DetailDim styles de-emphasised text in the detail view.
	DetailDim = lipgloss.NewStyle().Faint(true)
)

// Paused is the style for the "paused" refresh indicator.
var Paused = lipgloss.NewStyle().Foreground(Theme.Red.GetForeground()).Bold(true)

// Active is the style for the "active" refresh indicator.
var Active = lipgloss.NewStyle().Foreground(Theme.Green.GetForeground()).Bold(true)

// Apply sets the base clib theme and reinitialises all derived styles.
// It runs at most once; subsequent calls are no-ops.
func Apply(t *clibtheme.Theme) {
	applyOnce.Do(func() {
		applyTheme(t)
	})
}

func applyTheme(t *clibtheme.Theme) {
	Theme = t

	// Urgency styles.
	UrgencyHigh = lipgloss.NewStyle().Foreground(t.Red.GetForeground()).Bold(true)
	UrgencyLow = lipgloss.NewStyle().Foreground(t.Yellow.GetForeground())
	UrgencyResolved = lipgloss.NewStyle().Foreground(t.Dim.GetForeground()).Faint(true)

	// Priority styles.
	PriorityStyles = map[string]lipgloss.Style{
		"P1": lipgloss.NewStyle().Foreground(t.Red.GetForeground()).Bold(true),
		"P2": lipgloss.NewStyle().Foreground(t.Orange.GetForeground()).Bold(true),
		"P3": lipgloss.NewStyle().Foreground(t.Yellow.GetForeground()),
		"P4": lipgloss.NewStyle().Foreground(t.Blue.GetForeground()),
		"P5": lipgloss.NewStyle().Faint(true),
	}

	// Status flash styles.
	StatusOK = lipgloss.NewStyle().Foreground(t.Green.GetForeground()).Bold(true)
	StatusErr = lipgloss.NewStyle().Foreground(t.Red.GetForeground()).Bold(true)

	// UI chrome colours - preset-specific overrides.
	applyChrome(t)

	// Compound styles that depend on chrome colours.
	TableHeader = lipgloss.NewStyle().
		Foreground(ColorHeaderFg).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)
	SelectedRow = lipgloss.NewStyle().
		Background(ColorHighlightBg).
		Foreground(ColorHighlightFg).
		Bold(true)
	Title = lipgloss.NewStyle().
		Foreground(ColorTitleFg).
		Bold(true).
		Padding(0, 1)
	HelpOverlay = lipgloss.NewStyle().
		Background(ColorOverlayBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOverlayBorder).
		Padding(1, 2)
	HelpKey = lipgloss.NewStyle().Foreground(t.Yellow.GetForeground()).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(ColorStatusBarFg).Faint(true)

	// Pill styles.
	PillDanger = Pill.Foreground(t.Red.GetForeground()).Bold(true)
	PillWarning = Pill.Foreground(t.Yellow.GetForeground())
	PillDim = Pill.Faint(true)

	// Detail view styles.
	DetailHeader = lipgloss.NewStyle().Bold(true).Foreground(t.Magenta.GetForeground())
	DetailLabel = lipgloss.NewStyle().Bold(true).Foreground(t.Green.GetForeground())
	DetailValue = lipgloss.NewStyle().Foreground(t.MarkdownText.GetForeground())
	DetailDim = lipgloss.NewStyle().Faint(true)

	// Indicator styles.
	Paused = lipgloss.NewStyle().Foreground(t.Red.GetForeground()).Bold(true)
	Active = lipgloss.NewStyle().Foreground(t.Green.GetForeground()).Bold(true)
}

// applyChrome sets UI chrome colours based on the active theme. The light
// theme needs inverted chrome (dark text on light backgrounds) while dark
// and high-contrast share the same dark-background chrome.
func applyChrome(t *clibtheme.Theme) {
	if isLightTheme(t) {
		// Light theme chrome.
		ColorStatusBarFg = lipgloss.Color("#2E3440")
		ColorTitleFg = lipgloss.Color("#2E3440")
		ColorHighlightBg = lipgloss.Color("#C8CED8")
		ColorHighlightFg = lipgloss.Color("#2E3440")
		ColorHeaderFg = lipgloss.Color("#4C566A")
		ColorOverlayBg = lipgloss.Color("#ECEFF4")
		ColorOverlayBorder = lipgloss.Color("#81A1C1")
		ColorSelectedBg = lipgloss.Color("#D8E8D8")
	} else {
		// Dark / high-contrast theme chrome.
		ColorStatusBarFg = lipgloss.Color("#E0E0E0")
		ColorTitleFg = lipgloss.Color("#FFFFFF")
		ColorHighlightBg = lipgloss.Color("#2D2D44")
		ColorHighlightFg = lipgloss.Color("#FFFFFF")
		ColorHeaderFg = lipgloss.Color("#A0A0C0")
		ColorOverlayBg = lipgloss.Color("#222233")
		ColorOverlayBorder = lipgloss.Color("#7F849C")
		ColorSelectedBg = lipgloss.Color("#313244")
	}
}

// isLightTheme returns true if the theme uses a light background. Known
// light presets are matched by name via Theme.String(). Unknown themes
// fall back to a luminance heuristic on the MarkdownText foreground.
func isLightTheme(t *clibtheme.Theme) bool {
	switch t.String() {
	case "catppuccin-latte":
		return true
	default:
		fg := t.MarkdownText.GetForeground()
		r, g, b, _ := fg.RGBA()
		luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
		return luminance < 0.5
	}
}

// EntityColor returns a consistent colour for a named entity by hashing
// into the clib theme's 20-colour palette.
func EntityColor(name string) lipgloss.Style {
	if name == "" {
		return lipgloss.NewStyle()
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	colors := Theme.EntityColors
	if len(colors) == 0 {
		return lipgloss.NewStyle()
	}
	//nolint:gosec // palette length is always small and positive
	c := colors[h.Sum32()%uint32(len(colors))]
	return lipgloss.NewStyle().Foreground(c)
}

// RenderEntityNames colours each name individually using EntityColor
// and joins the results with ", ". Empty and whitespace-only elements
// are skipped. Callers are responsible for sanitising names before
// passing them in.
func RenderEntityNames(names []string) string {
	var styled []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		styled = append(styled, EntityColor(name).Render(name))
	}
	return strings.Join(styled, ", ")
}
