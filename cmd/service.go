package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
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

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "View PagerDuty services",
	Long:    "List and inspect PagerDuty services.",
	GroupID: "resources",
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	Args:  cobra.NoArgs,
	Example: `# List all services
$ pdc service list

# Filter by team
$ pdc service list --team PTEAM01`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		teams, _ := cmd.Flags().GetStringSlice("team")
		query, _ := cmd.Flags().GetString("query")
		sortBy, _ := cmd.Flags().GetString("sort")

		r := ResolverFromContext(cmd)
		if r != nil {
			var resolveErr error
			teams, resolveErr = resolveSlice(!det.Active, teams, func(s string) (string, []resolve.Match, error) { return r.Team(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}

		services, err := client.ListServices(ctx, api.ListServicesOpts{
			TeamIDs: teams,
			Query:   query,
			SortBy:  sortBy,
		})
		if err != nil {
			return fmt.Errorf("listing services: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(services)).Msg("listed services")

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
			meta := agent.Metadata{Total: len(services)}
			return output.RenderAgentJSON(w, "service list", compact.ResourceService, services, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, services, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("ID").Link(func(v string) string {
				return "https://app.pagerduty.com/services/" + strings.TrimSpace(v)
			}))
			tbl.AddCol(table.Col("Name").Style(func(v string) lipgloss.Style {
				return pdctheme.EntityColor(strings.TrimSpace(v))
			}))
			tbl.AddCol(table.Col("Status").StyleMap(serviceStatusStyles(th)))
			tbl.AddCol(table.Col("Description").Flex())
			for _, s := range services {
				tbl.Row(s.ID, s.Name, s.Status, s.Description)
			}
			return tbl.Render()
		}
	},
}

var serviceShowCmd = &cobra.Command{
	Use:         "show <id>",
	Short:       "Show service details",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='service'"},
	Example: `# Show service details
$ pdc service show PSVC001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Service(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		service, err := client.GetService(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting service: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched service")

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
			return output.RenderAgentJSON(w, "service show", compact.ResourceService, service, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, service, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("Field").Bold())
			tbl.AddCol(table.Col("Value").Flex())
			tbl.Row("ID", service.ID)
			tbl.Row("Name", service.Name)
			tbl.Row("Status", service.Status)
			tbl.Row("Description", service.Description)
			return tbl.Render()
		}
	},
}

func serviceStatusStyles(th *theme.Theme) map[string]lipgloss.Style {
	if th == nil {
		return nil
	}
	return map[string]lipgloss.Style{
		"active":      lipgloss.NewStyle().Foreground(th.Green.GetForeground()),
		"warning":     lipgloss.NewStyle().Foreground(th.Yellow.GetForeground()),
		"critical":    lipgloss.NewStyle().Foreground(th.Red.GetForeground()),
		"maintenance": lipgloss.NewStyle().Foreground(th.Blue.GetForeground()),
		"disabled":    lipgloss.NewStyle().Faint(true),
	}
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceShowCmd)

	f := serviceListCmd.Flags()
	f.StringSlice("team", nil, "Filter by team IDs")
	f.String("query", "", "Filter services by name or description")
	f.String("sort", "", "Sort order (e.g. name:asc)")

	clib.Extend(f.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=team",
		Terse:       "team filter",
	})
	clib.Extend(f.Lookup("query"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TEXT",
		Terse:       "name filter",
	})
	clib.Extend(f.Lookup("sort"), clib.FlagExtra{
		Group:       "Output",
		Placeholder: "FIELD:DIR",
		Terse:       "sort order",
	})
}
