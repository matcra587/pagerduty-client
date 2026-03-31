package cmd

import (
	"fmt"
	"os"

	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/output"
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
	Args:  cobra.NoArgs,
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

		headers := []string{"ID", "Name", "Description"}
		rows := make([][]string, len(teams))
		for i, t := range teams {
			rows[i] = []string{t.ID, t.Name, t.Description}
		}

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(teams)}
			return output.RenderAgentJSON(os.Stdout, "team list", output.ResourceTeam, teams, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, teams, isTTY)
		default:
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var teamShowCmd = &cobra.Command{
	Use:         "show <id>",
	Short:       "Show team details",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='team'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		team, err := client.GetTeam(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting team: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched team")

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "team show", output.ResourceTeam, team, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, team, isTTY)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", team.ID},
				{"Name", team.Name},
				{"Description", team.Description},
			}
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
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
