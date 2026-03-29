package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// EditDiff holds the fields changed by the edit overlay.
// Nil pointer = no change, non-nil = new value.
type EditDiff struct {
	Title    *string
	Urgency  *string
	Priority *string // priority ID; empty string = clear
}

// IsEmpty returns true if no fields were changed.
func (d EditDiff) IsEmpty() bool {
	return d.Title == nil && d.Urgency == nil && d.Priority == nil
}

// EditSubmittedMsg is sent when the user submits the edit overlay.
type EditSubmittedMsg struct {
	IncidentID string
	Diff       EditDiff
}

// EditCancelledMsg is sent when the user cancels the edit overlay.
type EditCancelledMsg struct{}

// editFields holds the editable field values for diffing.
type editFields struct {
	title    string
	urgency  string
	priority string // priority ID or "" for none
}

func (o editFields) diff(current editFields) EditDiff {
	var d EditDiff
	if current.title != o.title {
		d.Title = &current.title
	}
	if current.urgency != o.urgency {
		d.Urgency = &current.urgency
	}
	if current.priority != o.priority {
		d.Priority = &current.priority
	}
	return d
}

// EditOverlay is a huh.Form-based overlay for editing incident fields.
// current is a pointer so that huh.Form's Value() bindings survive
// the Bubble Tea value-copy cycle across Update calls.
type EditOverlay struct {
	Visible    bool
	form       *huh.Form
	incidentID string
	original   editFields
	current    *editFields
}

// Show creates and shows the edit overlay pre-filled from the incident.
// priorities is a list of pagerduty.Priority for the priority select.
func (o EditOverlay) Show(incident pagerduty.Incident, priorities []pagerduty.Priority) EditOverlay {
	currentPriority := ""
	if incident.Priority != nil {
		currentPriority = incident.Priority.ID
	}

	o.incidentID = incident.ID
	o.original = editFields{
		title:    incident.Title,
		urgency:  incident.Urgency,
		priority: currentPriority,
	}
	// Allocate on the heap so huh's Value() pointers survive copies.
	o.current = &editFields{
		title:    o.original.title,
		urgency:  o.original.urgency,
		priority: o.original.priority,
	}

	// Build priority options: None + each priority.
	priorityOpts := []huh.Option[string]{
		huh.NewOption[string]("None", ""),
	}
	for _, p := range priorities {
		priorityOpts = append(priorityOpts, huh.NewOption[string](p.Name, p.ID))
	}

	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("esc"))

	o.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&o.current.title),
			huh.NewSelect[string]().
				Title("Urgency").
				Options(
					huh.NewOption[string]("high", "high"),
					huh.NewOption[string]("low", "low"),
				).
				Value(&o.current.urgency),
			huh.NewSelect[string]().
				Title("Priority").
				Options(priorityOpts...).
				Value(&o.current.priority),
		),
	).WithTheme(huh.ThemeFunc(editTheme)).
		WithKeyMap(km).
		WithWidth(60).
		WithShowHelp(true)

	o.Visible = true
	return o
}

// Init implements tea.Model.
func (o EditOverlay) Init() tea.Cmd {
	if !o.Visible || o.form == nil {
		return nil
	}
	return o.form.Init()
}

// Update implements tea.Model.
func (o EditOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !o.Visible || o.form == nil {
		return o, nil
	}

	m, cmd := o.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		o.form = f
	}

	switch o.form.State {
	case huh.StateCompleted:
		o.Visible = false
		diff := o.original.diff(*o.current)
		return o, func() tea.Msg {
			return EditSubmittedMsg{
				IncidentID: o.incidentID,
				Diff:       diff,
			}
		}
	case huh.StateAborted:
		o.Visible = false
		return o, func() tea.Msg {
			return EditCancelledMsg{}
		}
	}

	return o, cmd
}

// View implements tea.Model.
func (o EditOverlay) View() tea.View {
	if !o.Visible || o.form == nil {
		return tea.NewView("")
	}

	content := o.form.View()
	return tea.NewView(RenderOverlay(content, 60))
}

// editTheme returns a huh theme styled to match the app's overlay look.
func editTheme(_ bool) *huh.Styles {
	t := huh.ThemeBase(true) // assume dark background

	accent := theme.Theme.Yellow.GetForeground()
	green := theme.Theme.Green.GetForeground()

	// Focused field: visible left border in accent colour.
	t.Focused.Base = t.Focused.Base.
		BorderForeground(accent)
	t.Focused.Title = t.Focused.Title.
		Foreground(theme.ColorTitleFg).
		Bold(true)
	t.Focused.SelectSelector = t.Focused.SelectSelector.
		Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.
		Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.
		Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.
		Foreground(green)
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Background(accent)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.
		Foreground(accent)

	// Blurred field: hidden border, dimmed title.
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.
		BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Title = t.Blurred.Title.
		Faint(true).
		Bold(false)
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title

	return t
}
