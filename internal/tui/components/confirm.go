package components

import (
	tea "charm.land/bubbletea/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// ConfirmResult is sent when the user accepts or cancels a confirmation modal.
// When Confirmed is true, OnYes carries the command to execute.
type ConfirmResult struct {
	Confirmed bool
	OnYes     tea.Cmd
}

// Confirm is a modal dialogue that asks the user to approve or cancel an action.
type Confirm struct {
	Visible bool
	title   string
	message string
	onYes   tea.Cmd
}

// Show returns a new Confirm with the given context and Visible set to true.
func (c Confirm) Show(title, message string, onYes tea.Cmd) Confirm {
	c.Visible = true
	c.title = title
	c.message = message
	c.onYes = onYes
	return c
}

// Init implements tea.Model.
func (c Confirm) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (c Confirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !c.Visible {
		return c, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return c, nil
	}

	switch key.String() {
	case "y", "enter":
		cmd := c.onYes
		c.Visible = false
		c.onYes = nil
		return c, func() tea.Msg {
			return ConfirmResult{Confirmed: true, OnYes: cmd}
		}
	case "n", "esc":
		c.Visible = false
		c.onYes = nil
		return c, func() tea.Msg {
			return ConfirmResult{Confirmed: false}
		}
	}

	return c, nil
}

// View implements tea.Model. Returns an empty string when not visible.
func (c Confirm) View() tea.View {
	if !c.Visible {
		return tea.NewView("")
	}

	title := theme.Title.Render(c.title)
	message := theme.HelpDesc.Render(c.message)
	keys := theme.HelpKey.Render("y") + theme.HelpDesc.Render(" confirm  ") +
		theme.HelpKey.Render("n") + theme.HelpDesc.Render(" cancel")

	content := title + "\n\n" + message + "\n\n" + keys

	return tea.NewView(RenderOverlay(content, 40))
}
