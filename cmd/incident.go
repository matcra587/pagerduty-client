package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/integration"
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
	Args:  cobra.NoArgs,
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
		all, _ := cmd.Flags().GetBool("all")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		sortBy, _ := cmd.Flags().GetString("sort")

		// --all overrides --since/--until to fetch all incidents.
		if all {
			since = ""
			until = ""
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

		payload, _ := cmd.Flags().GetBool("payload")
		if payload {
			return showIncidentPayload(cmd, client, det, cfg, incident)
		}

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
func showIncidentPayload(cmd *cobra.Command, client *api.Client, det agent.DetectionResult, cfg *config.Config, incident *pagerduty.Incident) error {
	ctx := cmd.Context()
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

	switch format {
	case output.FormatAgentJSON:
		data := payloadResult(summary, body)
		return output.RenderAgentJSON(os.Stdout, "incident show --payload", data, nil, nil)
	case output.FormatJSON:
		data := payloadResult(summary, body)
		return output.RenderJSON(os.Stdout, data, isTTY)
	default:
		return renderPayloadText(summary, body)
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
func renderPayloadText(summary integration.Summary, body map[string]any) error {
	w := os.Stdout

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

	// incident list flags
	lf := incidentListCmd.Flags()
	lf.StringSlice("status", nil, "Filter by status (triggered, acknowledged, resolved)")
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
	clib.Extend(lf.Lookup("all"), clib.FlagExtra{
		Group: "Filters",
		Terse: "fetch all incidents",
	})
	clib.Extend(lf.Lookup("since"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Enum:        []string{"7d", "30d", "60d", "90d"},
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
	sf.Bool("payload", false, "Include raw alert event payload with integration detection")
	clib.Extend(sf.Lookup("payload"), clib.FlagExtra{
		Group: "Output",
		Terse: "show alert event payload",
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
