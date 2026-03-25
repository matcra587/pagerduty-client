package components

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// InputSubmitted is sent when the user presses Enter in the text input overlay.
type InputSubmitted struct {
	Action string // "snooze", "note", "reassign"
	Value  string
	ID     string // incident ID
}

// InputCancelled is sent when the user presses Esc in the text input overlay.
type InputCancelled struct{}

// TextInput is a modal text input overlay.
type TextInput struct {
	Visible    bool
	action     string
	incidentID string
	prompt     string
	input      textinput.Model
}

// NewTextInput creates an uninitialised TextInput overlay.
func NewTextInput() TextInput {
	ti := textinput.New()
	ti.CharLimit = 256
	return TextInput{input: ti}
}

// Show makes the overlay visible with the given action context and focuses
// the input field.
func (t TextInput) Show(action, incidentID, prompt, placeholder string) TextInput {
	t.Visible = true
	t.action = action
	t.incidentID = incidentID
	t.prompt = prompt
	t.input.Placeholder = placeholder
	t.input.SetValue("")
	t.input.Focus()
	return t
}

// Init implements tea.Model.
func (t TextInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (t TextInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !t.Visible {
		return t, nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			val := t.input.Value()
			action := t.action
			id := t.incidentID
			t.Visible = false
			t.input.Blur()
			return t, func() tea.Msg {
				return InputSubmitted{Action: action, Value: val, ID: id}
			}
		case "esc":
			t.Visible = false
			t.input.Blur()
			return t, func() tea.Msg { return InputCancelled{} }
		}
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return t, cmd
}

// View implements tea.Model. Returns an empty string when not visible.
func (t TextInput) View() tea.View {
	if !t.Visible {
		return tea.NewView("")
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(theme.ColorTitleFg).
		Bold(true)

	content := promptStyle.Render(t.prompt) + "\n\n" +
		t.input.View() + "\n\n" +
		theme.HelpDesc.Render("  enter confirm  esc cancel")

	return tea.NewView(RenderOverlay(content, 40))
}
