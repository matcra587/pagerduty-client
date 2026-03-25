package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// initConfig collects values entered during the init wizard.
type initConfig struct {
	credentialSource credential.Source
	defaultEmail     string
	defaultTeamID    string
	defaultServiceID string
	interactive      bool
}

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Interactive first-run setup - creates the pdc config file",
	GroupID: "config",
	Long: `pdc init guides you through creating ~/.config/pagerduty-client/config.toml.

It configures how pdc obtains your PagerDuty API token:
  - OS Keyring (macOS Keychain / Windows Credential Manager / Linux Secret Service)

If the PDC_TOKEN environment variable is already set, init validates the
token and skips keyring setup.

The token is never written to config.toml.`,
	// Suppress the root PersistentPreRunE, which requires a valid token.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
	RunE:              runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	det := agent.Detect()
	if det.Active || !terminal.Is(os.Stdout) {
		return errors.New("pdc init requires an interactive terminal")
	}

	cfgPath := config.DefaultConfigPath()
	existingCfg, _ := config.Load(config.WithPath(cfgPath))

	var apiOpts []api.Option
	if existingCfg.BaseURL != "" {
		apiOpts = append(apiOpts, api.WithBaseURL(existingCfg.BaseURL))
	}

	// Confirm overwrite if config already exists.
	if _, err := os.Stat(cfgPath); err == nil {
		var overwrite bool
		if err := huh.NewConfirm().
			Title("A config file already exists at " + cfgPath + ". Overwrite it?").
			Value(&overwrite).
			Run(); err != nil {
			return err
		}
		if !overwrite {
			clog.Info().Msg("Aborted - existing config unchanged")
			return nil
		}
	}

	ic := initConfig{}

	// Optional email for write operations (ack, resolve, etc.).
	var email string
	if err := huh.NewInput().
		Title("PagerDuty email address (optional, used for write operations)").
		Description("Leave blank to look up automatically from the API token.").
		Value(&email).
		Run(); err != nil {
		return err
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			clog.Warn().Str("email", email).Msg("invalid email - skipping")
		} else {
			ic.defaultEmail = email
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resolvedToken string

	// If PDC_TOKEN is set, validate it and offer keyring as a fallback.
	if envToken := os.Getenv("PDC_TOKEN"); envToken != "" {
		suffix := envToken
		if len(envToken) > 3 {
			suffix = envToken[len(envToken)-3:]
		}
		clog.Info().Str("token", "..."+suffix).Msg("PDC_TOKEN detected in environment")

		if err := validateTokenViaAPI(ctx, envToken, apiOpts); err != nil {
			clog.Info().Err(err).Msg("token validation failed - continuing with setup")
		} else {
			clog.Info().Msg("Token verified")
			resolvedToken = envToken
		}

		var setupKeyring bool
		if err := huh.NewConfirm().
			Title("Also store a token in the OS keyring? (fallback when PDC_TOKEN is not set)").
			Value(&setupKeyring).
			Run(); err != nil {
			return err
		}
		if setupKeyring {
			if err := setupKeyringToken(ctx, &ic, &resolvedToken, apiOpts); err != nil {
				return err
			}
		}
	} else {
		// No env token - prompt for keyring storage.
		if err := setupKeyringToken(ctx, &ic, &resolvedToken, apiOpts); err != nil {
			return err
		}
	}

	// Default team/service selection (only when we have a validated token).
	if resolvedToken != "" {
		if err := runTeamSelection(ctx, resolvedToken, &ic, apiOpts); err != nil {
			clog.Warn().Err(err).Msg("team selection failed - skipping")
		}
		if err := runServiceSelection(ctx, resolvedToken, &ic, apiOpts); err != nil {
			clog.Warn().Err(err).Msg("service selection failed - skipping")
		}
	}

	var enableInteractive bool
	if err := huh.NewConfirm().
		Title("Enable interactive mode by default? (launches TUI dashboard on pdc)").
		Value(&enableInteractive).
		Run(); err != nil {
		return err
	}
	ic.interactive = enableInteractive

	configDir := filepath.Dir(cfgPath)
	if err := writeInitConfig(configDir, ic); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	clog.Info().Path("config", cfgPath).Msg("Config written")
	return nil
}

// validateTokenViaAPI calls GET /users/me to confirm the token is valid.
func validateTokenViaAPI(ctx context.Context, token string, opts []api.Option) error {
	client := api.NewClient(token, opts...)
	_, err := client.GetCurrentUser(ctx)
	if err != nil {
		if apiErr, ok := errors.AsType[*api.APIError](err); ok {
			switch apiErr.StatusCode {
			case 400:
				return errors.New("bad request (HTTP 400) - this usually means the token is malformed or is the wrong type (use a REST API user token, not an Events API key)")
			case 401, 403:
				return fmt.Errorf("authentication failed (HTTP %d) - check your token is a valid API user token, not a service integration key", apiErr.StatusCode)
			default:
				return fmt.Errorf("PagerDuty API returned HTTP %d - check your token and try again", apiErr.StatusCode)
			}
		}
		return fmt.Errorf("could not reach PagerDuty API: %w", err)
	}
	return nil
}

// writeInitConfig writes config.toml to configDir with mode 0600.
func writeInitConfig(configDir string, ic initConfig) error {
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# pdc configuration - generated by pdc init\n\n")

	if ic.credentialSource != "" {
		_, _ = fmt.Fprintf(&sb, "credential_source = %q\n", ic.credentialSource)
		sb.WriteString("# Note: PDC_TOKEN env var takes precedence over keyring when set\n")
	}

	sb.WriteString("\n[defaults]\n")
	sb.WriteString("format           = \"table\"\n")
	sb.WriteString("refresh_interval = 30\n")
	if ic.defaultEmail != "" {
		_, _ = fmt.Fprintf(&sb, "email            = %q\n", ic.defaultEmail)
	}
	if ic.defaultTeamID != "" {
		_, _ = fmt.Fprintf(&sb, "team             = %q\n", ic.defaultTeamID)
	}
	if ic.defaultServiceID != "" {
		_, _ = fmt.Fprintf(&sb, "service          = %q\n", ic.defaultServiceID)
	}
	_, _ = fmt.Fprintf(&sb, "interactive      = %t\n", ic.interactive)

	path := filepath.Join(configDir, "config.toml")
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// setupKeyringToken handles the keyring token flow: check existing, prompt for new, validate, store.
func setupKeyringToken(ctx context.Context, ic *initConfig, resolvedToken *string, apiOpts []api.Option) error {
	if existing, err := keyring.Get(credential.ServiceName, credential.AccountName); err == nil && existing != "" {
		var overwriteKey bool
		if err := huh.NewConfirm().
			Title("A token is already stored in the OS keyring. Overwrite it?").
			Value(&overwriteKey).
			Run(); err != nil {
			return err
		}
		if !overwriteKey {
			clog.Info().Msg("Keeping existing keyring token")
			*resolvedToken = existing
		}
	}

	if *resolvedToken == "" {
		var rawToken string
		if err := huh.NewInput().
			Title("PagerDuty API token").
			EchoMode(huh.EchoModePassword).
			Value(&rawToken).
			Validate(func(s string) error {
				if s == "" {
					return errors.New("token is required")
				}
				return nil
			}).
			Run(); err != nil {
			return err
		}

		if err := validateTokenViaAPI(ctx, rawToken, apiOpts); err != nil {
			clog.Info().Err(err).Msg("token validation failed - storing anyway")
		} else {
			clog.Info().Msg("Token verified")
		}

		if err := keyring.Set(credential.ServiceName, credential.AccountName, rawToken); err != nil {
			clog.Warn().Err(err).Msg("could not store token in OS keyring - use PDC_TOKEN or --token instead")
		}
		*resolvedToken = rawToken
	} else if *resolvedToken != "" {
		if err := validateTokenViaAPI(ctx, *resolvedToken, apiOpts); err != nil {
			clog.Info().Err(err).Msg("token validation failed - continuing with setup")
		} else {
			clog.Info().Msg("Token verified")
		}
	}

	ic.credentialSource = credential.SourceKeyring
	return nil
}

// runTeamSelection runs the optional team picker step.
func runTeamSelection(ctx context.Context, token string, ic *initConfig, opts []api.Option) error {
	c := api.NewClient(token, opts...)
	teams, err := c.ListTeams(ctx, api.ListTeamsOpts{})
	if err != nil || len(teams) == 0 {
		if len(teams) == 0 {
			clog.Info().Msg("No teams found - skipping default team")
		}
		return err
	}

	options := []huh.Option[string]{huh.NewOption("No default", "")}
	for _, t := range teams {
		options = append(options, huh.NewOption(t.Name, t.ID))
	}

	return huh.NewSelect[string]().
		Title("Select a default team filter (used for incident and on-call views)").
		Options(options...).
		Value(&ic.defaultTeamID).
		Run()
}

// runServiceSelection runs the optional service picker step.
func runServiceSelection(ctx context.Context, token string, ic *initConfig, opts []api.Option) error {
	c := api.NewClient(token, opts...)
	services, err := c.ListServices(ctx, api.ListServicesOpts{})
	if err != nil || len(services) == 0 {
		if len(services) == 0 {
			clog.Info().Msg("No services found - skipping default service")
		}
		return err
	}

	options := []huh.Option[string]{huh.NewOption("No default", "")}
	for _, s := range services {
		options = append(options, huh.NewOption(s.Name, s.ID))
	}

	return huh.NewSelect[string]().
		Title("Select a default service filter (used for incident views)").
		Options(options...).
		Value(&ic.defaultServiceID).
		Run()
}
