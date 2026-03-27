// Package cmd contains Cobra command definitions for the pdc CLI.
// Each file wires flags and subcommands; all business logic lives in internal/.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	"github.com/matcra587/pagerduty-client/internal/tui"
	"github.com/spf13/cobra"
)

// contextKey is a package-local type for context value keys to avoid collisions.
type contextKey string

const (
	configKey    contextKey = "config"
	clientKey    contextKey = "client"
	agentKey     contextKey = "agent"
	userEmailKey contextKey = "userEmail"
)

// comp holds the hidden completion flags added by clib.NewCompletion.
var comp *clib.Completion

// rootCmd is the base command for pdc.
var rootCmd = &cobra.Command{
	Use:           "pdc",
	Short:         "PagerDuty CLI",
	Long:          "AI-agent-ready CLI for PagerDuty. Every command produces structured, self-describing output.",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		pf := cmd.Root().PersistentFlags()

		// Collect flag overrides (token resolved separately after config load).
		var opts []config.Option

		if cfgPath, _ := pf.GetString("config"); cfgPath != "" {
			opts = append(opts, config.WithPath(cfgPath))
		}
		if team, _ := pf.GetString("team"); team != "" {
			opts = append(opts, config.WithTeam(team))
		}

		cfg, err := config.Load(opts...)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Apply remaining flag overrides directly to cfg.
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
				return fmt.Errorf("invalid colour mode %q: must be \"auto\", \"always\" or \"never\"", colorMode)
			}
		}

		// Agent mode detection.
		agentFlag, _ := pf.GetBool("agent")
		det := agent.DetectWithFlag(agentFlag)
		cfg.AgentMode = det.Active
		clog.Debug().Str("agent", det.Name).Bool("active", det.Active).Msg("agent detection")

		// API client options.
		var apiOpts []api.Option
		if cfg.BaseURL != "" {
			clog.Debug().Str("base_url", cfg.BaseURL).Msg("custom base URL")
			apiOpts = append(apiOpts, api.WithBaseURL(cfg.BaseURL))
		}

		// Handle shell completion requests before token resolution so
		// --install-completion and --print-completion work without a token.
		// Dynamic completion (team/service names) is best-effort: it uses
		// whatever token is available from env/keyring.
		flagToken, _ := pf.GetString("token")
		gen := complete.NewGenerator("pdc").FromFlags(clib.FlagMeta(cmd.Root()))
		gen.Subs = clib.Subcommands(cmd.Root())
		handled, err := comp.Handle(gen, completionHandler(flagToken, apiOpts...))
		if err != nil {
			return err
		}
		if handled {
			os.Exit(0) //nolint:revive // completion handler must exit after handling
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Resolve token from flag, env, or credential store.
		flagTokenFile, _ := pf.GetString("token-file")
		resolvedToken, err := resolveToken(ctx, cfg, flagToken, flagTokenFile)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}
		cfg.Token = resolvedToken
		clog.Debug().Str("source", tokenSource(flagToken, flagTokenFile, cfg)).Msg("token resolved")

		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.NewClient(cfg.Token, apiOpts...)

		// Store values on context.
		ctx = context.WithValue(ctx, configKey, cfg)
		ctx = context.WithValue(ctx, clientKey, client)
		ctx = context.WithValue(ctx, agentKey, det)

		// Resolve the token owner's email for write operations (From header).
		// Best-effort: a failure here must not block startup.
		if cfg.Token != "" {
			if u, err := client.GetCurrentUser(ctx); err == nil {
				clog.Debug().Str("email", u.Email).Msg("resolved token owner")
				ctx = context.WithValue(ctx, userEmailKey, u.Email)
			}
		}

		cmd.SetContext(ctx)

		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		det := AgentFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		isTTY := terminal.Is(os.Stdout)

		if !det.Active && isTTY && cfg.Interactive {
			client := ClientFromContext(cmd)
			email := UserEmailFromContext(cmd)
			app := tui.New(cmd.Context(), client, cfg, email)
			p := tea.NewProgram(app, tea.WithContext(cmd.Context()))
			_, err := p.Run()
			return err
		}

		return cmd.Help()
	},
}

// Execute runs the root command and returns any error.
func Execute() error {
	setup()
	err := rootCmd.Execute()
	if err != nil {
		clog.Error().Err(err).Send()
	}
	return err
}

// setup wires completion flags after all subcommands are registered.
// Called from Execute to guarantee init() across all cmd/ files has run.
func setup() {
	comp = clib.NewCompletion(rootCmd)
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringP("token", "t", "", "PagerDuty API token (not recommended - visible in process list)")
	pf.String("token-file", "", "Read API token from file (preferred over --token)")
	pf.StringP("team", "T", "", "Team name or ID filter (overrides PDC_TEAM)")
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
	clib.Extend(pf.Lookup("format"), clib.FlagExtra{
		Group:       "Output",
		Enum:        []string{"table", "json"},
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
		EnumDefault: "auto",
		Terse:       "colour mode",
	})

	rootCmd.MarkFlagsMutuallyExclusive("token", "token-file")

	// Command groups for themed help.
	rootCmd.AddGroup(
		&cobra.Group{ID: "resources", Title: "Resources"},
		&cobra.Group{ID: "config", Title: "Configuration"},
	)

	// Themed help rendering.
	th := theme.New(
		theme.WithEnumStyle(theme.EnumStyleHighlightBoth),
		theme.WithHelpRepeatEllipsisEnabled(true),
	)
	renderer := help.NewRenderer(th)
	rootCmd.SetHelpFunc(clib.HelpFunc(renderer, clib.Sections,
		help.WithHelpFlags("Print help", "Print help with examples"),
		help.WithLongHelp(os.Args, buildExamplesSection()),
	))
}

func buildExamplesSection() help.Section {
	return help.Section{
		Title: "Examples",
		Content: []help.Content{
			help.Examples{
				{Comment: "Launch the TUI dashboard", Command: "pdc -i"},
				{Comment: "List triggered incidents as JSON", Command: "pdc incident list --format json"},
				{Comment: "Acknowledge an incident", Command: "pdc incident ack P000001"},
				{Comment: "Show who is on call", Command: "pdc oncall"},
			},
		},
	}
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
			return "", errors.New("no token found in OS keyring; run \"pdc init\" to configure credentials")
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

// completionHandler returns a handler for dynamic shell completion requests.
// It queries the PagerDuty API for resources matching the requested completion
// kind (e.g. "team", "service") and prints matching names to stdout.
func completionHandler(token string, opts ...api.Option) complete.Handler {
	return func(_, kind string, _ []string) {
		if token == "" {
			return
		}
		client := api.NewClient(token, opts...)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var names []string
		switch kind {
		case "team":
			teams, err := client.ListTeams(ctx, api.ListTeamsOpts{})
			if err != nil {
				return
			}
			for _, t := range teams {
				names = append(names, t.Name)
			}
		case "service":
			services, err := client.ListServices(ctx, api.ListServicesOpts{})
			if err != nil {
				return
			}
			for _, s := range services {
				names = append(names, s.Name)
			}
		default:
			return
		}
		for _, n := range names {
			_, _ = fmt.Println(n)
		}
	}
}
