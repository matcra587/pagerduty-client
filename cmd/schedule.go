package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/output"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:     "schedule",
	Short:   "Manage PagerDuty schedules",
	Long:    "List, view and override PagerDuty on-call schedules.",
	GroupID: "resources",
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List schedules",
	Example: `# List all schedules
$ pdc schedule list

# Filter by name
$ pdc schedule list --query primary`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		query, _ := cmd.Flags().GetString("query")

		schedules, err := client.ListSchedules(ctx, api.ListSchedulesOpts{Query: query})
		if err != nil {
			return fmt.Errorf("listing schedules: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(schedules)).Msg("listed schedules")

		headers, rows := scheduleRows(schedules)

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
			meta := agent.Metadata{Total: len(schedules)}
			return output.RenderAgentJSON(w, "schedule list", compact.ResourceSchedule, schedules, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, schedules, th)
		default:
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var scheduleShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show schedule details",
	Example: `# Show schedule details
$ pdc schedule show PSCHED01`,
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='schedule'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Schedule(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		schedule, err := client.GetSchedule(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting schedule: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched schedule")

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
			return output.RenderAgentJSON(w, "schedule show", compact.ResourceSchedule, schedule, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, schedule, th)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", schedule.ID},
				{"Name", schedule.Name},
				{"Time Zone", schedule.TimeZone},
				{"Description", schedule.Description},
			}
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

var scheduleOverrideCmd = &cobra.Command{
	Use:   "override <schedule-id>",
	Short: "Create a schedule override",
	Example: `# Create a schedule override
$ pdc schedule override --user PUSER01 --start 2024-01-15T09:00:00Z --end 2024-01-15T17:00:00Z PSCHED01`,
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='schedule'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Schedule(ctx, args[0])
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

		userID, _ := cmd.Flags().GetString("user")
		if userID == "" {
			return errors.New("--user is required")
		}

		if r != nil {
			rid, matches, fnErr := r.User(ctx, userID)
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			userID = resolved
		}

		start, _ := cmd.Flags().GetString("start")
		if start == "" {
			return errors.New("--start is required")
		}
		end, _ := cmd.Flags().GetString("end")
		if end == "" {
			return errors.New("--end is required")
		}

		if err := client.CreateOverride(ctx, args[0], from, api.CreateOverrideOpts{
			UserID: userID,
			Start:  start,
			End:    end,
		}); err != nil {
			return fmt.Errorf("creating override: %w", err)
		}

		if det.Active {
			return output.RenderAgentJSON(cmd.OutOrStdout(), "schedule override", compact.ResourceNone, map[string]string{
				"schedule": args[0],
				"user":     userID,
				"start":    start,
				"end":      end,
			}, nil, nil)
		}
		clog.Info().Str("schedule", args[0]).Msg("Override created")
		return nil
	},
}

func scheduleRows(schedules []pagerduty.Schedule) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Time Zone", "Description"}
	rows := make([][]string, len(schedules))
	for i, s := range schedules {
		rows[i] = []string{s.ID, s.Name, s.TimeZone, s.Description}
	}
	return headers, rows
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleShowCmd)
	scheduleCmd.AddCommand(scheduleOverrideCmd)

	scheduleListCmd.Flags().String("query", "", "Filter schedules by name")
	clib.Extend(scheduleListCmd.Flags().Lookup("query"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TEXT",
		Terse:       "name filter",
	})

	of := scheduleOverrideCmd.Flags()
	of.String("from", "", "Email of the acting user (defaults to current API token user)")
	of.String("user", "", "User ID to place on call (required)")
	of.String("start", "", "Override start time (ISO 8601, required)")
	of.String("end", "", "Override end time (ISO 8601, required)")

	clib.Extend(of.Lookup("from"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "EMAIL",
		Terse:       "acting user email",
	})

	clib.Extend(of.Lookup("user"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "ID",
		Complete:    "predictor=user",
		Terse:       "on-call user",
	})
	clib.Extend(of.Lookup("start"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "TIME",
		Terse:       "override start",
	})
	clib.Extend(of.Lookup("end"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "TIME",
		Terse:       "override end",
	})
}
