package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/resolve"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var oncallCmd = &cobra.Command{
	Use:   "oncall",
	Short: "List on-call entries",
	Long:  "Show who is currently on call, optionally filtered by team, schedule or escalation policy.",
	Example: `# List current on-call entries
$ pdc oncall

# Filter by schedule
$ pdc oncall --schedule PSCHED01`,
	Args:    cobra.NoArgs,
	GroupID: "resources",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		teams, _ := cmd.Flags().GetStringSlice("team")
		schedules, _ := cmd.Flags().GetStringSlice("schedule")
		eps, _ := cmd.Flags().GetStringSlice("escalation-policy")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")

		r := ResolverFromContext(cmd)
		if r != nil {
			var resolveErr error
			teams, resolveErr = resolveSlice(!det.Active, teams, func(s string) (string, []resolve.Match, error) { return r.Team(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
			schedules, resolveErr = resolveSlice(!det.Active, schedules, func(s string) (string, []resolve.Match, error) { return r.Schedule(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}

		oncalls, err := client.ListOnCalls(ctx, api.ListOnCallsOpts{
			ScheduleIDs:         schedules,
			EscalationPolicyIDs: eps,
			Since:               since,
			Until:               until,
		})
		if err != nil {
			return fmt.Errorf("listing on-calls: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(oncalls)).Msg("listed on-calls")

		// Client-side team filtering: the PagerDuty /oncalls API does not support
		// team_id filtering directly, so we resolve team members and filter locally.
		if len(teams) > 0 {
			memberSet := make(map[string]bool)
			for _, teamID := range teams {
				members, err := client.ListTeamMembers(ctx, teamID)
				if err != nil {
					return fmt.Errorf("listing team members for %s: %w", teamID, err)
				}
				for _, m := range members {
					memberSet[m.User.ID] = true
				}
			}
			var filtered []pagerduty.OnCall
			for _, oc := range oncalls {
				if memberSet[oc.User.ID] {
					filtered = append(filtered, oc)
				}
			}
			oncalls = filtered
		}

		headers, rows := oncallRows(oncalls)

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
			meta := agent.Metadata{Total: len(oncalls)}
			return output.RenderAgentJSON(w, "oncall", compact.ResourceOnCall, oncalls, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, oncalls, th)
		default:
			return output.RenderTable(w, headers, rows, th)
		}
	},
}

func oncallRows(oncalls []pagerduty.OnCall) ([]string, [][]string) {
	headers := []string{"User", "Schedule", "Escalation Policy", "Level", "Start", "End"}
	rows := make([][]string, len(oncalls))
	for i, oc := range oncalls {
		rows[i] = []string{
			oc.User.Summary,
			oc.Schedule.Summary,
			oc.EscalationPolicy.Summary,
			strconv.FormatUint(uint64(oc.EscalationLevel), 10),
			oc.Start,
			oc.End,
		}
	}
	return headers, rows
}

func init() {
	rootCmd.AddCommand(oncallCmd)

	f := oncallCmd.Flags()
	f.StringSlice("team", nil, "Filter by team IDs")
	f.StringSlice("schedule", nil, "Filter by schedule IDs")
	f.StringSlice("escalation-policy", nil, "Filter by escalation policy IDs")
	f.String("since", "", "Start of time window (ISO 8601)")
	f.String("until", "", "End of time window (ISO 8601)")

	clib.Extend(f.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=team",
		Terse:       "team filter",
	})
	clib.Extend(f.Lookup("schedule"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Complete:    "predictor=schedule",
		Terse:       "schedule filter",
	})
	clib.Extend(f.Lookup("escalation-policy"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "escalation policy filter",
	})
	clib.Extend(f.Lookup("since"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Terse:       "start time",
	})
	clib.Extend(f.Lookup("until"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "TIME",
		Terse:       "end time",
	})
}
