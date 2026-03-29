package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// binding describes a single keybinding entry.
type binding struct {
	key  string
	desc string
}

// section groups bindings under a heading.
type section struct {
	title    string
	bindings []binding
	footer   string
}

var listSections = []section{
	{
		title: "Actions",
		bindings: []binding{
			{"a", "acknowledge"},
			{"r *", "resolve"},
			{"e", "edit"},
			{"x *", "escalate"},
			{"m *", "merge selected"},
			{"s", "snooze"},
			{"n", "add note"},
		},
		footer: "* alt skips confirmation",
	},
	{
		title: "Navigation",
		bindings: []binding{
			{"↑↓", "navigate"},
			{"enter", "detail"},
			{"esc", "back / deselect"},
			{"space", "toggle selection"},
			{"shift+↑↓", "select and move"},
			{"ctrl+a", "select all"},
		},
	},
	{
		title: "Filters",
		bindings: []binding{
			{"/", "filter incidents"},
			{"O", "filter options"},
			{"t", "team switcher"},
		},
	},
	{
		title: "Other",
		bindings: []binding{
			{"o", "open in browser"},
			{"alt+o", "open external link"},
			{"y", "copy URL"},
			{"R", "toggle refresh"},
			{"?", "help"},
			{"q", "quit"},
		},
	},
}

var detailSections = []section{
	{
		title: "Actions",
		bindings: []binding{
			{"a", "acknowledge"},
			{"r *", "resolve"},
			{"e", "edit"},
			{"x *", "escalate"},
			{"n", "add note"},
			{"p", "set priority"},
		},
		footer: "* alt skips confirmation",
	},
	{
		title: "Navigation",
		bindings: []binding{
			{"↑↓", "scroll"},
			{"tab", "next tab"},
			{"shift+tab", "previous tab"},
			{"esc", "back to list"},
		},
	},
	{
		title: "Other",
		bindings: []binding{
			{"o", "open in browser"},
			{"alt+o", "open external link"},
			{"y", "copy URL"},
			{"?", "help"},
			{"q", "quit"},
		},
	},
}

// Help is a Bubble Tea model that renders a context-aware keybinding overlay.
type Help struct {
	Visible     bool
	CurrentView string // "dashboard" or "detail"
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

// View implements tea.Model.
func (h Help) View() tea.View {
	if !h.Visible {
		return tea.NewView("")
	}

	sections := listSections
	if h.CurrentView == "detail" {
		sections = detailSections
	}

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorTitleFg)
	dimDesc := lipgloss.NewStyle().Faint(true)

	var columns []string
	for _, sec := range sections {
		var sb strings.Builder
		sb.WriteString(sectionStyle.Render(sec.title))
		sb.WriteString("\n")
		for _, b := range sec.bindings {
			key := theme.HelpKey.Render(fmt.Sprintf("%-10s", b.key))
			desc := dimDesc.Render(b.desc)
			sb.WriteString(key + desc + "\n")
		}
		if sec.footer != "" {
			sb.WriteString(dimDesc.Render(sec.footer) + "\n")
		}
		columns = append(columns, sb.String())
	}

	// Two-column layout: pair sections side by side.
	var left, right string
	switch len(columns) {
	case 4:
		left = columns[0] + "\n" + columns[2]
		right = columns[1] + "\n" + columns[3]
	case 3:
		left = columns[0]
		right = columns[1] + "\n" + columns[2]
	case 2:
		left = columns[0]
		right = columns[1]
	case 1:
		left = columns[0]
	}

	var content string
	if right != "" {
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", right)
	} else {
		content = left
	}

	return tea.NewView(RenderOverlay(content, 0))
}
