package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// PrioritySelected is sent when the user picks a priority.
type PrioritySelected struct {
	IncidentID string
	Priority   string // "P1"-"P5" display name
}

// PriorityPickerClosed is sent when the user cancels the picker.
type PriorityPickerClosed struct{}

// PriorityPicker is a small overlay for selecting P1-P5.
// Navigation: up/down arrows or j/k, enter to confirm, esc to cancel.
type PriorityPicker struct {
	Visible    bool
	incidentID string
	choices    []string
	cursor     int
}

// NewPriorityPicker returns a PriorityPicker with P1-P5 choices.
func NewPriorityPicker() PriorityPicker {
	return PriorityPicker{
		choices: []string{"P1", "P2", "P3", "P4", "P5"},
	}
}

// Show makes the picker visible for the given incident.
func (p PriorityPicker) Show(incidentID string) PriorityPicker {
	p.Visible = true
	p.incidentID = incidentID
	p.cursor = 0
	return p
}

// Init implements tea.Model.
func (p PriorityPicker) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (p PriorityPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.Visible {
		return p, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}

	switch key.String() {
	case "j", "down":
		if p.cursor < len(p.choices)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "enter":
		choice := p.choices[p.cursor]
		id := p.incidentID
		p.Visible = false
		return p, func() tea.Msg {
			return PrioritySelected{IncidentID: id, Priority: choice}
		}
	case "esc":
		p.Visible = false
		return p, func() tea.Msg { return PriorityPickerClosed{} }
	}

	return p, nil
}

// View implements tea.Model. Returns an empty string when not visible.
func (p PriorityPicker) View() tea.View {
	if !p.Visible {
		return tea.NewView("")
	}

	var sb strings.Builder
	sb.WriteString(theme.Title.Render("Set Priority"))
	sb.WriteString("\n\n")

	for i, choice := range p.choices {
		cursor := "  "
		if i == p.cursor {
			cursor = "> "
		}

		style, ok := theme.PriorityStyle(choice)
		if !ok {
			style = lipgloss.NewStyle()
		}

		fmt.Fprintf(&sb, "%s%s\n", cursor, style.Render(choice))
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpDesc.Render("↑↓ navigate  enter select  esc cancel"))

	return tea.NewView(RenderOverlay(sb.String(), 36))
}
