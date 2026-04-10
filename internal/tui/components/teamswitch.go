package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// TeamSelected is sent when the user confirms a team selection.
type TeamSelected struct {
	TeamID   string
	TeamName string
}

// TeamSwitcherClosed is sent when the user dismisses the team switcher without
// making a selection (esc).
type TeamSwitcherClosed struct{}

// TeamSwitcher is a Bubble Tea overlay component that lets the user pick a
// team from a list. Teams are fetched once and cached; subsequent openings
// reuse the cached list.
//
// TeamSwitcher does not depend on any package outside components. The parent
// provides a fetch command that returns a TeamsLoadedMsg.
type TeamSwitcher struct {
	Visible bool
	teams   []pagerduty.Team
	cursor  int
	loading bool
	err     error
}

// NewTeamSwitcher creates a TeamSwitcher. Teams are not fetched until Show is
// called or a teamsLoadedMsg is received.
func NewTeamSwitcher() TeamSwitcher {
	return TeamSwitcher{}
}

// TeamsLoadedMsg is the message carrying the fetched team list. It is
// constructed by the parent (App) from the fetch command result and handled
// by TeamSwitcher.Update.
type TeamsLoadedMsg struct {
	Teams []pagerduty.Team
	Err   error
}

// Show marks the switcher visible. If teams have already been loaded the cache
// is reused. fetchCmd is a tea.Cmd that fetches teams from the API; it is
// provided by the parent (App) so TeamSwitcher does not depend on internal/api.
func (ts TeamSwitcher) Show(fetchCmd tea.Cmd) (TeamSwitcher, tea.Cmd) {
	ts.Visible = true
	if len(ts.teams) > 0 || ts.loading {
		return ts, nil
	}
	ts.loading = true
	return ts, fetchCmd
}

// Init implements tea.Model.
func (ts TeamSwitcher) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (ts TeamSwitcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TeamsLoadedMsg:
		ts.loading = false
		ts.err = msg.Err
		if msg.Err == nil {
			ts.teams = msg.Teams
		}
		if ts.cursor >= len(ts.teams) {
			ts.cursor = 0
		}
		return ts, nil

	case tea.KeyPressMsg:
		if !ts.Visible {
			return ts, nil
		}
		switch msg.String() {
		case "esc":
			ts.Visible = false
			return ts, func() tea.Msg { return TeamSwitcherClosed{} }
		case "j", "down":
			if ts.cursor < len(ts.teams)-1 {
				ts.cursor++
			}
		case "k", "up":
			if ts.cursor > 0 {
				ts.cursor--
			}
		case "enter":
			if len(ts.teams) > 0 {
				t := ts.teams[ts.cursor]
				ts.Visible = false
				return ts, func() tea.Msg {
					return TeamSelected{TeamID: t.ID, TeamName: t.Name}
				}
			}
		}
	}
	return ts, nil
}

// View implements tea.Model. Returns an empty string when not visible.
func (ts TeamSwitcher) View() tea.View {
	if !ts.Visible {
		return tea.NewView("")
	}

	var sb strings.Builder
	sb.WriteString(theme.Title.Render("Switch Team"))
	sb.WriteString("\n\n")

	switch {
	case ts.loading:
		sb.WriteString("  Loading teams…\n")
	case ts.err != nil:
		sb.WriteString("  Error: " + ts.err.Error() + "\n")
	case len(ts.teams) == 0:
		sb.WriteString("  No teams found.\n")
	default:
		for i, t := range ts.teams {
			name := truncateStr(t.Name, 30)
			line := "  " + name
			if i == ts.cursor {
				line = PersistBg("> "+name, theme.CursorBg)
			}
			sb.WriteString(line + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpDesc.Render("  ↑↓ navigate  enter select  esc cancel"))

	return tea.NewView(RenderOverlay(sb.String(), 36))
}
