package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// binding describes a single keybinding entry.
type binding struct {
	key  string
	desc string
}

var bindings = []binding{
	{"j/k / ↑↓", "navigate"},
	{"g / G", "jump to top / bottom"},
	{"enter", "detail"},
	{"esc", "back / deselect"},
	{"space", "toggle selection"},
	{"ctrl+a", "select all"},
	{"a", "acknowledge"},
	{"r / alt+r", "resolve / immediate"},
	{"e / alt+e", "escalate / immediate"},
	{"m / alt+m", "merge selected / immediate"},
	{"s", "snooze"},
	{"n", "add note"},
	{"p", "set priority (detail)"},
	{"y", "copy URL"},
	{"/", "filter incidents"},
	{"O", "filter options"},
	{"o", "open in browser"},
	{"alt+o", "open external link"},
	{"R", "toggle refresh"},
	{"t", "team switcher"},
	{"?", "help"},
	{"q", "quit"},
}

// Help is a Bubble Tea model that renders a keybinding overlay.
// It is shown when Visible is true and overlaid on the parent view.
type Help struct {
	Visible bool
}

// Init implements tea.Model.
func (h Help) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (h Help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "?", "esc":
			h.Visible = false
		}
	}
	return h, nil
}

// View implements tea.Model. Returns an empty string when not visible.
func (h Help) View() tea.View {
	if !h.Visible {
		return tea.NewView("")
	}

	var sb strings.Builder
	sb.WriteString(theme.Title.Render("Keybindings"))
	sb.WriteString("\n\n")

	for _, b := range bindings {
		key := theme.HelpKey.Render(fmt.Sprintf("%-16s", b.key))
		desc := theme.HelpDesc.Render(b.desc)
		sb.WriteString(key + desc + "\n")
	}

	return tea.NewView(RenderOverlay(sb.String(), 0))
}
