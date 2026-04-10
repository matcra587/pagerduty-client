package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"slices"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/integration"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/resolve"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var incidentCmd = &cobra.Command{
	Use:     "incident",
	Short:   "Manage PagerDuty incidents",
	Long:    "List, view and act on PagerDuty incidents.",
	GroupID: "resources",
}

var incidentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List incidents",
	Args:  cobra.NoArgs,
	Example: `# List triggered and acknowledged incidents
$ pdc incident list

# List all resolved incidents
$ pdc incident list --all --status resolved

# Filter by team
$ pdc incident list --team PTEAM01`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		statuses, _ := cmd.Flags().GetStringSlice("status")
		if len(statuses) == 0 {
			statuses = []string{"triggered", "acknowledged"}
		}
		urgencies, _ := cmd.Flags().GetStringSlice("urgency")
		teams, _ := cmd.Flags().GetStringSlice("team")
		services, _ := cmd.Flags().GetStringSlice("service")
		users, _ := cmd.Flags().GetStringSlice("user")
		schedules, _ := cmd.Flags().GetStringSlice("schedule")

		r := ResolverFromContext(cmd)
		if r != nil {
			var resolveErr error
			teams, resolveErr = resolveSlice(!det.Active, teams, func(s string) (string, []resolve.Match, error) { return r.Team(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
			services, resolveErr = resolveSlice(!det.Active, services, func(s string) (string, []resolve.Match, error) { return r.Service(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
			users, resolveErr = resolveSlice(!det.Active, users, func(s string) (string, []resolve.Match, error) { return r.User(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
			schedules, resolveErr = resolveSlice(!det.Active, schedules, func(s string) (string, []resolve.Match, error) { return r.Schedule(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}

		if len(schedules) == 0 && len(users) == 0 && len(teams) == 0 && len(services) == 0 && cfg.Service != "" {
			services = []string{cfg.Service}
		}
		all, _ := cmd.Flags().GetBool("all")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		sortBy, _ := cmd.Flags().GetString("sort")

		// --all overrides --since/--until and status defaults,
		// but preserves an explicit --status from the user.
		if all {
			since = ""
			until = ""
			if !cmd.Flags().Changed("status") {
				statuses = nil
			}
		} else {
			since = expandSinceShorthand(since)
		}

		if len(schedules) > 0 {
			oncalls, err := client.ListOnCalls(ctx, api.ListOnCallsOpts{
				ScheduleIDs: schedules,
				Earliest:    true,
			})
			if err != nil {
				return fmt.Errorf("resolving schedule on-calls: %w", err)
			}
			seen := make(map[string]bool, len(users))
			for _, u := range users {
				seen[u] = true
			}
			for _, oc := range oncalls {
				if id := oc.User.ID; id != "" && !seen[id] {
					users = append(users, id)
					seen[id] = true
				}
			}
		}

		opts := api.ListIncidentsOpts{
			Statuses:   statuses,
			Urgencies:  urgencies,
			TeamIDs:    teams,
			ServiceIDs: services,
			UserIDs:    users,
			Since:      since,
			Until:      until,
			SortBy:     sortBy,
		}
		if all {
			opts.DateRange = "all"
		}

		incidents, err := client.ListIncidents(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing incidents: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(incidents)).Msg("listed incidents")

		headers, rows := incidentRows(incidents)

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		var th *theme.Theme
		if isTTY {
			th = pdctheme.Resolve(cfg.UI.Theme)
		}

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(incidents)}
			return output.RenderAgentJSON(w, "incident list", compact.ResourceIncident, incidents, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, incidents, th)
		default:
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var incidentShowCmd = &cobra.Command{
	Use:         "show <id>",
	Short:       "Show incident details",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Show incident details
$ pdc incident show P000001

# Include alert payloads
$ pdc incident show --alerts --payload P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		incident, _, err := client.GetIncident(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting incident: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched incident")

		alerts, _ := cmd.Flags().GetBool("alerts")
		payload, _ := cmd.Flags().GetBool("payload")
		detailed, _ := cmd.Flags().GetBool("detailed")
		if alerts && payload {
			return errors.New("--alerts and --payload are mutually exclusive")
		}
		if alerts {
			alertList, err := client.ListIncidentAlerts(ctx, args[0])
			if err != nil {
				return fmt.Errorf("listing alerts: %w", err)
			}
			clog.Debug().Elapsed("duration").Int("count", len(alertList)).Msg("listed alerts")

			headers, rows := alertRows(alertList)

			w := cmd.OutOrStdout()
			isTTY := terminal.Is(os.Stdout)
			format := output.DetectFormat(output.FormatOpts{
				AgentMode: det.Active,
				Format:    cfg.Format,
				IsTTY:     isTTY,
			})

			var th *theme.Theme
			if isTTY {
				th = pdctheme.Resolve(cfg.UI.Theme)
			}

			switch format {
			case output.FormatAgentJSON:
				meta := agent.Metadata{Total: len(alertList)}
				return output.RenderAgentJSON(w, "incident show --alerts", compact.ResourceAlert, alertList, &meta, nil)
			case output.FormatJSON:
				return output.RenderJSON(w, alertList, th)
			default:
				return output.RenderTable(w, headers, rows, th)
			}
		}
		if payload {
			return showIncidentPayload(cmd.Context(), cmd.OutOrStdout(), client, det, cfg, incident)
		}

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		var th *theme.Theme
		if isTTY {
			th = pdctheme.Resolve(cfg.UI.Theme)
		}

		enriched := enrichIncident(ctx, client, incident)

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(w, "incident show", compact.ResourceIncident, enriched, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, enriched, th)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", incident.ID},
				{"Title", incident.Title},
				{"Status", incident.Status},
				{"Urgency", incident.Urgency},
				{"Service", incident.Service.Summary},
				{"Created", incident.CreatedAt},
			}
			if incident.IncidentKey != "" {
				rows = append(rows, []string{"Incident Key", incident.IncidentKey})
			}
			rows = append(rows, []string{"Alerts", fmt.Sprintf("%d triggered, %d resolved", incident.AlertCounts.Triggered, incident.AlertCounts.Resolved)})
			if incident.LastStatusChangeBy.Summary != "" {
				rows = append(rows, []string{"Last Changed By", incident.LastStatusChangeBy.Summary})
			}
			if len(incident.Teams) > 0 {
				teamNames := make([]string, len(incident.Teams))
				for i, t := range incident.Teams {
					teamNames[i] = t.Summary
				}
				rows = append(rows, []string{"Teams", strings.Join(teamNames, ", ")})
			}
			if enriched.Integration != nil {
				rows = append(rows, []string{"Source", enriched.Integration.Source})
				for _, f := range enriched.Integration.Fields {
					if !detailed && isVerboseField(f.Label) {
						continue
					}
					rows = append(rows, []string{f.Label, f.Value})
				}
			}
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var incidentAckCmd = &cobra.Command{
	Use:         "ack <id>",
	Short:       "Acknowledge an incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Acknowledge an incident
$ pdc incident ack P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		if err := client.AckIncident(ctx, args[0], from); err != nil {
			return fmt.Errorf("acknowledging incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident ack", compact.ResourceNone, map[string]string{"id": args[0], "status": "acknowledged"}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident acknowledged")
		return nil
	},
}

var incidentResolveCmd = &cobra.Command{
	Use:         "resolve <id>",
	Short:       "Resolve an incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Resolve an incident
$ pdc incident resolve P000001

# Resolve with a closing note
$ pdc incident resolve --note "Root cause identified and fixed" P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		note, _ := cmd.Flags().GetString("note")

		// Interactive terminal: prompt for an optional note.
		if note == "" && !det.Active && terminal.Is(os.Stdout) {
			if err := huh.NewText().
				Title("Resolution note (enter to skip)").
				Value(&note).
				Run(); err != nil {
				return err
			}
		}
		note = strings.TrimSpace(note)

		// Post note before resolving (same pattern as prl: comment then close).
		if note != "" {
			if err := client.AddIncidentNote(ctx, args[0], from, note); err != nil {
				clog.Warn().Err(err).Msg("could not add resolution note")
			}
		}

		if err := client.ResolveIncident(ctx, args[0], from); err != nil {
			return fmt.Errorf("resolving incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident resolve", compact.ResourceNone, map[string]string{"id": args[0], "status": "resolved"}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident resolved")
		return nil
	},
}

var incidentSnoozeCmd = &cobra.Command{
	Use:         "snooze <id>",
	Short:       "Snooze an incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Snooze for 2 hours
$ pdc incident snooze --duration 2h P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		durationStr, _ := cmd.Flags().GetString("duration")
		dur, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("parsing duration %q: %w", durationStr, err)
		}

		if err := client.SnoozeIncident(ctx, args[0], from, dur); err != nil {
			return fmt.Errorf("snoozing incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident snooze", compact.ResourceNone, map[string]string{"id": args[0], "duration": durationStr}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Duration("duration", dur).Msg("Incident snoozed")
		return nil
	},
}

var incidentReassignCmd = &cobra.Command{
	Use:         "reassign <id>",
	Short:       "Reassign an incident to one or more users",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Reassign to another user
$ pdc incident reassign --user PUSER01 P000001

# Reassign to multiple users
$ pdc incident reassign --user PUSER01 --user PUSER02 P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		users, _ := cmd.Flags().GetStringSlice("user")
		if r != nil {
			var resolveErr error
			users, resolveErr = resolveSlice(!det.Active, users, func(s string) (string, []resolve.Match, error) { return r.User(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}
		if len(users) == 0 {
			return errors.New("--user is required")
		}

		if err := client.ReassignIncident(ctx, args[0], from, users); err != nil {
			return fmt.Errorf("reassigning incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident reassign", compact.ResourceNone, map[string]any{"id": args[0], "assignees": users}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Strs("users", users).Msg("Incident reassigned")
		return nil
	},
}

var incidentMergeCmd = &cobra.Command{
	Use:         "merge <target-id>",
	Short:       "Merge source incidents into a target incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Merge two incidents into a target
$ pdc incident merge --source P000002 --source P000003 P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		sources, _ := cmd.Flags().GetStringSlice("source")
		if len(sources) == 0 {
			return errors.New("--source is required")
		}

		if err := client.MergeIncidents(ctx, args[0], from, sources); err != nil {
			return fmt.Errorf("merging incidents: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident merge", compact.ResourceNone, map[string]any{"target": args[0], "sources": sources}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Strs("sources", sources).Msg("Incidents merged")
		return nil
	},
}

var incidentNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage incident notes",
	Long:  "List and add notes on PagerDuty incidents.",
}

var incidentNoteAddCmd = &cobra.Command{
	Use:         "add <id>",
	Short:       "Add a note to an incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Add a note to an incident
$ pdc incident note add --content "Investigating the issue" P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		content, _ := cmd.Flags().GetString("content")
		if content == "" {
			return errors.New("--content is required")
		}

		if err := client.AddIncidentNote(ctx, args[0], from, content); err != nil {
			return fmt.Errorf("adding note: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident note add", compact.ResourceNone, map[string]string{"id": args[0]}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Note added")
		return nil
	},
}

var incidentNoteListCmd = &cobra.Command{
	Use:         "list <id>",
	Short:       "List notes for an incident",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# List notes for an incident
$ pdc incident note list P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		notes, err := client.ListIncidentNotes(ctx, args[0])
		if err != nil {
			return fmt.Errorf("listing notes: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(notes)).Msg("listed notes")

		headers, rows := noteRows(notes)

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		var th *theme.Theme
		if isTTY {
			th = pdctheme.Resolve(cfg.UI.Theme)
		}

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(notes)}
			return output.RenderAgentJSON(w, "incident note list", compact.ResourceNote, notes, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, notes, th)
		default:
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var incidentLogCmd = &cobra.Command{
	Use:         "log <id>",
	Short:       "Show incident timeline",
	Long:        "List log entries for an incident, showing the timeline of actions.",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Show incident timeline
$ pdc incident log P000001

# Show last 7 days of entries
$ pdc incident log --since 7d P000001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		overview, _ := cmd.Flags().GetBool("overview")

		since = expandSinceShorthand(since)

		entries, err := client.ListIncidentLogEntries(ctx, args[0], api.LogEntryOpts{
			Since:      since,
			Until:      until,
			IsOverview: overview,
		})
		if err != nil {
			return fmt.Errorf("listing log entries: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(entries)).Msg("listed log entries")

		headers, rows := logEntryRows(entries)

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		var th *theme.Theme
		if isTTY {
			th = pdctheme.Resolve(cfg.UI.Theme)
		}

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(entries)}
			return output.RenderAgentJSON(w, "incident log", compact.ResourceLogEntry, entries, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, entries, th)
		default:
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var incidentUrgencyCmd = &cobra.Command{
	Use:         "urgency <id> <high|low>",
	Short:       "Set incident urgency",
	Args:        cobra.ExactArgs(2),
	Annotations: map[string]string{"clib": "dynamic-args='incident,urgency'"},
	Example: `# Set urgency to high
$ pdc incident urgency P000001 high`,
	RunE: func(cmd *cobra.Command, args []string) error {
		level := args[1]
		if level != "high" && level != "low" {
			return fmt.Errorf("urgency must be high or low, got %q", level)
		}

		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		if err := client.SetUrgency(ctx, args[0], from, level); err != nil {
			return fmt.Errorf("setting urgency: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident urgency", compact.ResourceNone, map[string]string{"id": args[0], "urgency": level}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident urgency set to " + level)
		return nil
	},
}

var incidentTitleCmd = &cobra.Command{
	Use:         "title <id> <new-title>",
	Short:       "Set incident title",
	Args:        cobra.ExactArgs(2),
	Annotations: map[string]string{"clib": "dynamic-args='incident'"},
	Example: `# Update the incident title
$ pdc incident title P000001 "Database connection pool exhausted"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[1]
		if title == "" {
			return errors.New("title must not be empty")
		}

		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		if err := client.SetTitle(ctx, args[0], from, title); err != nil {
			return fmt.Errorf("setting title: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident title", compact.ResourceNone, map[string]string{"id": args[0], "title": title}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident title updated")
		return nil
	},
}

var incidentResolveAlertCmd = &cobra.Command{
	Use:         "resolve-alert <incident-id> <alert-id>...",
	Short:       "Resolve one or more alerts within an incident",
	Args:        cobra.MinimumNArgs(2),
	Annotations: map[string]string{"clib": "dynamic-args='incident,alert'"},
	Example: `# Resolve a specific alert within an incident
$ pdc incident resolve-alert P000001 A000001

# Resolve multiple alerts
$ pdc incident resolve-alert P000001 A000001 A000002`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Incident(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		incidentID := args[0]
		alertIDs := args[1:]

		if err := client.ResolveAlerts(ctx, incidentID, from, alertIDs); err != nil {
			return fmt.Errorf("resolving alerts: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "incident resolve-alert", compact.ResourceNone, map[string]any{"incident_id": incidentID, "resolved": alertIDs}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(incidentID), incidentID).Int("count", len(alertIDs)).Msg(fmt.Sprintf("%d alerts resolved", len(alertIDs)))
		return nil
	},
}

// resolveFromEmail determines the acting user's email for PagerDuty write operations.
// Precedence: --from flag > cached context email > config email > API lookup.
func resolveFromEmail(cmd *cobra.Command) (string, error) {
	from, _ := cmd.Flags().GetString("from")
	if from == "" {
		from = UserEmailFromContext(cmd)
	}
	if from == "" {
		if cfg := ConfigFromContext(cmd); cfg != nil && cfg.Email != "" {
			from = cfg.Email
		}
	}
	if from == "" {
		client := ClientFromContext(cmd)
		if client == nil {
			return "", errors.New("no API client available; check your token configuration")
		}
		u, err := client.GetCurrentUser(cmd.Context())
		if err != nil {
			return "", fmt.Errorf("resolving current user: %w", err)
		}
		from = u.Email
	}
	addr, err := parseFromEmail(from)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// parseFromEmail validates and extracts the bare email address.
// Accepts both "user@example.com" and "Name <user@example.com>".
func parseFromEmail(email string) (string, error) {
	a, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("invalid email %q: %w", email, err)
	}
	return a.Address, nil
}

// showIncidentPayload fetches the first alert's raw body payload, runs
// integration detection and displays the source, extracted fields and raw JSON.
func showIncidentPayload(ctx context.Context, w io.Writer, client *api.Client, det agent.DetectionResult, cfg *config.Config, incident *pagerduty.Incident) error {
	alerts, err := client.ListIncidentAlerts(ctx, incident.ID)
	if err != nil {
		return fmt.Errorf("fetching alerts: %w", err)
	}
	if len(alerts) == 0 {
		clog.Warn().Str("incident", incident.ID).Msg("no alerts found")
		return nil
	}

	body := alerts[0].Body
	summary := integration.Detect(body)

	isTTY := terminal.Is(os.Stdout)
	format := output.DetectFormat(output.FormatOpts{
		AgentMode: det.Active,
		Format:    cfg.Format,
		IsTTY:     isTTY,
	})

	var th *theme.Theme
	if isTTY {
		th = pdctheme.Resolve(cfg.UI.Theme)
	}

	switch format {
	case output.FormatAgentJSON:
		data := payloadResult(summary, body)
		return output.RenderAgentJSON(w, "incident show --payload", compact.ResourceNone, data, nil, nil)
	case output.FormatJSON:
		data := payloadResult(summary, body)
		return output.RenderJSON(w, data, th)
	default:
		return renderPayloadText(w, summary, body)
	}
}

// payloadField is a JSON-serialisable field extracted from an alert body.
type payloadField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// payloadLink is a JSON-serialisable link extracted from an alert body.
type payloadLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// enrichedIncident wraps an incident with parsed integration fields
// for JSON output.
type enrichedIncident struct {
	*pagerduty.Incident
	Integration *integrationSummary `json:"integration,omitempty"`
}

// integrationSummary holds parsed fields from an alert body.
// This is a JSON-friendly projection of integration.Summary —
// integration.Field carries a Type (FieldBadge, FieldCode, etc.)
// for TUI rendering that must not leak into agent/JSON output.
type integrationSummary struct {
	Source string         `json:"source"`
	Fields []payloadField `json:"fields,omitempty"`
	Links  []payloadLink  `json:"links,omitempty"`
}

// verboseFields are integration fields hidden from the default table
// view because they duplicate incident metadata or add noise. Shown
// with --detailed.
var verboseFields = map[string]struct{}{
	"Summary":   {},
	"Condition": {},
	"Metric":    {},
	"Body":      {},
	"Tags":      {},
	"Title":     {},
	"Query":     {},
}

func isVerboseField(label string) bool {
	_, ok := verboseFields[label]
	return ok
}

// enrichIncident fetches the first alert body for the incident and
// runs integration detection to extract structured fields. Returns
// the incident without integration fields when there are no alerts
// or detection fails.
func enrichIncident(ctx context.Context, client *api.Client, incident *pagerduty.Incident) enrichedIncident {
	result := enrichedIncident{Incident: incident}

	alerts, err := client.ListIncidentAlerts(ctx, incident.ID)
	if err != nil {
		clog.Debug().Err(err).Msg("failed to fetch alerts for enrichment")
		return result
	}
	if len(alerts) == 0 {
		return result
	}

	body := alerts[0].Body
	if len(body) == 0 {
		return result
	}

	summary := integration.Detect(body)
	if summary.Source == "" {
		return result
	}

	is := &integrationSummary{Source: summary.Source}
	for _, f := range summary.Fields {
		is.Fields = append(is.Fields, payloadField{Label: f.Label, Value: f.Value})
	}
	for _, l := range summary.Links {
		is.Links = append(is.Links, payloadLink{Label: l.Label, URL: l.URL})
	}
	result.Integration = is

	return result
}

// payloadResult builds a structured result for JSON/agent output.
func payloadResult(summary integration.Summary, body map[string]any) map[string]any {
	fields := make([]payloadField, len(summary.Fields))
	for i, f := range summary.Fields {
		fields[i] = payloadField{Label: f.Label, Value: f.Value}
	}
	links := make([]payloadLink, len(summary.Links))
	for i, l := range summary.Links {
		links[i] = payloadLink{Label: l.Label, URL: l.URL}
	}
	return map[string]any{
		"source": summary.Source,
		"fields": fields,
		"links":  links,
		"body":   body,
	}
}

// renderPayloadText writes human-readable alert payload output.
func renderPayloadText(w io.Writer, summary integration.Summary, body map[string]any) error {
	_, _ = fmt.Fprintf(w, "Detected source: %s\n\n", summary.Source)

	if len(summary.Fields) > 0 {
		_, _ = fmt.Fprintln(w, "Extracted fields:")
		for _, f := range summary.Fields {
			val := f.Value
			if len(val) > 80 {
				val = val[:77] + "..."
			}
			_, _ = fmt.Fprintf(w, "  %-16s %s\n", f.Label+":", val)
		}
		_, _ = fmt.Fprintln(w)
	}

	for _, l := range summary.Links {
		_, _ = fmt.Fprintf(w, "  %s: %s\n", l.Label, l.URL)
	}
	if len(summary.Links) > 0 {
		_, _ = fmt.Fprintln(w)
	}

	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling body: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Raw alert body:")
	_, _ = fmt.Fprintln(w, string(raw))

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Tip: if fields are missing, configure custom_fields in your config file.")

	return nil
}

// expandSinceShorthand converts shorthand duration values (7d, 30d, 60d, 90d)
// to RFC3339 timestamps. Other values (including ISO 8601) pass through unchanged.
func expandSinceShorthand(s string) string {
	var dur time.Duration
	switch s {
	case "7d":
		dur = 7 * 24 * time.Hour
	case "30d":
		dur = 30 * 24 * time.Hour
	case "60d":
		dur = 60 * 24 * time.Hour
	case "90d":
		dur = 90 * 24 * time.Hour
	default:
		return s
	}
	return time.Now().UTC().Add(-dur).Format(time.RFC3339)
}

func incidentRows(incidents []pagerduty.Incident) ([]string, [][]string) {
	headers := []string{"ID", "Title", "Status", "Urgency", "Service", "Created"}
	rows := make([][]string, len(incidents))
	for i, inc := range incidents {
		rows[i] = []string{
			inc.ID,
			inc.Title,
			inc.Status,
			inc.Urgency,
			inc.Service.Summary,
			inc.CreatedAt,
		}
	}
	return headers, rows
}

func incidentURL(id string) string {
	return "https://app.pagerduty.com/incidents/" + id
}

func alertRows(alerts []pagerduty.IncidentAlert) ([]string, [][]string) {
	headers := []string{"ID", "Status", "Severity", "Summary", "Created"}
	rows := make([][]string, len(alerts))
	for i, a := range alerts {
		rows[i] = []string{
			a.ID,
			a.Status,
			a.Severity,
			a.Summary,
			a.CreatedAt,
		}
	}
	return headers, rows
}

func noteRows(notes []pagerduty.IncidentNote) ([]string, [][]string) {
	headers := []string{"ID", "User", "Content", "Created"}
	rows := make([][]string, len(notes))
	for i, n := range notes {
		rows[i] = []string{
			n.ID,
			n.User.Summary,
			n.Content,
			n.CreatedAt,
		}
	}
	return headers, rows
}

func logEntryRows(entries []pagerduty.LogEntry) ([]string, [][]string) {
	headers := []string{"Time", "Type", "Agent", "Summary"}
	rows := make([][]string, len(entries))
	for i, e := range entries {
		entryType := strings.TrimSuffix(e.Type, "_log_entry")
		rows[i] = []string{
			e.CreatedAt,
			entryType,
			e.Agent.Summary,
			logEntrySummary(e),
		}
	}
	return headers, rows
}

func logEntrySummary(e pagerduty.LogEntry) string {
	if s, ok := e.Channel.Raw["summary"].(string); ok && s != "" {
		return s
	}
	// EventDetails is a map; sort keys for deterministic output.
	keys := make([]string, 0, len(e.EventDetails))
	for k := range e.EventDetails {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if v := e.EventDetails[k]; v != "" {
			return v
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(incidentCmd)
	incidentCmd.AddCommand(incidentListCmd)
	incidentCmd.AddCommand(incidentShowCmd)
	incidentCmd.AddCommand(incidentAckCmd)
	incidentCmd.AddCommand(incidentResolveCmd)
	incidentResolveCmd.Flags().StringP("note", "n", "", "Resolution note (optional)")
	clib.Extend(incidentResolveCmd.Flags().Lookup("note"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "TEXT",
		Terse:       "resolution note",
	})
	incidentCmd.AddCommand(incidentSnoozeCmd)
	incidentCmd.AddCommand(incidentReassignCmd)
	incidentCmd.AddCommand(incidentMergeCmd)
	incidentCmd.AddCommand(incidentNoteCmd)
	incidentNoteCmd.AddCommand(incidentNoteListCmd)
	incidentNoteCmd.AddCommand(incidentNoteAddCmd)
	incidentCmd.AddCommand(incidentLogCmd)
	incidentCmd.AddCommand(incidentUrgencyCmd)
	incidentCmd.AddCommand(incidentTitleCmd)
	incidentCmd.AddCommand(incidentResolveAlertCmd)

	logF := incidentLogCmd.Flags()
	logF.String("since", "", "Show entries since this time (e.g. 7d, 30d or ISO 8601)")
	logF.String("until", "", "Show entries until this time (ISO 8601)")
	logF.Bool("overview", false, "Show overview entries only")

	clib.Extend(logF.Lookup("since"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Enum:        []string{"7d", "30d", "60d", "90d"},
		EnumTerse:   []string{"last 7 days", "last 30 days", "last 60 days", "last 90 days"},
		Terse:       "start time (shorthand or ISO 8601)",
	})
	clib.Extend(logF.Lookup("until"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Terse:       "end time",
	})
	clib.Extend(logF.Lookup("overview"), clib.FlagExtra{
		Group: "Filters",
		Terse: "overview entries only",
	})

	// incident list flags
	lf := incidentListCmd.Flags()
	lf.StringSlice("status", nil, "Filter by status (default: triggered,acknowledged)")
	lf.StringSlice("urgency", nil, "Filter by urgency (high, low)")
	lf.StringSlice("team", nil, "Filter by team IDs")
	lf.StringSlice("service", nil, "Filter by service IDs")
	lf.StringSlice("user", nil, "Filter by user IDs")
	lf.StringSlice("schedule", nil, "Filter by schedule IDs (resolves current on-call users)")
	lf.Bool("all", false, "Fetch all incidents (overrides --since/--until)")
	lf.String("since", "", "Return incidents since this time (e.g. 7d, 30d or ISO 8601)")
	lf.String("until", "", "Return incidents until this time (ISO 8601)")
	lf.String("sort", "", "Sort order (e.g. created_at:asc)")

	clib.Extend(lf.Lookup("status"), clib.FlagExtra{
		Group:     "Filters",
		Enum:      []string{"triggered", "acknowledged", "resolved"},
		EnumTerse: []string{"awaiting response", "responder engaged", "incident closed"},
		Terse:     "status filter",
	})
	clib.Extend(lf.Lookup("urgency"), clib.FlagExtra{
		Group:     "Filters",
		Enum:      []string{"high", "low"},
		EnumTerse: []string{"high-urgency notifications", "low-urgency notifications"},
		Terse:     "urgency filter",
	})
	clib.Extend(lf.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=team",
		Terse:       "team filter",
	})
	clib.Extend(lf.Lookup("service"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=service",
		Terse:       "service filter",
	})
	clib.Extend(lf.Lookup("user"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=user",
		Terse:       "user filter",
	})
	clib.Extend(lf.Lookup("schedule"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=schedule",
		Terse:       "schedule filter",
	})
	clib.Extend(lf.Lookup("all"), clib.FlagExtra{
		Group: "Filters",
		Terse: "fetch all incidents",
	})
	clib.Extend(lf.Lookup("since"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Enum:        []string{"7d", "30d", "60d", "90d"},
		EnumTerse:   []string{"last 7 days", "last 30 days", "last 60 days", "last 90 days"},
		Terse:       "start time (shorthand or ISO 8601)",
	})
	clib.Extend(lf.Lookup("until"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Terse:       "end time",
	})
	clib.Extend(lf.Lookup("sort"), clib.FlagExtra{
		Group:       "Output",
		Placeholder: "FIELD:DIR",
		Terse:       "sort order",
	})

	// incident show flags
	sf := incidentShowCmd.Flags()
	sf.Bool("alerts", false, "List alerts grouped under the incident")
	clib.Extend(sf.Lookup("alerts"), clib.FlagExtra{
		Group: "Output",
		Terse: "show grouped alerts",
	})
	sf.Bool("payload", false, "Include raw alert event payload with integration detection")
	clib.Extend(sf.Lookup("payload"), clib.FlagExtra{
		Group: "Output",
		Terse: "show alert event payload",
	})
	sf.Bool("detailed", false, "Show all integration fields in table output")
	clib.Extend(sf.Lookup("detailed"), clib.FlagExtra{
		Group: "Output",
		Terse: "show all integration fields",
	})

	// shared --from flag
	for _, sub := range []*cobra.Command{
		incidentAckCmd, incidentResolveCmd, incidentSnoozeCmd,
		incidentReassignCmd, incidentMergeCmd, incidentNoteAddCmd,
		incidentUrgencyCmd, incidentTitleCmd, incidentResolveAlertCmd,
	} {
		sub.Flags().String("from", "", "Email of the acting user (defaults to current API token user)")
		clib.Extend(sub.Flags().Lookup("from"), clib.FlagExtra{
			Group:       "Action",
			Placeholder: "EMAIL",
			Terse:       "acting user email",
		})
	}

	incidentSnoozeCmd.Flags().String("duration", "4h", "Snooze duration (e.g. 4h, 30m)")
	clib.Extend(incidentSnoozeCmd.Flags().Lookup("duration"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "DURATION",
		Terse:       "snooze duration",
	})

	incidentReassignCmd.Flags().StringSlice("user", nil, "Target user IDs")
	clib.Extend(incidentReassignCmd.Flags().Lookup("user"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "ID",
		Complete:    "predictor=user",
		Terse:       "target user",
	})

	incidentMergeCmd.Flags().StringSlice("source", nil, "Source incident IDs to merge")
	clib.Extend(incidentMergeCmd.Flags().Lookup("source"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "ID",
		Complete:    "predictor=incident",
		Terse:       "source incident",
	})

	incidentNoteAddCmd.Flags().String("content", "", "Note content (required)")
	clib.Extend(incidentNoteAddCmd.Flags().Lookup("content"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "TEXT",
		Terse:       "note content",
	})
}
