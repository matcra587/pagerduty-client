package cmd

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/output"
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
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		statuses, _ := cmd.Flags().GetStringSlice("status")
		urgencies, _ := cmd.Flags().GetStringSlice("urgency")
		teams, _ := cmd.Flags().GetStringSlice("team")
		services, _ := cmd.Flags().GetStringSlice("service")
		users, _ := cmd.Flags().GetStringSlice("user")
		schedules, _ := cmd.Flags().GetStringSlice("schedule")
		if len(schedules) == 0 && len(users) == 0 && len(teams) == 0 && len(services) == 0 && cfg.Service != "" {
			services = []string{cfg.Service}
		}
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		sortBy, _ := cmd.Flags().GetString("sort")

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

		incidents, err := client.ListIncidents(ctx, api.ListIncidentsOpts{
			Statuses:   statuses,
			Urgencies:  urgencies,
			TeamIDs:    teams,
			ServiceIDs: services,
			UserIDs:    users,
			Since:      since,
			Until:      until,
			SortBy:     sortBy,
		})
		if err != nil {
			return fmt.Errorf("listing incidents: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(incidents)).Msg("listed incidents")

		headers, rows := incidentRows(incidents)

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(incidents)}
			return output.RenderAgentJSON(os.Stdout, "incident list", output.ProjectIncidentsForAgent(incidents), &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, incidents, isTTY)
		default:
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var incidentShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show incident details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		incident, err := client.GetIncident(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting incident: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched incident")

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "incident show", output.ProjectIncidentForAgent(*incident), nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, incident, isTTY)
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
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var incidentAckCmd = &cobra.Command{
	Use:   "ack <id>",
	Short: "Acknowledge an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		if err := client.AckIncident(ctx, args[0], from); err != nil {
			return fmt.Errorf("acknowledging incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "incident ack", map[string]string{"id": args[0], "status": "acknowledged"}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident acknowledged")
		return nil
	},
}

var incidentResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Resolve an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		if err := client.ResolveIncident(ctx, args[0], from); err != nil {
			return fmt.Errorf("resolving incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "incident resolve", map[string]string{"id": args[0], "status": "resolved"}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Incident resolved")
		return nil
	},
}

var incidentSnoozeCmd = &cobra.Command{
	Use:   "snooze <id>",
	Short: "Snooze an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

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
			return output.RenderAgentJSON(os.Stdout, "incident snooze", map[string]string{"id": args[0], "duration": durationStr}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Duration("duration", dur).Msg("Incident snoozed")
		return nil
	},
}

var incidentReassignCmd = &cobra.Command{
	Use:   "reassign <id>",
	Short: "Reassign an incident to one or more users",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		from, err := resolveFromEmail(cmd)
		if err != nil {
			return err
		}

		users, _ := cmd.Flags().GetStringSlice("user")
		if len(users) == 0 {
			return errors.New("--user is required")
		}

		if err := client.ReassignIncident(ctx, args[0], from, users); err != nil {
			return fmt.Errorf("reassigning incident: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "incident reassign", map[string]any{"id": args[0], "assignees": users}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Strs("users", users).Msg("Incident reassigned")
		return nil
	},
}

var incidentMergeCmd = &cobra.Command{
	Use:   "merge <target-id>",
	Short: "Merge source incidents into a target incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

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
			return output.RenderAgentJSON(os.Stdout, "incident merge", map[string]any{"target": args[0], "sources": sources}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Strs("sources", sources).Msg("Incidents merged")
		return nil
	},
}

var incidentNoteCmd = &cobra.Command{
	Use:   "note <id>",
	Short: "Add a note to an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

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
			return output.RenderAgentJSON(os.Stdout, "incident note", map[string]string{"id": args[0]}, nil, nil)
		}
		clog.Info().Link("incident", incidentURL(args[0]), args[0]).Msg("Note added")
		return nil
	},
}

// resolveFromEmail determines the acting user's email for PagerDuty write operations.
// Precedence: --from flag > cached context email > API lookup.
func resolveFromEmail(cmd *cobra.Command) (string, error) {
	from, _ := cmd.Flags().GetString("from")
	if from == "" {
		from = UserEmailFromContext(cmd)
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

func init() {
	rootCmd.AddCommand(incidentCmd)
	incidentCmd.AddCommand(incidentListCmd)
	incidentCmd.AddCommand(incidentShowCmd)
	incidentCmd.AddCommand(incidentAckCmd)
	incidentCmd.AddCommand(incidentResolveCmd)
	incidentCmd.AddCommand(incidentSnoozeCmd)
	incidentCmd.AddCommand(incidentReassignCmd)
	incidentCmd.AddCommand(incidentMergeCmd)
	incidentCmd.AddCommand(incidentNoteCmd)

	// incident list flags
	lf := incidentListCmd.Flags()
	lf.StringSlice("status", nil, "Filter by status (triggered, acknowledged, resolved)")
	lf.StringSlice("urgency", nil, "Filter by urgency (high, low)")
	lf.StringSlice("team", nil, "Filter by team IDs")
	lf.StringSlice("service", nil, "Filter by service IDs")
	lf.StringSlice("user", nil, "Filter by user IDs")
	lf.StringSlice("schedule", nil, "Filter by schedule IDs (resolves current on-call users)")
	lf.String("since", "", "Return incidents since this time (ISO 8601)")
	lf.String("until", "", "Return incidents until this time (ISO 8601)")
	lf.String("sort", "", "Sort order (e.g. created_at:asc)")

	clib.Extend(lf.Lookup("status"), clib.FlagExtra{
		Group: "Filters",
		Enum:  []string{"triggered", "acknowledged", "resolved"},
		Terse: "status filter",
	})
	clib.Extend(lf.Lookup("urgency"), clib.FlagExtra{
		Group: "Filters",
		Enum:  []string{"high", "low"},
		Terse: "urgency filter",
	})
	clib.Extend(lf.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "team filter",
	})
	clib.Extend(lf.Lookup("service"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "service filter",
	})
	clib.Extend(lf.Lookup("user"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "user filter",
	})
	clib.Extend(lf.Lookup("schedule"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "schedule filter",
	})
	clib.Extend(lf.Lookup("since"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Terse:       "start time",
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

	// shared --from flag
	for _, sub := range []*cobra.Command{
		incidentAckCmd, incidentResolveCmd, incidentSnoozeCmd,
		incidentReassignCmd, incidentMergeCmd, incidentNoteCmd,
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
		Terse:       "target user",
	})

	incidentMergeCmd.Flags().StringSlice("source", nil, "Source incident IDs to merge")
	clib.Extend(incidentMergeCmd.Flags().Lookup("source"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "ID",
		Terse:       "source incident",
	})

	incidentNoteCmd.Flags().String("content", "", "Note content")
	clib.Extend(incidentNoteCmd.Flags().Lookup("content"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "TEXT",
		Terse:       "note content",
	})
}
