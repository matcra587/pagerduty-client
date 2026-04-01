// Package tui implements the interactive terminal UI for pdc.
package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/atotto/clipboard"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/ansi"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/integration"
	"github.com/matcra587/pagerduty-client/internal/tui/components"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
)

// view enumerates the top-level screens the App can display.
type view int

const (
	viewDashboard view = iota
	viewDetail
)

// topTab defines a top-level tab in the app.
type topTab struct {
	id    string // config key: "incidents", "escalation-policies", etc.
	label string // display label
}

// tabLabels maps config tab IDs to display labels.
var tabLabels = map[string]string{
	"incidents":           "Incidents",
	"escalation-policies": "Escalation Policies",
	"services":            "Services",
	"teams":               "Teams",
}

// buildTabs constructs the tab list from config.
// Falls back to all tabs if none configured.
func buildTabs(cfg *config.Config) []topTab {
	names := cfg.TUI.Tabs
	if len(names) == 0 {
		names = config.DefaultTabs
	}

	var tabs []topTab
	for _, name := range names {
		if label, ok := tabLabels[name]; ok {
			tabs = append(tabs, topTab{id: name, label: label})
		}
	}

	// Ensure incidents is always present as the first tab.
	if len(tabs) == 0 || tabs[0].id != "incidents" {
		tabs = append([]topTab{{id: "incidents", label: "Incidents"}}, tabs...)
	}

	return tabs
}

// activeTabID returns the config ID of the currently active tab.
func (a App) activeTabID() string {
	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		return a.tabs[a.activeTab].id
	}
	return "incidents"
}

// tickMsg is sent on each polling interval.
type tickMsg time.Time

// uiTickMsg triggers a UI-only re-render (e.g. to update the refresh age counter).
type uiTickMsg time.Time

// incidentsLoadedMsg carries freshly fetched incidents from the API.
type incidentsLoadedMsg struct {
	incidents []pagerduty.Incident
	err       error
}

// clearStatusMsg is sent after a delay to clear the status bar feedback.
type clearStatusMsg struct{ id int }

// prioritiesLoadedMsg carries fetched priorities for the edit overlay.
type prioritiesLoadedMsg struct {
	priorities []pagerduty.Priority
	incident   pagerduty.Incident
	err        error
}

// incidentActionPendingMsg is sent when an incident action starts, before
// the API call completes.
type incidentActionPendingMsg struct {
	op string
	id string
}

// App is the root Bubble Tea model. It owns the API client, config, view
// routing, polling ticker and the global help overlay.
type App struct {
	ctx            context.Context //nolint:containedctx // Bubble Tea models are value-typed; context must travel with the model.
	cancel         context.CancelFunc
	client         *api.Client
	cfg            *config.Config
	ansi           *ansi.ANSI
	fromEmail      string
	current        view
	dashboard      Dashboard
	detail         incidentDetail
	ep             escalationPolicies
	epCache        map[string]pagerduty.EscalationPolicy
	svc            services
	svcCache       map[string]pagerduty.Service
	tmv            teamsView
	statusBar      components.StatusBar
	help           components.Help
	confirm        components.Confirm
	teamSwitch     components.TeamSwitcher
	filterOpts     components.FilterOptions
	textInput      components.TextInput
	priorityPicker components.PriorityPicker
	editOverlay    components.EditOverlay
	priorities     []pagerduty.Priority
	spinner        spinner.Model
	loading        bool
	width          int
	height         int
	filterState    components.FilterState
	paused         bool
	interval       time.Duration
	statusID       int
	tabs           []topTab
	activeTab      int
	bodyH          int
	keys           appKeyMap
}

// New constructs an App wired to the given client, config and acting user email.
func New(ctx context.Context, client *api.Client, cfg *config.Config, fromEmail string) App {
	ctx, cancel := context.WithCancel(ctx)

	interval := time.Duration(cfg.RefreshInterval) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	preset, ok := theme.Presets[cfg.TUI.Theme]
	if !ok {
		preset = theme.Presets["dark"]
	}
	theme.Apply(preset())

	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	sp.Style = lipgloss.NewStyle().Foreground(theme.ColorHeaderFg)

	filterOpts := components.NewFilterOptions()
	defaultState := components.DefaultFilterState()

	a := App{
		ctx:            ctx,
		cancel:         cancel,
		client:         client,
		cfg:            cfg,
		ansi:           ansi.Force(),
		fromEmail:      fromEmail,
		current:        viewDashboard,
		dashboard:      newDashboard(ctx, client, fromEmail, cfg.Service != ""),
		ep:             newEscalationPolicies(),
		epCache:        make(map[string]pagerduty.EscalationPolicy),
		svc:            newServices(),
		svcCache:       make(map[string]pagerduty.Service),
		tmv:            newTeamsView(),
		statusBar:      components.StatusBar{Team: cfg.Team, FilterState: defaultState},
		teamSwitch:     components.NewTeamSwitcher(),
		filterOpts:     filterOpts,
		filterState:    defaultState,
		textInput:      components.NewTextInput(),
		priorityPicker: components.NewPriorityPicker(),
		spinner:        sp,
		loading:        true,
		interval:       interval,
		tabs:           buildTabs(cfg),
		activeTab:      0,
		keys:           newAppKeyMap(),
	}
	a.statusBar.LastRefresh = time.Now()
	return a
}

// Init starts the polling ticker and triggers the first data fetch.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.dashboard.Init(),
		tickCmd(a.interval),
		uiTickCmd(),
		a.fetchIncidentsCmd(),
		a.spinner.Tick,
	)
}

// Update handles all messages, routing key presses to updateKeyPress
// and dispatching other message types inline.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.statusBar.Width = msg.Width

		// Compute body height: total minus header and footer chrome.
		const headerH = 2 // tab bar + border-bottom separator
		const footerH = 2 // labelled border + hint line
		a.bodyH = max(a.height-headerH-footerH, 1)

		// Forward the body-scoped size to children.
		childSize := tea.WindowSizeMsg{Width: msg.Width, Height: a.bodyH}
		dm, _ := a.dashboard.Update(childSize)
		a.dashboard = dm.(Dashboard)
		em, _ := a.ep.Update(childSize)
		a.ep = em.(escalationPolicies)
		sm, _ := a.svc.Update(childSize)
		a.svc = sm.(services)
		tm, _ := a.tmv.Update(childSize)
		a.tmv = tm.(teamsView)
		if a.current == viewDetail {
			dm2, _ := a.detail.Update(childSize)
			a.detail = dm2.(incidentDetail)
		}
		return a, nil

	case tea.KeyPressMsg:
		return a.updateKeyPress(msg)

	case spinner.TickMsg:
		if a.loading {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}
		return a, nil

	case uiTickMsg:
		// Re-render only (updates the refresh age counter). No API call.
		return a, uiTickCmd()

	case tickMsg:
		if !a.paused {
			// Background poll: no spinner overlay, just fetch silently.
			a.statusBar.LastRefresh = time.Time(msg)
			return a, tea.Batch(
				tickCmd(a.interval),
				a.fetchIncidentsCmd(),
			)
		}
		return a, nil

	case incidentsLoadedMsg:
		a.loading = false
		if msg.err != nil {
			a, cmd := a.flashResult(fmt.Sprintf("Fetch failed: %v", msg.err), true)
			return a, cmd
		}
		a.dashboard.SetIncidents(msg.incidents)
		a.updateStatusBarCounts(msg.incidents)
		return a, nil

	case epLoadedMsg:
		em, cmd := a.ep.Update(msg)
		a.ep = em.(escalationPolicies)
		// Populate cache.
		a.epCache = make(map[string]pagerduty.EscalationPolicy, len(msg.policies))
		for _, p := range msg.policies {
			a.epCache[p.ID] = p
		}
		return a, cmd

	case servicesLoadedMsg:
		sm, cmd := a.svc.Update(msg)
		a.svc = sm.(services)
		// Populate cache.
		a.svcCache = make(map[string]pagerduty.Service, len(msg.services))
		for _, sv := range msg.services {
			a.svcCache[sv.ID] = sv
		}
		return a, cmd

	case teamTabLoadedMsg:
		tm, cmd := a.tmv.Update(msg)
		a.tmv = tm.(teamsView)
		return a, cmd

	case incidentActionPendingMsg:
		a = a.setStatusPending(msg.op, msg.id)
		return a, nil

	case clearStatusMsg:
		if msg.id == a.statusID {
			a.statusBar.StatusMsg = ""
		}
		return a, nil

	case components.TeamSelected:
		a.cfg.Team = msg.TeamID
		a.statusBar.Team = msg.TeamName
		a.dashboard.incidents.ClearFilter()
		a.loading = true
		return a, tea.Batch(
			a.fetchIncidentsCmd(),
			a.spinner.Tick,
		)

	case components.TeamSwitcherClosed:
		return a, nil

	case components.FilterAppliedMsg:
		switch msg.Origin {
		case "incidents":
			a.filterState = a.filterOpts.State()
			a.dashboard.incidents.filterState = a.filterState
			a.statusBar.FilterState = a.filterState
			a.loading = true
			return a, tea.Batch(a.fetchIncidentsCmd(), a.spinner.Tick)
		case "services":
			status := msg.Selections["Status"]
			if status == "" {
				status = "all"
			}
			a.svc.statusFilter = status
			a.svc.cursor = 0
			a.svc.scrollOffset = 0
			return a, nil
		}
		return a, nil

	case components.FilterClosed:
		return a, nil

	case IncidentSelected:
		a.detail = newIncidentDetail(a.ctx, a.client, a.cfg, a.ansi, msg.Incident)
		a.detail.width = a.width
		a.detail.height = max(a.bodyH, 1)
		a.detail.syncContent()
		a.current = viewDetail
		return a, a.detail.Init()

	case IncidentAcked:
		return a.reloadAfterAction("Acknowledged " + msg.ID)

	case IncidentResolved:
		return a.reloadAfterAction("Resolved " + msg.ID)

	case IncidentSnoozed:
		return a.reloadAfterAction("Snoozed " + msg.ID)

	case IncidentNoteAdded:
		return a.reloadAfterAction("Note added to " + msg.ID)

	case IncidentReassigned:
		return a.reloadAfterAction("Reassigned " + msg.ID)

	case statusMsg:
		a, cmd := a.flashResult(string(msg), false)
		return a, cmd

	case incidentErrMsg:
		a, cmd := a.flashResult(msg.op+": "+msg.err.Error(), true)
		return a, cmd

	case showInputMsg:
		a.textInput = a.textInput.Show(msg.action, msg.incidentID, msg.prompt, msg.placeholder)
		return a, a.textInput.Init()

	case components.InputSubmitted:
		if msg.Action == "note" && strings.TrimSpace(msg.Value) == "" {
			a, cmd := a.flashResult("Note cannot be empty", true)
			return a, cmd
		}
		return a, a.executeInputAction(msg)

	case components.InputCancelled:
		return a, nil

	case components.ConfirmResult:
		if msg.Confirmed && msg.OnYes != nil {
			return a, msg.OnYes
		}
		return a, nil

	case IncidentEscalated:
		return a.reloadAfterAction("Escalated " + msg.ID)

	case detailAckMsg:
		return a, a.detailAckCmd(msg.id)

	case detailNoteMsg:
		return a, func() tea.Msg {
			return showInputMsg{
				action:     "note",
				incidentID: msg.id,
				prompt:     "Add note:",
			}
		}

	case detailSnoozeMsg:
		return a, func() tea.Msg {
			return showInputMsg{
				action:      "snooze",
				incidentID:  msg.id,
				prompt:      "Snooze duration (e.g. 4h, 30m):",
				placeholder: "4h",
			}
		}

	case detailEditMsg:
		if len(a.priorities) > 0 {
			a.editOverlay = a.editOverlay.Show(msg.incident, a.priorities)
			return a, a.editOverlay.Init()
		}
		// Lazy-fetch priorities, then open overlay.
		inc := msg.incident
		client := a.client
		appCtx := a.ctx
		return a, func() tea.Msg {
			ctx, cancel := context.WithTimeout(appCtx, 5*time.Second)
			defer cancel()
			priorities, err := client.ListPriorities(ctx)
			return prioritiesLoadedMsg{priorities: priorities, incident: inc, err: err}
		}

	case prioritiesLoadedMsg:
		if msg.err != nil {
			// Open overlay without priorities if fetch fails.
			a.editOverlay = a.editOverlay.Show(msg.incident, nil)
		} else {
			a.priorities = msg.priorities
			a.editOverlay = a.editOverlay.Show(msg.incident, msg.priorities)
		}
		return a, a.editOverlay.Init()

	case components.EditSubmittedMsg:
		if msg.Diff.IsEmpty() {
			a, cmd := a.flashResult("No changes", false)
			return a, cmd
		}
		return a, a.updateIncidentCmd(msg.IncidentID, msg.Diff)

	case components.EditCancelledMsg:
		return a, nil

	case detailEscalateMsg:
		escalateCmd := a.escalateCmd(msg.id)
		if msg.confirm {
			a.confirm = a.confirm.Show(
				"Escalate incident",
				fmt.Sprintf("Escalate %s?", msg.id),
				escalateCmd,
			)
			return a, nil
		}
		return a, escalateCmd

	case detailSetPriorityMsg:
		a.priorityPicker = a.priorityPicker.Show(msg.id)
		return a, nil

	case components.PrioritySelected:
		return a, a.updatePriorityCmd(msg.IncidentID, msg.Priority)

	case components.PriorityPickerClosed:
		return a, nil

	case IncidentPriorityUpdated:
		return a.reloadAfterAction("Priority updated for " + msg.ID)

	case IncidentUpdated:
		return a.reloadAfterAction("Updated " + msg.ID)

	case IncidentMerged:
		return a.reloadAfterAction("Merged into " + msg.TargetID)

	case detailResolveMsg:
		if msg.confirm {
			return a, func() tea.Msg {
				return showInputMsg{
					action:      "resolve",
					incidentID:  msg.id,
					prompt:      fmt.Sprintf("Resolve %s - add a note (enter to skip):", msg.id),
					placeholder: "",
				}
			}
		}
		return a, a.detailResolveCmd(msg.id)

	case batchResultMsg:
		a.dashboard.incidents.selections = make(map[string]bool)
		var text string
		if msg.failures == 0 {
			text = fmt.Sprintf("%sed %d incidents", capitalise(msg.op), msg.success)
		} else {
			text = fmt.Sprintf("%sed %d/%d (%d failed)", capitalise(msg.op), msg.success, msg.success+msg.failures, msg.failures)
			if msg.firstErr != nil {
				text += fmt.Sprintf(" (%v)", msg.firstErr)
			}
		}
		a.loading = true
		a, flashCmd := a.flashResult(text, msg.failures > 0)
		return a, tea.Batch(
			flashCmd,
			a.fetchIncidentsCmd(),
			a.spinner.Tick,
		)

	case browserOpenedMsg:
		if msg.err != nil {
			a, cmd := a.flashResult("Failed to open browser: "+msg.err.Error(), true)
			return a, cmd
		}
		return a, nil
	}

	// Forward non-key messages to the edit overlay when visible.
	// huh.Form sends internal messages (nextFieldMsg, nextGroupMsg, etc.)
	// via commands that must reach the form's Update to function.
	if a.editOverlay.Visible {
		em, cmd := a.editOverlay.Update(msg)
		a.editOverlay = em.(components.EditOverlay)
		return a, cmd
	}

	// Non-key messages for the active view.
	switch a.current {
	case viewDashboard:
		if a.activeTabID() == "escalation-policies" {
			em, cmd := a.ep.Update(msg)
			a.ep = em.(escalationPolicies)
			return a, cmd
		}
		if a.activeTabID() == "services" {
			sm, cmd := a.svc.Update(msg)
			a.svc = sm.(services)
			return a, cmd
		}
		if a.activeTabID() == "teams" {
			tm, cmd := a.tmv.Update(msg)
			a.tmv = tm.(teamsView)
			return a, cmd
		}
		dm, cmd := a.dashboard.Update(msg)
		a.dashboard = dm.(Dashboard)
		return a, cmd
	case viewDetail:
		dm, cmd := a.detail.Update(msg)
		a.detail = dm.(incidentDetail)
		return a, cmd
	}

	return a, nil
}

// updateKeyPress handles all key press messages, routing through
// overlays first, then global keys, then view-specific handlers.
func (a App) updateKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Overlay routing cascade - active overlay consumes the key.
	if a.editOverlay.Visible {
		em, cmd := a.editOverlay.Update(msg)
		a.editOverlay = em.(components.EditOverlay)
		return a, cmd
	}
	if a.textInput.Visible {
		tm, cmd := a.textInput.Update(msg)
		a.textInput = tm.(components.TextInput)
		return a, cmd
	}
	if a.confirm.Visible {
		cm, cmd := a.confirm.Update(msg)
		a.confirm = cm.(components.Confirm)
		return a, cmd
	}
	if a.teamSwitch.Visible {
		tm, cmd := a.teamSwitch.Update(msg)
		a.teamSwitch = tm.(components.TeamSwitcher)
		return a, cmd
	}
	if a.priorityPicker.Visible {
		pm, cmd := a.priorityPicker.Update(msg)
		a.priorityPicker = pm.(components.PriorityPicker)
		return a, cmd
	}
	if a.filterOpts.Visible {
		fm, cmd := a.filterOpts.Update(msg)
		a.filterOpts = fm.(components.FilterOptions)
		return a, cmd
	}
	if a.help.Visible {
		hm, cmd := a.help.Update(msg)
		a.help = hm.(components.Help)
		return a, cmd
	}

	// Filter input is active - let the dashboard consume the key.
	if a.current == viewDashboard && a.dashboard.FilterActive() {
		dm, cmd := a.dashboard.Update(msg)
		a.dashboard = dm.(Dashboard)
		return a, cmd
	}

	// Esc in detail view returns to dashboard.
	if key.Matches(msg, a.keys.Back) && a.current == viewDetail {
		a.current = viewDashboard
		return a, nil
	}

	// Top-level tab switching (dashboard view only).
	if a.current == viewDashboard {
		// Number keys are dynamic (tab count varies), so string matching
		// is used here rather than key.Binding.
		if idx := tabIndexFromKey(msg.String()); idx >= 0 && idx < len(a.tabs) {
			a.activeTab = idx
			if cmd := a.lazyLoadTab(idx); cmd != nil {
				return a, cmd
			}
			return a, nil
		}
		switch {
		case key.Matches(msg, a.keys.Tab):
			a.activeTab = (a.activeTab + 1) % len(a.tabs)
			if cmd := a.lazyLoadTab(a.activeTab); cmd != nil {
				return a, cmd
			}
			return a, nil
		case key.Matches(msg, a.keys.ShiftTab):
			a.activeTab = (a.activeTab - 1 + len(a.tabs)) % len(a.tabs)
			if cmd := a.lazyLoadTab(a.activeTab); cmd != nil {
				return a, cmd
			}
			return a, nil
		}
	}

	km := a.keys

	// Global keys.
	switch {
	case key.Matches(msg, km.Quit):
		a.cancel()
		return a, tea.Quit
	case key.Matches(msg, km.Help):
		a.help.Visible = true
		switch {
		case a.current == viewDetail:
			a.help.CurrentView = "detail"
		case a.activeTabID() == "escalation-policies":
			a.help.CurrentView = "escalation-policies"
		case a.activeTabID() == "services":
			a.help.CurrentView = "services"
		case a.activeTabID() == "teams":
			a.help.CurrentView = "teams"
		default:
			a.help.CurrentView = "dashboard"
		}
		return a, nil
	case key.Matches(msg, km.TogglePause):
		a.paused = !a.paused
		a.statusBar.Paused = a.paused
		if !a.paused {
			return a, tickCmd(a.interval)
		}
		return a, nil
	case key.Matches(msg, km.FilterOpts):
		switch a.activeTabID() {
		case "incidents":
			a.filterOpts = a.filterOpts.ShowWithRows("incidents", components.IncidentFilterRows())
		case "services":
			a.filterOpts = a.filterOpts.ShowWithRows("services", components.ServiceFilterRows())
		}
		return a, nil
	case key.Matches(msg, km.TeamSwitch):
		ts, cmd := a.teamSwitch.Show(a.fetchTeamsCmd())
		a.teamSwitch = ts
		return a, cmd
	case key.Matches(msg, km.CopyURL):
		var url string
		switch a.current {
		case viewDashboard:
			vis := a.dashboard.incidents.visibleIncidents()
			if len(vis) > 0 {
				url = vis[a.dashboard.incidents.cursor].HTMLURL
			}
		case viewDetail:
			url = a.detail.incident.HTMLURL
		}
		if url != "" {
			if err := clipboard.WriteAll(url); err != nil {
				a, cmd := a.flashResult("Failed to copy URL", true)
				return a, cmd
			}
			a, cmd := a.flashResult("Copied URL", false)
			return a, cmd
		}
		return a, nil
	case key.Matches(msg, km.Open):
		if a.current == viewDashboard {
			vis := a.dashboard.incidents.visibleIncidents()
			if len(vis) > 0 {
				inc := vis[a.dashboard.incidents.cursor]
				url := inc.HTMLURL
				if url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
					return a, openBrowser(a.ctx, url)
				}
			}
			return a, nil
		}
		if a.current == viewDetail {
			url := a.detail.incident.HTMLURL
			if url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
				return a, openBrowser(a.ctx, url)
			}
			return a, nil
		}
	case key.Matches(msg, km.OpenExt):
		if a.current == viewDashboard {
			a, cmd := a.flashResult("Open detail view first", true)
			return a, cmd
		}
		url := a.resolveExternalLink()
		if url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			return a, openBrowser(a.ctx, url)
		}
		a, cmd := a.flashResult("No external link configured", true)
		return a, cmd
	}

	// Dashboard-only action keys (incidents tab only).
	if a.current == viewDashboard && a.activeTabID() == "incidents" {
		return a.updateDashboardKey(msg)
	}

	// EP tab keys.
	if a.current == viewDashboard && a.activeTabID() == "escalation-policies" {
		if msg.String() == "R" {
			a.ep.loading = true
			return a, a.fetchEPCmd()
		}
		em, cmd := a.ep.Update(msg)
		a.ep = em.(escalationPolicies)
		return a, cmd
	}

	// Services tab keys.
	if a.current == viewDashboard && a.activeTabID() == "services" {
		if msg.String() == "R" {
			a.svc.loading = true
			return a, a.fetchServicesCmd()
		}
		if msg.String() == "f" {
			a.filterOpts = a.filterOpts.ShowWithRows("services", components.ServiceFilterRows())
			return a, nil
		}
		sm, cmd := a.svc.Update(msg)
		a.svc = sm.(services)
		return a, cmd
	}

	// Teams tab keys.
	if a.current == viewDashboard && a.activeTabID() == "teams" {
		if msg.String() == "R" {
			a.tmv.loading = true
			return a, a.fetchTeamsTabCmd()
		}
		tm, cmd := a.tmv.Update(msg)
		a.tmv = tm.(teamsView)
		return a, cmd
	}

	// Forward unhandled keys to the active view.
	if a.current == viewDetail {
		dm, cmd := a.detail.Update(msg)
		a.detail = dm.(incidentDetail)
		return a, cmd
	}

	return a, nil
}

// updateDashboardKey handles dashboard-specific action keys.
func (a App) updateDashboardKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := a.keys

	switch {
	case key.Matches(msg, km.Ack):
		if len(a.dashboard.incidents.selections) > 0 {
			n := len(a.dashboard.incidents.selections)
			a.confirm = a.confirm.Show(
				"Acknowledge incidents",
				fmt.Sprintf("Acknowledge %d selected incidents?", n),
				a.dashboard.incidents.batchAckCmd(),
			)
			return a, nil
		}
		dm, cmd := a.dashboard.Update(msg)
		a.dashboard = dm.(Dashboard)
		return a, cmd

	case key.Matches(msg, km.Resolve):
		if len(a.dashboard.incidents.selections) > 0 {
			n := len(a.dashboard.incidents.selections)
			a.confirm = a.confirm.Show(
				"Resolve incidents",
				fmt.Sprintf("Resolve %d selected incidents?", n),
				a.dashboard.incidents.batchResolveCmd(),
			)
			return a, nil
		}
		vis := a.dashboard.incidents.visibleIncidents()
		if len(vis) > 0 {
			inc := vis[a.dashboard.incidents.cursor]
			return a, func() tea.Msg {
				return showInputMsg{
					action:      "resolve",
					incidentID:  inc.ID,
					prompt:      fmt.Sprintf("Resolve %s - add a note (enter to skip):", inc.ID),
					placeholder: "",
				}
			}
		}
		return a, nil

	case key.Matches(msg, km.ResolveNow):
		if len(a.dashboard.incidents.selections) > 0 {
			return a, a.dashboard.incidents.batchResolveCmd()
		}
		return a, a.dashboard.incidents.resolveCmd()

	case key.Matches(msg, km.Edit):
		vis := a.dashboard.incidents.visibleIncidents()
		if len(vis) > 0 {
			inc := vis[a.dashboard.incidents.cursor]
			return a, func() tea.Msg {
				return detailEditMsg{incident: inc}
			}
		}
		return a, nil

	case key.Matches(msg, km.Escalate):
		vis := a.dashboard.incidents.visibleIncidents()
		if len(vis) > 0 {
			inc := vis[a.dashboard.incidents.cursor]
			a.confirm = a.confirm.Show(
				"Escalate incident",
				fmt.Sprintf("Escalate %s?", inc.ID),
				a.escalateCmd(inc.ID),
			)
		}
		return a, nil

	case key.Matches(msg, km.EscalateNow):
		vis := a.dashboard.incidents.visibleIncidents()
		if len(vis) > 0 {
			inc := vis[a.dashboard.incidents.cursor]
			return a, a.escalateCmd(inc.ID)
		}
		return a, nil

	case key.Matches(msg, km.Merge):
		selected := a.dashboard.incidents.selectedIncidents()
		if len(selected) >= 2 {
			a.confirm = a.confirm.Show(
				"Merge incidents",
				fmt.Sprintf("Merge %d incidents?", len(selected)),
				a.mergeCmd(selected),
			)
		}
		return a, nil

	case key.Matches(msg, km.MergeNow):
		selected := a.dashboard.incidents.selectedIncidents()
		if len(selected) >= 2 {
			return a, a.mergeCmd(selected)
		}
		return a, nil
	}

	// Unhandled key - forward to dashboard sub-model.
	dm, cmd := a.dashboard.Update(msg)
	a.dashboard = dm.(Dashboard)
	return a, cmd
}

// View composes three zones - header (tab bar), body (content) and footer
// (status bar) - and layers any active overlay on top.
func (a App) View() tea.View {
	if a.width == 0 {
		return tea.NewView("")
	}

	// --- header ---
	header := a.headerView()
	headerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.ColorOverlayBorder).
		Width(a.width)
	header = headerBorder.Render(header)

	// --- footer (set hint context first) ---
	var viewName string
	switch {
	case a.current == viewDetail:
		viewName = "detail"
	case a.activeTabID() == "escalation-policies":
		viewName = "escalation-policies"
	case a.activeTabID() == "services":
		viewName = "services"
	case a.activeTabID() == "teams":
		viewName = "teams"
	default:
		viewName = "dashboard"
	}
	a.statusBar.Hint = components.HintContext{
		View:          viewName,
		HasSelections: len(a.dashboard.incidents.selections) > 0,
		FilterActive:  a.current == viewDashboard && a.dashboard.FilterActive(),
		Paused:        a.paused,
	}
	footer := a.statusBar.View().Content

	// --- body (use pre-computed bodyH from Update, single source of truth) ---
	var bodyContent string
	switch a.current {
	case viewDashboard:
		switch a.activeTabID() {
		case "incidents":
			bodyContent = a.dashboard.View().Content
		case "escalation-policies":
			bodyContent = a.ep.View().Content
		case "services":
			bodyContent = a.svc.View().Content
		case "teams":
			bodyContent = a.tmv.View().Content
		default:
			bodyContent = lipgloss.Place(a.width, a.bodyH, lipgloss.Center, lipgloss.Center,
				lipgloss.NewStyle().Faint(true).Render("Not yet supported"))
		}
	case viewDetail:
		bodyContent = a.detail.View().Content
	default:
		bodyContent = a.dashboard.View().Content
	}
	// The detail view's View() outputs exactly bodyH lines (padded with
	// newlines), so we skip the expensive lipgloss render for that path.
	// Other views still need it for dimension enforcement.
	var body string
	if a.current == viewDetail {
		body = bodyContent
	} else {
		body = lipgloss.NewStyle().Width(a.width).Height(a.bodyH).MaxHeight(a.bodyH).Render(bodyContent)
	}

	// Dim the body and show a centred spinner while loading.
	if a.loading {
		body = lipgloss.NewStyle().Faint(true).Render(body)
		spinnerOverlay := components.RenderOverlay(a.spinner.View()+"  Loading...", 0)
		body = overlayCenter(body, spinnerOverlay, a.width, a.bodyH)
	}

	// Layer overlays on the body zone.
	if a.editOverlay.Visible {
		body = overlayCenter(body, a.editOverlay.View().Content, a.width, a.bodyH)
	} else if a.textInput.Visible {
		body = overlayCenter(body, a.textInput.View().Content, a.width, a.bodyH)
	} else if a.confirm.Visible {
		body = overlayCenter(body, a.confirm.View().Content, a.width, a.bodyH)
	} else if a.priorityPicker.Visible {
		body = overlayCenter(body, a.priorityPicker.View().Content, a.width, a.bodyH)
	} else if a.teamSwitch.Visible {
		body = overlayCenter(body, a.teamSwitch.View().Content, a.width, a.bodyH)
	} else if a.filterOpts.Visible {
		body = overlayCenter(body, a.filterOpts.View().Content, a.width, a.bodyH)
	} else if a.help.Visible {
		body = overlayCenter(body, a.help.View().Content, a.width, a.bodyH)
	}

	// Compose zones vertically.
	base := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	v := tea.NewView(base)
	v.AltScreen = true
	return v
}

// headerView returns context-specific header content for the current view.
func (a App) headerView() string {
	switch a.current {
	case viewDashboard:
		return a.topTabBar()
	case viewDetail:
		return a.detail.headerContent()
	default:
		return ""
	}
}

// topTabBar renders the top-level tab bar with count pills on the right.
func (a App) topTabBar() string {
	active := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorTitleFg).
		Underline(true).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(theme.ColorHeaderFg).
		Faint(true).
		Padding(0, 1)

	var parts []string
	for i, t := range a.tabs {
		label := t.label
		if i == a.activeTab {
			parts = append(parts, active.Render(label))
		} else {
			parts = append(parts, inactive.Render(label))
		}
	}
	tabs := strings.Join(parts, " ")

	// Count pills on the right.
	pills := a.countPills()
	pillsW := lipgloss.Width(pills)
	tabsW := lipgloss.Width(tabs)

	gap := max(a.width-tabsW-pillsW, 1)
	return tabs + fmt.Sprintf("%*s", gap, "") + pills
}

// countPills renders incident count pills for the header bar.
func (a App) countPills() string {
	pill := func(label string, count int, active, dim lipgloss.Style) string {
		s := fmt.Sprintf("%s %d", label, count)
		if count > 0 {
			return active.Render(s)
		}
		return dim.Render(s)
	}

	t := pill("triggered", a.statusBar.Triggered, theme.PillDanger, theme.PillDim)
	ac := pill("acked", a.statusBar.Acknowledged, theme.PillWarning, theme.PillDim)
	r := pill("resolved", a.statusBar.Resolved, theme.PillDim, theme.PillDim)

	return t + ac + r
}

// tabIndexFromKey returns the zero-based tab index for number keys "1"-"9",
// or -1 if the key is not a tab switch key.
func tabIndexFromKey(k string) int {
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		return int(k[0] - '1')
	}
	return -1
}

// fetchIncidentsCmd returns a tea.Cmd that loads incidents from the API.
func (a App) fetchIncidentsCmd() tea.Cmd {
	client := a.client
	appCtx := a.ctx
	opts := api.ListIncidentsOpts{}

	switch a.filterState.Status {
	case "open":
		opts.Statuses = []string{"triggered", "acknowledged"}
	case "triggered":
		opts.Statuses = []string{"triggered"}
	case "acked":
		opts.Statuses = []string{"acknowledged"}
	case "resolved":
		opts.Statuses = []string{"resolved"}
	default:
		opts.Statuses = []string{"triggered", "acknowledged", "resolved"}
	}

	switch a.filterState.Urgency {
	case "high":
		opts.Urgencies = []string{"high"}
	case "low":
		opts.Urgencies = []string{"low"}
	default:
	}

	since, until := ageRange(a.filterState.Age)
	opts.Since = since
	opts.Until = until
	if a.filterState.Age == "all" {
		opts.DateRange = "all"
	}

	opts.SortBy = "created_at:desc"

	if a.cfg.Team != "" {
		opts.TeamIDs = []string{a.cfg.Team}
	}
	if a.cfg.Service != "" {
		opts.ServiceIDs = []string{a.cfg.Service}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 30*time.Second)
		defer cancel()
		incidents, err := client.ListIncidents(ctx, opts)
		return incidentsLoadedMsg{incidents: incidents, err: err}
	}
}

// fetchTeamsCmd returns a tea.Cmd that lists teams from the API, producing
// a components.TeamsLoadedMsg on completion.
func (a App) fetchTeamsCmd() tea.Cmd {
	client := a.client
	appCtx := a.ctx
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 30*time.Second)
		defer cancel()
		teams, err := client.ListTeams(ctx, api.ListTeamsOpts{})
		return components.TeamsLoadedMsg{Teams: teams, Err: err}
	}
}

// lazyLoadTab triggers the first fetch for a tab that loads on demand.
func (a *App) lazyLoadTab(idx int) tea.Cmd {
	if idx < 0 || idx >= len(a.tabs) {
		return nil
	}
	switch a.tabs[idx].id {
	case "escalation-policies":
		if !a.ep.loaded && !a.ep.loading {
			a.ep.loading = true
			return a.fetchEPCmd()
		}
	case "services":
		if !a.svc.loaded && !a.svc.loading {
			a.svc.loading = true
			return a.fetchServicesCmd()
		}
	case "teams":
		if !a.tmv.loaded && !a.tmv.loading {
			a.tmv.loading = true
			return a.fetchTeamsTabCmd()
		}
	}
	return nil
}

// fetchEPCmd returns a tea.Cmd that loads escalation policies from the API.
func (a App) fetchEPCmd() tea.Cmd {
	client := a.client
	appCtx := a.ctx
	opts := api.ListEscalationPoliciesOpts{}
	if a.cfg.Team != "" {
		opts.TeamIDs = []string{a.cfg.Team}
	}
	return func() tea.Msg {
		policies, err := client.ListEscalationPolicies(appCtx, opts)
		return epLoadedMsg{policies: policies, err: err}
	}
}

// fetchServicesCmd returns a tea.Cmd that loads services from the API.
func (a App) fetchServicesCmd() tea.Cmd {
	client := a.client
	appCtx := a.ctx
	opts := api.ListServicesOpts{}
	if a.cfg.Team != "" {
		opts.TeamIDs = []string{a.cfg.Team}
	}
	return func() tea.Msg {
		svcs, err := client.ListServices(appCtx, opts)
		return servicesLoadedMsg{services: svcs, err: err}
	}
}

// fetchTeamsTabCmd returns a tea.Cmd that loads teams from the API.
func (a App) fetchTeamsTabCmd() tea.Cmd {
	client := a.client
	appCtx := a.ctx
	return func() tea.Msg {
		teams, err := client.ListTeams(appCtx, api.ListTeamsOpts{})
		return teamTabLoadedMsg{teams: teams, err: err}
	}
}

// updateStatusBarCounts recalculates the incident counts displayed in the
// status bar from the current incident list.
func (a *App) updateStatusBarCounts(incidents []pagerduty.Incident) {
	var triggered, acked, resolved int
	for _, inc := range incidents {
		switch inc.Status {
		case "triggered":
			triggered++
		case "acknowledged":
			acked++
		case "resolved":
			resolved++
		}
	}
	a.statusBar.Triggered = triggered
	a.statusBar.Acknowledged = acked
	a.statusBar.Resolved = resolved
	a.statusBar.FilterState = a.filterState
}

// tickCmd returns a tea.Cmd that fires a tickMsg after d.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// uiTickCmd fires a UI-only re-render every 10 seconds.
func uiTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return uiTickMsg(t)
	})
}

// overlayCenter renders the overlay string centred over base.
func overlayCenter(base, overlay string, w, h int) string {
	ow := lipgloss.Width(overlay)
	oh := lipgloss.Height(overlay)

	x := (w - ow) / 2
	y := (h - oh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	lines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, ol := range overlayLines {
		row := y + i
		if row >= len(lines) {
			break
		}
		line := lines[row]
		for lipgloss.Width(line) < x {
			line += " "
		}
		prefix := xansi.Truncate(line, x, "")
		lines[row] = prefix + ol
	}

	var sb strings.Builder
	for i, l := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(l)
	}
	return sb.String()
}

func (a App) readOnly() bool {
	return a.client == nil
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (a App) setStatusPending(op, id string) App {
	var verb string
	switch op {
	case "resolve":
		verb = "Resolving"
	case "escalate":
		verb = "Escalating"
	case "merge":
		verb = "Merging"
	case "snooze":
		verb = "Snoozing"
	default:
		verb = capitalise(op) + "ing"
	}
	a.statusBar.StatusMsg = theme.StatusOK.Render(verb + " " + id + "...")
	return a
}

// reloadAfterAction sets loading state and returns a batch of commands to
// flash a status message, refresh the incident list and tick the spinner.
func (a App) reloadAfterAction(text string) (tea.Model, tea.Cmd) {
	a.loading = true
	a, flashCmd := a.flashResult(text, false)
	return a, tea.Batch(
		flashCmd,
		a.fetchIncidentsCmd(),
		a.spinner.Tick,
	)
}

func (a App) flashResult(msg string, isErr bool) (App, tea.Cmd) {
	a.statusID++
	id := a.statusID
	if isErr {
		a.statusBar.StatusMsg = theme.StatusErr.Render("✗ " + msg)
	} else {
		a.statusBar.StatusMsg = theme.StatusOK.Render("✓ " + msg)
	}
	cmd := tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{id: id}
	})
	return a, cmd
}

func (a App) updatePriorityCmd(incidentID, priorityName string) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "priority", id: incidentID}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		priorities, err := client.ListPriorities(ctx)
		if err != nil {
			return incidentErrMsg{op: "priority", err: err}
		}
		var priorityID string
		for _, p := range priorities {
			if p.Name == priorityName {
				priorityID = p.ID
				break
			}
		}
		if priorityID == "" {
			return incidentErrMsg{op: "priority", err: fmt.Errorf("priority %q not found", priorityName)}
		}
		if err := client.UpdatePriority(ctx, incidentID, from, priorityID); err != nil {
			return incidentErrMsg{op: "priority", err: err}
		}
		return IncidentPriorityUpdated{ID: incidentID}
	}
	return tea.Batch(pending, action)
}

func (a App) updateIncidentCmd(incidentID string, diff components.EditDiff) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "edit", id: incidentID}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		opts := api.UpdateOpts{
			Title:    diff.Title,
			Urgency:  diff.Urgency,
			Priority: diff.Priority,
		}
		if _, err := client.UpdateIncident(ctx, incidentID, from, opts); err != nil {
			return incidentErrMsg{op: "edit", err: err}
		}
		return IncidentUpdated{ID: incidentID}
	}
	return tea.Batch(pending, action)
}

func (a App) mergeCmd(incidents []pagerduty.Incident) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	if len(incidents) < 2 {
		return nil
	}

	sorted := make([]pagerduty.Incident, len(incidents))
	copy(sorted, incidents)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt < sorted[j].CreatedAt
	})

	target := sorted[0]
	sourceIDs := make([]string, 0, len(sorted)-1)
	for _, inc := range sorted[1:] {
		sourceIDs = append(sourceIDs, inc.ID)
	}

	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "merge", id: target.ID}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		if err := client.MergeIncidents(ctx, target.ID, from, sourceIDs); err != nil {
			return incidentErrMsg{op: "merge", err: err}
		}
		return IncidentMerged{TargetID: target.ID}
	}
	return tea.Batch(pending, action)
}

func (a App) escalateCmd(id string) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "escalate", id: id}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		if err := client.EscalateIncident(ctx, id, from); err != nil {
			return incidentErrMsg{op: "escalate", err: err}
		}
		return IncidentEscalated{ID: id}
	}
	return tea.Batch(pending, action)
}

func (a App) detailAckCmd(id string) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "ack", id: id}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		if err := client.AckIncident(ctx, id, from); err != nil {
			return incidentErrMsg{op: "ack", err: err}
		}
		return IncidentAcked{ID: id}
	}
	return tea.Batch(pending, action)
}

func (a App) detailResolveCmd(id string) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail
	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: "resolve", id: id}
	}
	action := func() tea.Msg {
		ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
		defer cancel()
		if err := client.ResolveIncident(ctx, id, from); err != nil {
			return incidentErrMsg{op: "resolve", err: err}
		}
		return IncidentResolved{ID: id}
	}
	return tea.Batch(pending, action)
}

// browserOpenedMsg carries the result of an openBrowser attempt.
type browserOpenedMsg struct{ err error }

func openBrowser(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.CommandContext(ctx, "open", url) //nolint:gosec // URL validated by caller
		case "windows":
			cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // URL validated by caller
		default:
			cmd = exec.CommandContext(ctx, "xdg-open", url) //nolint:gosec // URL validated by caller
		}
		return browserOpenedMsg{err: cmd.Run()}
	}
}

func (a App) resolveExternalLink() string {
	var alerts []pagerduty.IncidentAlert
	if a.current == viewDetail {
		alerts = a.detail.alerts
	}
	if len(alerts) == 0 {
		return ""
	}
	body := alerts[0].Body
	if body == nil {
		return ""
	}

	// Check user-configured custom fields first.
	if a.cfg != nil {
		for _, cf := range a.cfg.CustomFields {
			if cf.Display != "link" {
				continue
			}
			val, ok := resolveFieldPath(body, cf.Path)
			if !ok {
				continue
			}
			s := fmt.Sprintf("%v", val)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}

	// Fall back to integration-detected links.
	summary := integration.Detect(body)
	if len(summary.Links) > 0 {
		return summary.Links[0].URL
	}

	return ""
}

var ageDurations = map[string]time.Duration{
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
	"60d": 60 * 24 * time.Hour,
	"90d": 90 * 24 * time.Hour,
}

// ageRange converts an age filter value to RFC3339 since/until timestamps.
func ageRange(age string) (since, until string) {
	dur, ok := ageDurations[age]
	if !ok {
		return "", ""
	}
	now := time.Now().UTC()
	return now.Add(-dur).Format(time.RFC3339), now.Format(time.RFC3339)
}

func (a App) executeInputAction(msg components.InputSubmitted) tea.Cmd {
	if a.readOnly() {
		return func() tea.Msg { return statusMsg("read-only in test mode") }
	}
	client := a.client
	appCtx := a.ctx
	from := a.fromEmail

	pending := func() tea.Msg {
		return incidentActionPendingMsg{op: msg.Action, id: msg.ID}
	}

	switch msg.Action {
	case "snooze":
		d, err := time.ParseDuration(msg.Value)
		if err != nil {
			return func() tea.Msg {
				return incidentErrMsg{op: "snooze", err: fmt.Errorf("invalid duration %q: %w", msg.Value, err)}
			}
		}
		id := msg.ID
		action := func() tea.Msg {
			ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
			defer cancel()
			if err := client.SnoozeIncident(ctx, id, from, d); err != nil {
				return incidentErrMsg{op: "snooze", err: err}
			}
			return IncidentSnoozed{ID: id}
		}
		return tea.Batch(pending, action)

	case "note":
		id := msg.ID
		content := msg.Value
		action := func() tea.Msg {
			ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
			defer cancel()
			if err := client.AddIncidentNote(ctx, id, from, content); err != nil {
				return incidentErrMsg{op: "note", err: err}
			}
			return IncidentNoteAdded{ID: id}
		}
		return tea.Batch(pending, action)

	case "resolve":
		id := msg.ID
		note := strings.TrimSpace(msg.Value)
		action := func() tea.Msg {
			// Best-effort: don't block resolve if the note fails.
			if note != "" {
				noteCtx, noteCancel := context.WithTimeout(appCtx, 15*time.Second)
				defer noteCancel()
				if err := client.AddIncidentNote(noteCtx, id, from, note); err != nil {
					clog.Warn().Err(err).Msg("could not add resolution note")
				}
			}
			resolveCtx, resolveCancel := context.WithTimeout(appCtx, 15*time.Second)
			defer resolveCancel()
			if err := client.ResolveIncident(resolveCtx, id, from); err != nil {
				return incidentErrMsg{op: "resolve", err: err}
			}
			return IncidentResolved{ID: id}
		}
		return tea.Batch(pending, action)

	case "reassign":
		id := msg.ID
		userID := msg.Value
		action := func() tea.Msg {
			ctx, cancel := context.WithTimeout(appCtx, 15*time.Second)
			defer cancel()
			if err := client.ReassignIncident(ctx, id, from, []string{userID}); err != nil {
				return incidentErrMsg{op: "reassign", err: err}
			}
			return IncidentReassigned{ID: id}
		}
		return tea.Batch(pending, action)
	}

	return nil
}
