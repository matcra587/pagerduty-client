// Package cmd contains Cobra command definitions for the pdc CLI.
// Each file wires flags and subcommands; all business logic lives in internal/.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/complete"
	"github.com/gechr/clib/help"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/matcra587/pagerduty-client/internal/resolve"
	"github.com/matcra587/pagerduty-client/internal/tui"
	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/matcra587/pagerduty-client/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// contextKey is a package-local type for context value keys to avoid collisions.
type contextKey string

const (
	configKey       contextKey = "config"
	clientKey       contextKey = "client"
	agentKey        contextKey = "agent"
	resolverKey     contextKey = "resolver"
	userEmailKey    contextKey = "userEmail"
	updateResultKey contextKey = "updateResult"
)

const statusHint = "PagerDuty may be experiencing issues — check https://status.pagerduty.com"

// isOutageError returns true for errors that suggest PD is down.
func isOutageError(err error) bool {
	if strings.Contains(err.Error(), "request failed after") {
		return true
	}
	if apiErr, ok := errors.AsType[*api.APIError](err); ok && apiErr.StatusCode >= 500 {
		return true
	}
	return false
}

// rootCmd is the base command for pdc.
var rootCmd = &cobra.Command{
	Use:   "pdc",
	Short: "PagerDuty CLI",
	Long:  "AI-agent-ready CLI for PagerDuty. Every command produces structured, self-describing output. Terminal output is sanitised to prevent control character injection.",
	Example: `# Launch the TUI dashboard
$ pdc -i

# List triggered incidents as JSON
$ pdc incident list --format json

# Acknowledge an incident
$ pdc incident ack P000001

# Show who is on call
$ pdc oncall`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// completion subcommand prints static scripts; skip full setup.
		if cmd.Name() == "completion" || (cmd.Parent() != nil && cmd.Parent().Name() == "completion") {
			return nil
		}

		pf := cmd.Root().PersistentFlags()

		state, err := loadConfigAndFlags(pf)
		if err != nil {
			return err
		}

		if cmd.Name() == "status" {
			state.cfg.SetTokenOptional()
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		flagToken, _ := pf.GetString("token")
		ctx, err = resolveAndStore(ctx, pf, state, flagToken)
		if err != nil {
			return err
		}
		cmd.SetContext(ctx)

		// Check for updates (cached for 24h, 3s timeout).
		isTTY := terminal.Is(os.Stdout)
		if update.ShouldCheck(state.det.Active, isTTY) {
			ch, _ := update.ParseChannel(state.cfg.UpdateChannel)
			result := update.CheckForUpdate(cmd.Context(), ch)
			update.NotifyCLI(result)
			cmd.SetContext(context.WithValue(cmd.Context(), updateResultKey, result))
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		det := AgentFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		isTTY := terminal.Is(os.Stdout)

		// Retrieve cached update check result from PersistentPreRunE.
		updateResult := UpdateResultFromContext(cmd)

		if !det.Active && isTTY && cfg.Interactive {
			if updateResult.UpdateAvail && !updateResult.Dismissed {
				ch, _ := update.ParseChannel(cfg.UpdateChannel)
				currentRef := version.Version
				if updateResult.Channel == update.ChannelDev {
					currentRef = version.ResolvedCommit()
				}
				choice, err := tui.RunUpdatePrompt(currentRef, updateResult.LatestRef, ch)
				if err != nil {
					clog.Debug().Err(err).Msg("update prompt error")
				}

				switch choice {
				case tui.UpdateNow:
					return update.Run(cmd.Context(), ch)
				case tui.UpdateDismiss:
					update.DismissVersion(updateResult.LatestRef)
				}
			}

			client := ClientFromContext(cmd)
			email := UserEmailFromContext(cmd)
			if email == "" && cfg.Email != "" {
				email = cfg.Email
			}
			if email == "" && client != nil {
				if u, err := client.GetCurrentUser(cmd.Context()); err == nil {
					email = u.Email
				}
			}
			app := tui.New(cmd.Context(), client, cfg, email)
			var p *tea.Program
			wc := tui.NewWheelCoalescer(func(msg tea.Msg) {
				if p != nil {
					p.Send(msg)
				}
			})
			defer wc.Stop()
			p = tea.NewProgram(app,
				tea.WithContext(cmd.Context()),
				tea.WithFilter(wc.Filter),
			)
			_, err := p.Run()
			return err
		}

		return cmd.Help()
	},
}

// Execute runs the root command and returns any error.
func Execute() error {
	if err := setup(); err != nil {
		clog.Error().Err(err).Send()
		return err
	}
	err := rootCmd.Execute()
	if err != nil {
		clog.Error().Err(err).Send()
		if isOutageError(err) {
			clog.Warn().Msg(statusHint)
		}
	}
	return err
}

// setup handles pre-parse completion and disables cobra's built-in
// completion subcommand. Called from Execute to guarantee init() across
// all cmd/ files has run.
func setup() error {
	// Shell completion subcommand (for Homebrew: pdc completion <shell>).
	// Also disables cobra's built-in completion subcommand.
	rootCmd.AddCommand(clib.CompletionCommand(rootCmd, func() *complete.Generator {
		gen := complete.NewGenerator("pdc", complete.WithOrder(complete.OrderKeep)).FromFlags(clib.FlagMeta(rootCmd))
		gen.Subs = clib.Subcommands(rootCmd)
		return gen
	}))

	flags, positional, ok := clib.Preflight()
	if !ok {
		return nil
	}

	// Resolve token for dynamic completions (env → keyring).
	// Static completions (install/print/uninstall) don't need a token.
	// --token and --token-file are unavailable because Cobra has not
	// parsed flags yet; this is acceptable because completions are
	// best-effort and --token is the least recommended credential path.
	var token string
	var apiOpts []api.Option

	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg.BaseURL != "" {
		apiOpts = append(apiOpts, api.WithBaseURL(cfg.BaseURL))
	}

	if v := os.Getenv("PDC_TOKEN"); v != "" {
		token = v
	} else if cfgErr == nil && cfg.CredentialSource == credential.SourceKeyring {
		p := credential.KeyringProvider{}
		if t, err := p.Provide(context.Background()); err == nil {
			token = t
		}
	}

	gen := complete.NewGenerator("pdc", complete.WithOrder(complete.OrderKeep)).FromFlags(clib.FlagMeta(rootCmd))
	gen.Subs = clib.Subcommands(rootCmd)
	handled, err := flags.Handle(gen, completionHandler(token, cfg, apiOpts...), complete.WithArgs(positional))
	if err != nil {
		return err
	}
	if handled {
		os.Exit(0) //nolint:revive // completion handler must exit after handling
	}
	return nil
}

type preRunState struct {
	cfg     *config.Config
	det     agent.DetectionResult
	apiOpts []api.Option
}

// loadConfigAndFlags loads configuration from file/env, applies flag
// overrides, sets up logging and detects agent mode.
func loadConfigAndFlags(pf *pflag.FlagSet) (preRunState, error) {
	var opts []config.Option
	if cfgPath, _ := pf.GetString("config"); cfgPath != "" {
		opts = append(opts, config.WithPath(cfgPath))
	}
	if team, _ := pf.GetString("team"); team != "" {
		opts = append(opts, config.WithTeam(team))
	}
	if service, _ := pf.GetString("service"); service != "" {
		opts = append(opts, config.WithService(service))
	}

	cfg, err := config.Load(opts...)
	if err != nil {
		return preRunState{}, fmt.Errorf("loading config: %w", err)
	}

	if pf.Changed("format") {
		cfg.Format, _ = pf.GetString("format")
	}
	if pf.Changed("interactive") {
		if v, _ := pf.GetBool("interactive"); v {
			cfg.Interactive = true
		}
	}
	if debug, _ := pf.GetBool("debug"); debug {
		cfg.Debug = true
	}

	clog.SetEnvPrefix("PDC")
	clog.SetVerbose(cfg.Debug)

	if colorMode, _ := pf.GetString("color"); colorMode != "" {
		switch colorMode {
		case "auto":
			// default, no action
		case "always":
			clog.SetColorMode(clog.ColorAlways)
		case "never":
			clog.SetColorMode(clog.ColorNever)
		default:
			return preRunState{}, fmt.Errorf("invalid colour mode %q: must be \"auto\", \"always\" or \"never\"", colorMode)
		}
	}

	agentFlag, _ := pf.GetBool("agent")
	det := agent.DetectWithFlag(agentFlag)
	cfg.AgentMode = det.Active
	clog.Debug().Str("agent", det.Name).Bool("active", det.Active).Msg("agent detection")

	var apiOpts []api.Option
	if cfg.BaseURL != "" {
		clog.Debug().Str("base_url", cfg.BaseURL).Msg("custom base URL")
		apiOpts = append(apiOpts, api.WithBaseURL(cfg.BaseURL))
	}

	return preRunState{cfg: cfg, det: det, apiOpts: apiOpts}, nil
}

// resolveAndStore resolves the API token, creates the client and
// stores config, client, agent detection and user email on context.
func resolveAndStore(ctx context.Context, pf *pflag.FlagSet, state preRunState, flagToken string) (context.Context, error) {
	cfg, det, apiOpts := state.cfg, state.det, state.apiOpts
	flagTokenFile, _ := pf.GetString("token-file")
	resolvedToken, err := resolveToken(ctx, cfg, flagToken, flagTokenFile)
	if err != nil {
		return ctx, fmt.Errorf("resolving credentials: %w", err)
	}
	cfg.Token = resolvedToken
	clog.Debug().Str("source", tokenSource(flagToken, flagTokenFile, cfg)).Msg("token resolved")

	if err := cfg.Validate(); err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, configKey, cfg)
	ctx = context.WithValue(ctx, agentKey, det)

	if cfg.Token != "" {
		client := api.NewClient(cfg.Token, apiOpts...)
		ctx = context.WithValue(ctx, clientKey, client)
		ctx = context.WithValue(ctx, resolverKey, resolve.New(client))

		// Resolve the token owner's email for write operations (From header).
		// Best-effort: a failure here must not block startup.
		if u, err := client.GetCurrentUser(ctx); err == nil {
			clog.Debug().Str("email", u.Email).Msg("resolved token owner")
			ctx = context.WithValue(ctx, userEmailKey, u.Email)
		}
	}

	return ctx, nil
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringP("token", "t", "", "PagerDuty API token (not recommended - visible in process list)")
	pf.String("token-file", "", "Read API token from file (preferred over --token)")
	pf.StringP("team", "T", "", "Team name or ID filter (overrides PDC_TEAM)")
	pf.StringP("service", "S", "", "Service name or ID filter (overrides config)")
	pf.StringP("format", "f", "table", `Output format: "table" or "json"`)
	pf.BoolP("interactive", "i", false, "Launch interactive TUI dashboard")
	pf.Bool("agent", false, "Force agent mode (structured JSON output)")
	pf.StringP("config", "c", "", "Config file path (default: $XDG_CONFIG_HOME/pagerduty-client/config.toml)")
	pf.BoolP("debug", "d", false, "Enable debug output")
	pf.String("color", "auto", `Colour mode: "auto", "always", or "never"`)

	// Flag metadata for themed help rendering.
	clib.Extend(pf.Lookup("token"), clib.FlagExtra{
		Group:       "Connection",
		Placeholder: "TOKEN",
		Terse:       "API token (not recommended)",
	})
	clib.Extend(pf.Lookup("token-file"), clib.FlagExtra{
		Group:       "Connection",
		Placeholder: "PATH",
		Hint:        "file",
		Terse:       "token file path",
	})
	clib.Extend(pf.Lookup("team"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "NAME|ID",
		Complete:    "predictor=team",
		Terse:       "team filter",
	})
	clib.Extend(pf.Lookup("service"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "NAME|ID",
		Complete:    "predictor=service",
		Terse:       "service filter",
	})
	clib.Extend(pf.Lookup("format"), clib.FlagExtra{
		Group:       "Output",
		Enum:        []string{"table", "json"},
		EnumTerse:   []string{"table output", "JSON output"},
		EnumDefault: "table",
		Terse:       "output format",
	})
	clib.Extend(pf.Lookup("config"), clib.FlagExtra{
		Group:       "Connection",
		Placeholder: "PATH",
		Hint:        "file",
		Terse:       "config file path",
	})
	clib.Extend(pf.Lookup("interactive"), clib.FlagExtra{
		Group: "Output",
		Terse: "launch interactive TUI",
	})
	clib.Extend(pf.Lookup("agent"), clib.FlagExtra{
		Group: "Output",
		Terse: "force agent mode",
	})
	clib.Extend(pf.Lookup("debug"), clib.FlagExtra{
		Group: "Output",
		Terse: "debug output",
	})
	clib.Extend(pf.Lookup("color"), clib.FlagExtra{
		Group:       "Output",
		Enum:        []string{"auto", "always", "never"},
		EnumTerse:   []string{"detect terminal", "force colour", "no colour"},
		EnumDefault: "auto",
		Terse:       "colour mode",
	})

	rootCmd.MarkFlagsMutuallyExclusive("token", "token-file")

	// Command groups for themed help.
	rootCmd.AddGroup(
		&cobra.Group{ID: "resources", Title: "Resources"},
		&cobra.Group{ID: "config", Title: "Configuration"},
	)

	// Enable PDC_THEME env var for theme selection.
	theme.SetEnvPrefix("PDC")

	// Themed help rendering.
	th := theme.Default().With(
		theme.WithEnumStyle(theme.EnumStyleHighlightBoth),
		theme.WithHelpRepeatEllipsisEnabled(true),
	)
	renderer := help.NewRenderer(th)
	rootCmd.SetHelpFunc(clib.HelpFunc(renderer, clib.SectionsWithOptions(clib.WithSubcommandOptional())))
}

// ConfigFromContext retrieves the loaded Config from the command context.
func ConfigFromContext(cmd *cobra.Command) *config.Config {
	v, _ := cmd.Context().Value(configKey).(*config.Config)
	return v
}

// ClientFromContext retrieves the API Client from the command context.
func ClientFromContext(cmd *cobra.Command) *api.Client {
	v, _ := cmd.Context().Value(clientKey).(*api.Client)
	return v
}

// ResolverFromContext retrieves the Resolver from the command context.
func ResolverFromContext(cmd *cobra.Command) *resolve.Resolver {
	v, _ := cmd.Context().Value(resolverKey).(*resolve.Resolver)
	return v
}

// AgentFromContext retrieves the agent DetectionResult from the command context.
func AgentFromContext(cmd *cobra.Command) agent.DetectionResult {
	v, _ := cmd.Context().Value(agentKey).(agent.DetectionResult)
	return v
}

// UserEmailFromContext retrieves the current user's email from the command context.
// Returns an empty string if the email was not resolved (e.g. no token or lookup failed).
func UserEmailFromContext(cmd *cobra.Command) string {
	v, _ := cmd.Context().Value(userEmailKey).(string)
	return v
}

// UpdateResultFromContext retrieves the cached update check result
// stored by PersistentPreRunE. Returns a zero-value CheckResult if
// the check was skipped or not yet run.
func UpdateResultFromContext(cmd *cobra.Command) update.CheckResult {
	v, _ := cmd.Context().Value(updateResultKey).(update.CheckResult)
	return v
}

// resolveToken determines the API token from available sources in precedence order:
//  1. flagToken   - from --token flag (already resolved by Cobra before PersistentPreRunE)
//  2. tokenFile   - from --token-file flag (mutually exclusive with --token)
//  3. PDC_TOKEN   - env var (always overrides credential_source)
//  4. keyring     - credential_source = "keyring": KeyringProvider.Provide
//  5. fallthrough - empty string; Validate() catches the missing token downstream
func resolveToken(ctx context.Context, cfg *config.Config, flagToken, tokenFile string) (string, error) {
	if flagToken != "" && tokenFile != "" {
		return "", errors.New("--token and --token-file are mutually exclusive")
	}
	if flagToken != "" {
		return flagToken, nil
	}
	if tokenFile != "" {
		b, err := os.ReadFile(tokenFile) //nolint:gosec // user-provided flag path, not attacker-controlled
		if err != nil {
			return "", fmt.Errorf("reading token file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	if v := os.Getenv("PDC_TOKEN"); v != "" {
		return v, nil
	}
	switch cfg.CredentialSource {
	case credential.SourceKeyring:
		p := credential.KeyringProvider{}
		token, err := p.Provide(ctx)
		if errors.Is(err, credential.ErrNotFound) {
			return "", errors.New("no token found in OS keyring; run \"pdc config init\" to configure credentials")
		}
		return token, err
	case "":
		return "", nil
	default:
		return "", fmt.Errorf("unknown credential_source %q in config", cfg.CredentialSource)
	}
}

func tokenSource(flagToken, tokenFile string, cfg *config.Config) string {
	if flagToken != "" {
		return "flag"
	}
	if tokenFile != "" {
		return "file"
	}
	if os.Getenv("PDC_TOKEN") != "" {
		return "env"
	}
	if cfg.CredentialSource != "" {
		return string(cfg.CredentialSource)
	}
	return "none"
}
