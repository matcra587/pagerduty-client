package components

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// HintContext carries the state needed to select a contextual keybinding hint.
type HintContext struct {
	View          string
	HasSelections bool
	FilterActive  bool
	Paused        bool
}

// StatusBar is a Bubble Tea model that renders a one-line status bar showing
// the active team/user filter, incident counts, last refresh time and the
// current refresh state (active or paused).
type StatusBar struct {
	Team         string
	User         string
	Triggered    int
	Acknowledged int
	Resolved     int
	LastRefresh  time.Time
	Paused       bool
	Width        int
	StatusMsg    string
	FilterState  FilterState
	Hint         HintContext
}

// Init implements tea.Model.
func (s StatusBar) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (s StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return s, nil }

// View implements tea.Model.
func (s StatusBar) View() tea.View {
	refresh := s.refreshLabel()

	// Build the labelled separator: ── ? help ──────────── ↻ 22s ──
	helpLabel := theme.HelpKey.Render("?") + " " + theme.HelpDesc.Render("help")
	borderStyle := lipgloss.NewStyle().Foreground(theme.ColorOverlayBorder)
	sep := LabelledBorder(s.Width, borderStyle, helpLabel, refresh)

	// Hint bar with keybindings.
	var hint string
	if s.StatusMsg != "" {
		hint = s.StatusMsg
	} else {
		hint = s.hintView()
	}

	hint = lipgloss.PlaceHorizontal(s.Width, lipgloss.Center, hint)
	return tea.NewView(sep + "\n" + hint)
}

// LabelledBorder renders a horizontal border line with left and right labels
// embedded in it: ── left ──────────────────── right ──
func LabelledBorder(width int, borderStyle lipgloss.Style, left, right string) string {
	rule := borderStyle.Render("─")
	gap := borderStyle.Render(" ")

	leftPart := rule + rule + gap + left + gap
	rightPart := gap + right + gap + rule + rule

	leftW := lipgloss.Width(leftPart)
	rightW := lipgloss.Width(rightPart)
	fillW := max(width-leftW-rightW, 0)
	fill := borderStyle.Render(strings.Repeat("─", fillW))

	return leftPart + fill + rightPart
}

// hintView renders keybinding hints on a single line. Bindings are added
// left to right until the line is full.
func (s StatusBar) hintView() string {
	if s.Width <= 0 {
		return ""
	}

	bindings := s.hintBindings()

	const sep = "  "
	sepW := lipgloss.Width(sep)

	renderBinding := func(b key.Binding) string {
		k := theme.HelpKey.Render(b.Help().Key)
		d := theme.HelpDesc.Render(b.Help().Desc)
		return k + " " + d
	}

	var parts []string
	usedW := 0
	for _, b := range bindings {
		rendered := renderBinding(b)
		w := lipgloss.Width(rendered)
		needed := w
		if len(parts) > 0 {
			needed += sepW
		}
		if usedW+needed > s.Width {
			ellipsis := theme.HelpDesc.Render("...")
			if usedW+lipgloss.Width(ellipsis) <= s.Width {
				parts = append(parts, ellipsis)
			}
			break
		}
		parts = append(parts, rendered)
		usedW += needed
	}

	return strings.Join(parts, sep)
}

// hintBindings returns the keybindings for the current context.
func (s StatusBar) hintBindings() []key.Binding {
	bind := func(k, desc string) key.Binding {
		return key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc))
	}

	switch {
	case s.Hint.FilterActive:
		return []key.Binding{
			bind("type", "filter"),
			bind("enter", "commit"),
			bind("esc", "clear"),
		}

	case s.Hint.HasSelections:
		return []key.Binding{
			bind("space", "select"),
			bind("a", "ack"),
			bind("r", "resolve"),
			bind("m", "merge"),
			bind("esc", "clear"),
			bind("ctrl+a", "all"),
			bind("alt+r", "resolve now"),
			bind("alt+m", "merge now"),
		}

	case s.Hint.View == "detail":
		return []key.Binding{
			bind("a", "ack"),
			bind("r", "resolve"),
			bind("esc", "back"),
			bind("↑↓", "scroll"),
			bind("n", "note"),
			bind("p", "priority"),
			bind("o", "open"),
			bind("y", "copy URL"),
			bind("alt+r", "resolve now"),
			bind("alt+o", "external"),
		}

	default:
		return []key.Binding{
			bind("a", "ack"),
			bind("r", "resolve"),
			bind("enter", "show"),
			bind("/", "filter"),
			bind("space", "select"),
			bind("e", "escalate"),
			bind("n", "note"),
			bind("o", "open"),
			bind("O", "options"),
			bind("t", "team"),
			bind("R", "refresh"),
			bind("q", "quit"),
		}
	}
}

func (s StatusBar) refreshLabel() string {
	if s.Paused {
		return theme.Paused.Render("⏸  paused")
	}
	if s.LastRefresh.IsZero() {
		return theme.Active.Render("↻")
	}
	secs := int(time.Since(s.LastRefresh).Seconds())
	return theme.Active.Render("↻") + " " + strconv.Itoa(secs) + "s"
}
