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

	var resolvedToken string

	// If PDC_TOKEN is set, validate it and offer keyring as a fallback.
	if envToken := os.Getenv("PDC_TOKEN"); envToken != "" {
		suffix := envToken
		if len(envToken) > 3 {
			suffix = envToken[len(envToken)-3:]
		}
		clog.Info().Str("token", "..."+suffix).Msg("PDC_TOKEN detected in environment")

		email, err := validateToken(envToken, apiOpts)
		if err != nil {
			clog.Info().Err(err).Msg("token validation failed - continuing with setup")
		} else {
			clog.Info().Msg("Token verified")
			resolvedToken = envToken
			ic.defaultEmail = email
		}

		var setupKeyring bool
		if err := huh.NewConfirm().
			Title("Also store a token in the OS keyring? (fallback when PDC_TOKEN is not set)").
			Value(&setupKeyring).
			Run(); err != nil {
			return err
		}
		if setupKeyring {
			if err := setupKeyringToken(&ic, &resolvedToken, apiOpts); err != nil {
				return err
			}
		}
	} else {
		// No env token - prompt for keyring storage.
		if err := setupKeyringToken(&ic, &resolvedToken, apiOpts); err != nil {
			return err
		}
	}

	// Prompt for email if we couldn't auto-detect it (account-level key
	// or token validation failed).
	if ic.defaultEmail == "" {
		var email string
		if err := huh.NewInput().
			Title("PagerDuty email address (optional, used for write operations like ack/resolve)").
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
	}

	// Default team/service selection (only when we have a validated token).
	if resolvedToken != "" {
		if err := withInitTimeout(func(ctx context.Context) error {
			return runTeamSelection(ctx, resolvedToken, &ic, apiOpts)
		}); err != nil {
			clog.Warn().Err(err).Msg("team selection failed - skipping")
		}
		if err := withInitTimeout(func(ctx context.Context) error {
			return runServiceSelection(ctx, resolvedToken, &ic, apiOpts)
		}); err != nil {
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

// validateToken checks a PagerDuty API token and returns the owner's email
// if available. It tries /users/me first (works with user tokens). If that
// returns 400 (account-level API key), it falls back to /abilities which
// works with both token types.
//
// Returns ("", nil) when the token is valid but no email could be resolved
// (account-level key). Returns ("", err) when the token is invalid.
func validateToken(token string, opts []api.Option) (string, error) {
	var email string
	err := withInitTimeout(func(ctx context.Context) error {
		client := api.NewClient(token, opts...)

		user, userErr := client.GetCurrentUser(ctx)
		if userErr == nil {
			email = user.Email
			return nil
		}

		if apiErr, ok := errors.AsType[*api.APIError](userErr); ok {
			switch apiErr.StatusCode {
			case 400:
				// Account-level API key - /users/me doesn't apply.
				// Fall back to /abilities to confirm the key is valid.
				clog.Debug().Msg("account-level API key detected, validating via /abilities")
				if _, abErr := client.ListAbilities(ctx); abErr != nil {
					return fmt.Errorf("token validation failed: %w", abErr)
				}
				return nil
			case 401, 403:
				return fmt.Errorf("authentication failed (HTTP %d) - check your token is a valid REST API key, not an Events/integration key", apiErr.StatusCode)
			default:
				return fmt.Errorf("PagerDuty API returned HTTP %d - check your token and try again", apiErr.StatusCode)
			}
		}
		return fmt.Errorf("could not reach PagerDuty API: %w", userErr)
	})
	return email, err
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

// isPlanLimitation returns true if the error is an HTTP 402 from PagerDuty,
// which indicates the account's plan does not include the requested feature.
func isPlanLimitation(err error) bool {
	apiErr, ok := errors.AsType[*api.APIError](err)
	return ok && apiErr.StatusCode == 402
}

// initCallTimeout is the per-call deadline for API requests made during
// the init wizard. Each call gets its own context so that time spent on
// interactive prompts does not eat into the deadline.
const initCallTimeout = 30 * time.Second

// withInitTimeout runs fn with a fresh 30-second context.
func withInitTimeout(fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), initCallTimeout)
	defer cancel()
	return fn(ctx)
}

// setupKeyringToken handles the keyring token flow: check existing, prompt for new, validate, store.
func setupKeyringToken(ic *initConfig, resolvedToken *string, apiOpts []api.Option) error {
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

		if email, err := validateToken(rawToken, apiOpts); err != nil {
			clog.Info().Err(err).Msg("token validation failed - storing anyway")
		} else {
			clog.Info().Msg("Token verified")
			if ic.defaultEmail == "" {
				ic.defaultEmail = email
			}
		}

		if err := keyring.Set(credential.ServiceName, credential.AccountName, rawToken); err != nil {
			clog.Warn().Err(err).Msg("could not store token in OS keyring - use PDC_TOKEN or --token instead")
		}
		*resolvedToken = rawToken
	} else {
		if email, err := validateToken(*resolvedToken, apiOpts); err != nil {
			clog.Info().Err(err).Msg("token validation failed - continuing with setup")
		} else {
			clog.Info().Msg("Token verified")
			if ic.defaultEmail == "" {
				ic.defaultEmail = email
			}
		}
	}

	ic.credentialSource = credential.SourceKeyring
	return nil
}

// runTeamSelection runs the optional team picker step.
func runTeamSelection(ctx context.Context, token string, ic *initConfig, opts []api.Option) error {
	c := api.NewClient(token, opts...)
	teams, err := c.ListTeams(ctx, api.ListTeamsOpts{})
	if err != nil {
		if isPlanLimitation(err) {
			clog.Info().Msg("Teams not available on this plan - skipping")
			return nil
		}
		return err
	}
	if len(teams) == 0 {
		clog.Info().Msg("No teams found - skipping default team")
		return nil
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
	if err != nil {
		if isPlanLimitation(err) {
			clog.Info().Msg("Services not available on this plan - skipping")
			return nil
		}
		return err
	}
	if len(services) == 0 {
		clog.Info().Msg("No services found - skipping default service")
		return nil
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
