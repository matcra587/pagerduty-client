package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/output"
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

		headers, rows := maintenanceWindowRows(windows)

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(windows)}
			return output.RenderAgentJSON(os.Stdout, "maintenance-window list", output.ResourceMaintenanceWindow, windows, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, windows, isTTY)
		default:
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
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

		mw, err := client.GetMaintenanceWindow(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting maintenance window: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched maintenance window")

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "maintenance-window show", output.ResourceMaintenanceWindow, mw, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, mw, isTTY)
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

			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", mw.ID},
				{"Description", mw.Description},
				{"Start", mw.StartTime},
				{"End", mw.EndTime},
				{"Created By", createdBy},
				{"Services", strings.Join(svcNames, ", ")},
				{"Teams", strings.Join(teamNames, ", ")},
			}
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

func maintenanceWindowRows(windows []pagerduty.MaintenanceWindow) ([]string, [][]string) {
	headers := []string{"ID", "Description", "Start", "End", "Services"}
	rows := make([][]string, len(windows))
	for i, w := range windows {
		svcNames := make([]string, len(w.Services))
		for j, s := range w.Services {
			svcNames[j] = s.Summary
		}
		rows[i] = []string{
			w.ID,
			w.Description,
			w.StartTime,
			w.EndTime,
			strings.Join(svcNames, ", "),
		}
	}
	return headers, rows
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
