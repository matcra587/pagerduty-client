package cmd

import (
	"fmt"
	"os"

	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/table"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:     "team",
	Short:   "View PagerDuty teams",
	Long:    "List and inspect PagerDuty teams.",
	GroupID: "resources",
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams",
	Example: `# List all teams
$ pdc team list`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		query, _ := cmd.Flags().GetString("query")

		teams, err := client.ListTeams(ctx, api.ListTeamsOpts{Query: query})
		if err != nil {
			return fmt.Errorf("listing teams: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(teams)).Msg("listed teams")

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
			meta := agent.Metadata{Total: len(teams)}
			return output.RenderAgentJSON(w, "team list", compact.ResourceTeam, teams, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, teams, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("ID"))
			tbl.AddCol(table.Col("Name").Flex())
			tbl.AddCol(table.Col("Description").Flex())
			for _, t := range teams {
				tbl.Row(t.ID, t.Name, t.Description)
			}
			return tbl.Render()
		}
	},
}

var teamShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show team details",
	Example: `# Show team details
$ pdc team show PTEAM01`,
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='team'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.Team(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		team, err := client.GetTeam(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting team: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched team")

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
			return output.RenderAgentJSON(w, "team show", compact.ResourceTeam, team, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, team, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("Field").Bold())
			tbl.AddCol(table.Col("Value").Flex())
			tbl.Row("ID", team.ID)
			tbl.Row("Name", team.Name)
			tbl.Row("Description", team.Description)
			return tbl.Render()
		}
	},
}

func init() {
	rootCmd.AddCommand(teamCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamShowCmd)

	teamListCmd.Flags().String("query", "", "Filter teams by name")
	clib.Extend(teamListCmd.Flags().Lookup("query"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TEXT",
		Terse:       "name filter",
	})
}
