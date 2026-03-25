package cmd

import (
	"fmt"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/output"
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
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		teams, _ := cmd.Flags().GetStringSlice("team")
		query, _ := cmd.Flags().GetString("query")

		users, err := client.ListUsers(ctx, api.ListUsersOpts{
			TeamIDs: teams,
			Query:   query,
		})
		if err != nil {
			return fmt.Errorf("listing users: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(users)).Msg("listed users")

		headers, rows := userRows(users)

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(users)}
			return output.RenderAgentJSON(os.Stdout, "user list", users, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, users, isTTY)
		default:
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var userShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show user details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		user, err := client.GetUser(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting user: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched user")

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "user show", user, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, user, isTTY)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", user.ID},
				{"Name", user.Name},
				{"Email", user.Email},
				{"Role", user.Role},
				{"Time Zone", user.Timezone},
			}
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the current API token user",
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

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "user me", user, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, user, isTTY)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", user.ID},
				{"Name", user.Name},
				{"Email", user.Email},
				{"Role", user.Role},
				{"Time Zone", user.Timezone},
			}
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

func userRows(users []pagerduty.User) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Email", "Role"}
	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{u.ID, u.Name, u.Email, u.Role}
	}
	return headers, rows
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
		Terse:       "team filter",
	})
	clib.Extend(f.Lookup("query"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TEXT",
		Terse:       "name filter",
	})
}
