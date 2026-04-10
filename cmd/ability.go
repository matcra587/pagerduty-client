package cmd

import (
	"errors"
	"fmt"
	"os"

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

var abilityCmd = &cobra.Command{
	Use:     "ability",
	Short:   "View account abilities",
	Long:    "List and test account abilities. Abilities describe your account's capabilities by feature name, based on your pricing plan or account state.",
	GroupID: "resources",
}

var abilityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List account abilities",
	Example: `# List all account abilities
$ pdc ability list

# List abilities as JSON
$ pdc ability list -f json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		abilities, err := client.ListAbilities(ctx)
		if err != nil {
			return fmt.Errorf("listing abilities: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(abilities)).Msg("listed abilities")

		out := cmd.OutOrStdout()
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
			meta := agent.Metadata{Total: len(abilities)}
			return output.RenderAgentJSON(out, "ability list", compact.ResourceNone, abilities, &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(out, abilities, th)
		default:
			items := make([]string, len(abilities))
			for i, a := range abilities {
				items[i] = a.Display
			}
			w := terminal.Width(os.Stdout)
			return output.RenderColumns(out, items, w, th)
		}
	},
}

var abilityTestCmd = &cobra.Command{
	Use:   "test <ability>",
	Short: "Test if account has an ability",
	Example: `# Check if account has SSO
$ pdc ability test sso

# Check for teams support
$ pdc ability test teams`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		name := args[0]
		err := client.TestAbility(ctx, name)

		available := err == nil
		if err != nil && !errors.Is(err, api.ErrPaymentRequired) {
			return fmt.Errorf("testing ability: %w", err)
		}

		data := map[string]any{
			"ability":   name,
			"available": available,
		}

		out := cmd.OutOrStdout()
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
			return output.RenderAgentJSON(out, "ability test", compact.ResourceNone, data, nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(out, data, th)
		default:
			if available {
				clog.Info().Str("ability", name).Msg("available")
			} else {
				clog.Warn().Str("ability", name).Msg("unavailable")
			}
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(abilityCmd)
	abilityCmd.AddCommand(abilityListCmd)
	abilityCmd.AddCommand(abilityTestCmd)
}
