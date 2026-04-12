package cmd

import (
	"fmt"
	"os"
	"strings"

	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/resolve"
	"github.com/matcra587/pagerduty-client/internal/table"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var maintenanceWindowCmd = &cobra.Command{
	Use:     "maintenance-window",
	Short:   "View maintenance windows",
	Long:    "List and view PagerDuty maintenance windows.",
	GroupID: "resources",
}

var maintenanceWindowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List maintenance windows",
	Example: `# List open maintenance windows (ongoing + future)
$ pdc maintenance-window list

# List only ongoing windows
$ pdc maintenance-window list --filter ongoing

# Filter by service
$ pdc maintenance-window list --service S1`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		query, _ := cmd.Flags().GetString("query")
		teams, _ := cmd.Flags().GetStringSlice("team")
		services, _ := cmd.Flags().GetStringSlice("service")
		filter, _ := cmd.Flags().GetString("filter")

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
		}

		windows, err := client.ListMaintenanceWindows(ctx, api.ListMaintenanceWindowsOpts{
			Query:      query,
			TeamIDs:    teams,
			ServiceIDs: services,
			Filter:     filter,
		})
		if err != nil {
			return fmt.Errorf("listing maintenance windows: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(windows)).Msg("listed maintenance windows")

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
			meta := agent.Metadata{Total: len(windows)}
			return output.RenderAgentJSON(w, "maintenance-window list", compact.ResourceMaintenanceWindow, windows, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, windows, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("ID").Link(func(v string) string {
				return "https://app.pagerduty.com/maintenance_windows/" + strings.TrimSpace(v)
			}))
			tbl.AddCol(table.Col("Description").Flex())
			tbl.AddCol(table.Col("Start").TimeAgo())
			tbl.AddCol(table.Col("End").TimeAgo())
			tbl.AddCol(table.Col("Services").Flex())
			for _, mw := range windows {
				svcNames := make([]string, len(mw.Services))
				for j, s := range mw.Services {
					svcNames[j] = s.Summary
				}
				tbl.Row(mw.ID, mw.Description, mw.StartTime, mw.EndTime, strings.Join(svcNames, ", "))
			}
			return tbl.Render()
		}
	},
}

var maintenanceWindowShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show maintenance window details",
	Example: `# Show maintenance window details
$ pdc maintenance-window show PW98YIO`,
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='maintenance_window'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.MaintenanceWindow(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		mw, err := client.GetMaintenanceWindow(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting maintenance window: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched maintenance window")

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
			return output.RenderAgentJSON(w, "maintenance-window show", compact.ResourceMaintenanceWindow, mw, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, mw, th)
		default:
			svcNames := make([]string, len(mw.Services))
			for i, s := range mw.Services {
				svcNames[i] = s.Summary
			}
			teamNames := make([]string, len(mw.Teams))
			for i, t := range mw.Teams {
				teamNames[i] = t.Summary
			}

			createdBy := ""
			if mw.CreatedBy != nil {
				createdBy = mw.CreatedBy.Summary
			}

			tbl := table.New(w, th)
			tbl.AddCol(table.Col("Field").Bold())
			tbl.AddCol(table.Col("Value").Flex())
			tbl.Row("ID", mw.ID)
			tbl.Row("Description", mw.Description)
			tbl.Row("Start", mw.StartTime)
			tbl.Row("End", mw.EndTime)
			tbl.Row("Created By", createdBy)
			tbl.Row("Services", strings.Join(svcNames, ", "))
			tbl.Row("Teams", strings.Join(teamNames, ", "))
			return tbl.Render()
		}
	},
}

func init() {
	rootCmd.AddCommand(maintenanceWindowCmd)
	maintenanceWindowCmd.AddCommand(maintenanceWindowListCmd)
	maintenanceWindowCmd.AddCommand(maintenanceWindowShowCmd)
	f := maintenanceWindowListCmd.Flags()
	f.String("query", "", "Filter maintenance windows by name")
	f.StringSlice("team", nil, "Filter by team IDs")
	f.StringSlice("service", nil, "Filter by service IDs")
	f.String("filter", "open", "Time filter (past, future, ongoing, open, all)")

	clib.Extend(f.Lookup("query"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TEXT",
		Terse:       "name filter",
	})
	clib.Extend(f.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=team",
		Terse:       "team filter",
	})
	clib.Extend(f.Lookup("service"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=service",
		Terse:       "service filter",
	})
	clib.Extend(f.Lookup("filter"), clib.FlagExtra{
		Group:       "Filters",
		Enum:        []string{"past", "future", "ongoing", "open", "all"},
		EnumDefault: "open",
		Terse:       "time filter",
	})
}
