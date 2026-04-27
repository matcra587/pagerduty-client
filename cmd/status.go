package cmd

import (
	"errors"
	"os"

	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/gechr/x/terminal"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/output"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check PagerDuty API health and token validity",
	Example: `# Check API status
$ pdc status

# Check status as JSON
$ pdc status -f json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		// Determine base URL.
		baseURL := "https://api.pagerduty.com"
		if cfg.BaseURL != "" {
			baseURL = cfg.BaseURL
		}

		// Check 1: API reachability (no token needed).
		apiReachable, statusCode := api.ProbeAPI(ctx, baseURL)
		if apiReachable {
			clog.Info().Str("status", "reachable").Msg("pagerduty api")
		} else {
			clog.Warn().Str("status", "unreachable").Str("help", "https://status.pagerduty.com").Msg("pagerduty api")
		}

		// Config file.
		cfgPath := config.DefaultConfigPath()
		if _, err := os.Stat(cfgPath); err == nil {
			clog.Info().Path("path", cfgPath).Msg("config")
		} else {
			clog.Warn().Str("status", "not found").Msg("config")
		}

		// Check 2: Token validation (skipped if no client).
		var tokenValid *bool
		var tokSrc, accountEmail string

		client := ClientFromContext(cmd)
		if client != nil {
			_, err := client.ListAbilities(ctx)
			valid := err == nil
			tokenValid = &valid

			pf := cmd.Root().PersistentFlags()
			flagToken, _ := pf.GetString("token")
			flagTokenFile, _ := pf.GetString("token-file")
			tokSrc = tokenSource(flagToken, flagTokenFile, cfg)

			if valid {
				clog.Info().Str("status", "valid").Str("source", tokSrc).Msg("token")
				accountEmail = UserEmailFromContext(cmd)
				if accountEmail == "" {
					accountEmail = cfg.Email // fall back to config defaults.email
				}
				if accountEmail != "" {
					clog.Info().Str("email", accountEmail).Msg("account")
				}
			} else {
				reason := "invalid or expired"
				if apiErr, ok := errors.AsType[*api.APIError](err); ok {
					reason = apiErr.Message
				}
				clog.Warn().Str("status", reason).Msg("token")
			}
		} else {
			clog.Warn().Str("status", "not configured").Msg("token")
		}

		// Structured output for JSON/agent mode.
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

		if format == output.FormatAgentJSON || format == output.FormatJSON {
			data := map[string]any{
				"api_reachable":   apiReachable,
				"api_status_code": statusCode,
				"config_path":     cfgPath,
			}
			if tokenValid != nil {
				data["token_valid"] = *tokenValid
				data["token_source"] = tokSrc
			}
			if accountEmail != "" {
				data["account_email"] = accountEmail
			}
			if format == output.FormatAgentJSON {
				return output.RenderAgentJSON(w, "status", compact.ResourceNone, data, nil, nil)
			}
			return output.RenderJSON(w, data, th)
		}

		// Table output already rendered via clog above.
		// Return error if API is unreachable so exit code is non-zero.
		if !apiReachable {
			return errors.New("PagerDuty API is unreachable")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
