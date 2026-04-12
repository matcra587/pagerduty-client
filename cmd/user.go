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

var userCmd = &cobra.Command{
	Use:     "user",
	Short:   "View PagerDuty users",
	Long:    "List and inspect PagerDuty users.",
	GroupID: "resources",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Args:  cobra.NoArgs,
	Example: `# List all users
$ pdc user list`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		teams, _ := cmd.Flags().GetStringSlice("team")
		query, _ := cmd.Flags().GetString("query")

		r := ResolverFromContext(cmd)
		if r != nil {
			var resolveErr error
			teams, resolveErr = resolveSlice(!det.Active, teams, func(s string) (string, []resolve.Match, error) { return r.Team(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}

		users, err := client.ListUsers(ctx, api.ListUsersOpts{
			TeamIDs: teams,
			Query:   query,
		})
		if err != nil {
			return fmt.Errorf("listing users: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(users)).Msg("listed users")

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
			meta := agent.Metadata{Total: len(users)}
			return output.RenderAgentJSON(w, "user list", compact.ResourceUser, users, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, users, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("ID").Link(func(v string) string {
				return "https://app.pagerduty.com/users/" + strings.TrimSpace(v)
			}))
			tbl.AddCol(table.Col("Name").Style(func(v string) lipgloss.Style {
				return pdctheme.EntityColor(strings.TrimSpace(v))
			}).Flex())
			tbl.AddCol(table.Col("Email"))
			tbl.AddCol(table.Col("Role"))
			for _, u := range users {
				tbl.Row(u.ID, u.Name, u.Email, u.Role)
			}
			return tbl.Render()
		}
	},
}

var userShowCmd = &cobra.Command{
	Use:         "show <id>",
	Short:       "Show user details",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='user'"},
	Example: `# Show user details
$ pdc user show PUSER01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.User(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		user, err := client.GetUser(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting user: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched user")

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
			return output.RenderAgentJSON(w, "user show", compact.ResourceUser, user, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, user, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("Field").Bold())
			tbl.AddCol(table.Col("Value").Flex())
			tbl.Row("ID", user.ID)
			tbl.Row("Name", user.Name)
			tbl.Row("Email", user.Email)
			tbl.Row("Role", user.Role)
			tbl.Row("Time Zone", user.Timezone)
			return tbl.Render()
		}
	},
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the current API token user",
	Args:  cobra.NoArgs,
	Example: `# Show the current API token owner
$ pdc user me`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		user, err := client.GetCurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("getting current user: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", user.ID).Msg("fetched current user")

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
			return output.RenderAgentJSON(w, "user me", compact.ResourceUser, user, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, user, th)
		default:
			tbl := table.New(w, th)
			tbl.AddCol(table.Col("Field").Bold())
			tbl.AddCol(table.Col("Value").Flex())
			tbl.Row("ID", user.ID)
			tbl.Row("Name", user.Name)
			tbl.Row("Email", user.Email)
			tbl.Row("Role", user.Role)
			tbl.Row("Time Zone", user.Timezone)
			return tbl.Render()
		}
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userMeCmd)

	f := userListCmd.Flags()
	f.StringSlice("team", nil, "Filter by team IDs")
	f.String("query", "", "Filter users by name or email")

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
}
