package components

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
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
	filter := s.filterLabel()
	counts := s.countsLabel()
	refresh := s.refreshLabel()

	// PersistBg operates on a single line. This is safe here because the
	// StatusBar style has no width/height constraints (only Padding(0,1)) and
	// the label helpers return plain inline strings with no embedded newlines,
	// so Render always produces a single line.
	bgEsc := ColorToANSIBg(theme.ColorStatusBarBg)
	left := PersistBg(theme.StatusBar.Render(filter+"  "+counts), bgEsc)
	right := PersistBg(theme.StatusBar.Render(refresh), bgEsc)

	gap := max(s.Width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	spacer := theme.StatusBar.Render(fmt.Sprintf("%*s", gap, ""))

	var hint string
	if s.StatusMsg != "" {
		hint = s.StatusMsg
	} else {
		hint = s.hintView()
	}

	hintStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.ColorOverlayBorder).
		Width(s.Width)
	hint = hintStyle.Render(hint)

	return tea.NewView(hint + "\n" + left + spacer + right)
}

// hintView uses bubbles' help.Model to render as many keybinding hints
// as fit in the available width, truncating with an ellipsis when needed.
func (s StatusBar) hintView() string {
	h := help.New()
	h.ShortSeparator = "  "
	// Account for the border's padding (2 chars left + 2 right).
	h.SetWidth(s.Width - 4)
	h.Styles = help.Styles{
		ShortKey:       theme.HelpKey,
		ShortDesc:      theme.HelpDesc,
		ShortSeparator: theme.HelpDesc,
		Ellipsis:       theme.HelpDesc,
	}
	return h.ShortHelpView(s.hintBindings())
}

// hintBindings returns the keybindings for the current context, ordered
// by priority (most important first). The help model shows as many as fit.
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
			bind("↑↓", "scroll"),
			bind("a", "ack"),
			bind("r", "resolve"),
			bind("o", "open"),
			bind("esc", "back"),
			bind("p", "priority"),
			bind("n", "note"),
			bind("y", "copy URL"),
			bind("alt+o", "external"),
			bind("alt+r", "resolve now"),
		}

	default:
		return []key.Binding{
			bind("enter", "show"),
			bind("space", "select"),
			bind("/", "filter"),
			bind("O", "options"),
			bind("?", "help"),
			bind("q", "quit"),
			bind("a", "ack"),
			bind("r", "resolve"),
			bind("e", "escalate"),
			bind("t", "team"),
			bind("R", "refresh"),
			bind("y", "copy URL"),
			bind("o", "open"),
		}
	}
}

func (s StatusBar) filterLabel() string {
	if s.Team != "" {
		return "team:" + s.Team
	}
	if s.User != "" {
		return "user:" + s.User
	}
	return "all"
}

func (s StatusBar) countsLabel() string {
	base := fmt.Sprintf(
		"triggered:%d  acked:%d  resolved:%d",
		s.Triggered, s.Acknowledged, s.Resolved,
	)
	if n := s.FilterState.ActiveCount(); n > 0 {
		badge := fmt.Sprintf("F:%d", n)
		chips := s.FilterState.ChipSummary()
		base += "  " + theme.HelpKey.Render(badge) + " " + theme.HelpDesc.Render(chips)
	}
	return base
}

func (s StatusBar) refreshLabel() string {
	var indicator string
	if s.Paused {
		indicator = theme.Paused.Render("⏸ paused")
	} else {
		indicator = theme.Active.Render("↻ active")
	}

	var age string
	if s.LastRefresh.IsZero() {
		age = "never"
	} else {
		age = fmt.Sprintf("%ds ago", int(time.Since(s.LastRefresh).Seconds()))
	}

	return indicator + theme.StatusBar.Render("  refreshed: "+age)
}
