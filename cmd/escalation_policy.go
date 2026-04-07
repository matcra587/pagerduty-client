package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/resolve"
	"github.com/spf13/cobra"
)

var escalationPolicyCmd = &cobra.Command{
	Use:     "escalation-policy",
	Short:   "View escalation policies",
	Long:    "List and view PagerDuty escalation policies.",
	GroupID: "resources",
}

var escalationPolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List escalation policies",
	Example: `# List all escalation policies
$ pdc escalation-policy list

# Filter by name
$ pdc escalation-policy list --query platform

# Filter by team
$ pdc escalation-policy list --team T1`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		query, _ := cmd.Flags().GetString("query")
		teams, _ := cmd.Flags().GetStringSlice("team")

		r := ResolverFromContext(cmd)
		if r != nil {
			var resolveErr error
			teams, resolveErr = resolveSlice(!det.Active, teams, func(s string) (string, []resolve.Match, error) { return r.Team(ctx, s) })
			if resolveErr != nil {
				return resolveErr
			}
		}

		policies, err := client.ListEscalationPolicies(ctx, api.ListEscalationPoliciesOpts{
			Query:   query,
			TeamIDs: teams,
		})
		if err != nil {
			return fmt.Errorf("listing escalation policies: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(policies)).Msg("listed escalation policies")

		headers, rows := escalationPolicyRows(policies)

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(policies)}
			return output.RenderAgentJSON(w, "escalation-policy list", output.ResourceEscalationPolicy, policies, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, policies, isTTY)
		default:
			return output.RenderTable(w, headers, rows, isTTY)
		}
	},
}

var escalationPolicyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show escalation policy details",
	Example: `# Show escalation policy details
$ pdc escalation-policy show PABC123`,
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"clib": "dynamic-args='escalation_policy'"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		r := ResolverFromContext(cmd)
		if r != nil {
			rid, matches, fnErr := r.EscalationPolicy(ctx, args[0])
			resolved, resolveErr := resolveOrPick(!det.Active, rid, matches, fnErr)
			if resolveErr != nil {
				return resolveErr
			}
			args[0] = resolved
		}

		ep, err := client.GetEscalationPolicy(ctx, args[0])
		if err != nil {
			return fmt.Errorf("getting escalation policy: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched escalation policy")

		w := cmd.OutOrStdout()
		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(w, "escalation-policy show", output.ResourceEscalationPolicy, ep, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(w, ep, isTTY)
		default:
			teamIDs := make([]string, len(ep.Teams))
			for i, t := range ep.Teams {
				teamIDs[i] = t.ID
			}

			detailHeaders := []string{"Field", "Value"}
			detailRows := [][]string{
				{"ID", ep.ID},
				{"Name", ep.Name},
				{"Description", ep.Description},
				{"Num Loops", strconv.FormatUint(uint64(ep.NumLoops), 10)},
				{"Teams", strings.Join(teamIDs, ", ")},
			}
			if err := output.RenderTable(w, detailHeaders, detailRows, isTTY); err != nil {
				return err
			}

			if len(ep.EscalationRules) > 0 {
				fmt.Fprintln(w)
				ruleHeaders, ruleRows := escalationRuleRows(ep.EscalationRules)
				return output.RenderTable(w, ruleHeaders, ruleRows, isTTY)
			}
			return nil
		}
	},
}

func escalationPolicyRows(policies []pagerduty.EscalationPolicy) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Num Loops", "Teams"}
	rows := make([][]string, len(policies))
	for i, p := range policies {
		teamIDs := make([]string, len(p.Teams))
		for j, t := range p.Teams {
			teamIDs[j] = t.ID
		}
		rows[i] = []string{
			p.ID,
			p.Name,
			strconv.FormatUint(uint64(p.NumLoops), 10),
			strings.Join(teamIDs, ", "),
		}
	}
	return headers, rows
}

// escalationRuleRows converts escalation rules into table rows.
// Each target is formatted as "Summary (type)" with the _reference
// suffix stripped from the type.
func escalationRuleRows(rules []pagerduty.EscalationRule) ([]string, [][]string) {
	headers := []string{"Level", "Delay", "Targets"}
	rows := make([][]string, len(rules))
	for i, r := range rules {
		targets := make([]string, len(r.Targets))
		for j, t := range r.Targets {
			typeName := strings.TrimSuffix(t.Type, "_reference")
			if t.Summary != "" {
				targets[j] = fmt.Sprintf("%s (%s)", t.Summary, typeName)
			} else {
				targets[j] = fmt.Sprintf("%s (%s)", t.ID, typeName)
			}
		}
		rows[i] = []string{
			strconv.Itoa(i + 1),
			fmt.Sprintf("%d min", r.Delay),
			strings.Join(targets, ", "),
		}
	}
	return headers, rows
}

func init() {
	rootCmd.AddCommand(escalationPolicyCmd)
	escalationPolicyCmd.AddCommand(escalationPolicyListCmd)
	escalationPolicyCmd.AddCommand(escalationPolicyShowCmd)

	f := escalationPolicyListCmd.Flags()
	f.String("query", "", "Filter escalation policies by name")
	f.StringSlice("team", nil, "Filter by team IDs")

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
}
