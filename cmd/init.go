package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/PagerDuty/go-pagerduty"
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
	tabs             []string
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage pdc configuration",
	Long:    "Subcommands for managing the pdc configuration file.",
	GroupID: "config",
	// Suppress the root PersistentPreRunE, which requires a valid token.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive first-run setup",
	Long: `pdc config init guides you through creating ~/.config/pagerduty-client/config.toml.

It configures how pdc obtains your PagerDuty API token:
  - OS Keyring (macOS Keychain / Windows Credential Manager / Linux Secret Service)

If the PDC_TOKEN environment variable is already set, init validates the
token and skips keyring setup.

The token is never written to config.toml.`,
	Example: `# Run the setup wizard
$ pdc config init`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	det := agent.Detect()
	if det.Active || !terminal.Is(os.Stdout) {
		return errors.New("pdc config init requires an interactive terminal")
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
			Title("Overwrite existing config?").
			Description(cfgPath).
			Affirmative("Overwrite").
			Negative("Keep existing").
			Value(&overwrite).
			Run(); err != nil {
			return err
		}
		if !overwrite {
			clog.Info().Msg("aborted - existing config unchanged")
			return nil
		}
	}

	// Phase 1: resolve and validate token.
	ic, resolvedToken, err := resolveInitToken(apiOpts)
	if err != nil {
		return err
	}

	// Phase 2: preferences form (email, team, service, interactive).
	if err := runInitPreferencesForm(resolvedToken, &ic, apiOpts); err != nil {
		return err
	}

	// Phase 3: write config.
	configDir := filepath.Dir(cfgPath)
	if err := writeInitConfig(configDir, ic); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	clog.Info().Path("config", cfgPath).Msg("config written")
	return nil
}

func resolveInitToken(apiOpts []api.Option) (initConfig, string, error) {
	var ic initConfig
	envToken := os.Getenv("PDC_TOKEN")
	wsl := isWSL()

	if wsl {
		if envToken == "" {
			return ic, "", errors.New(
				"OS keyring is not supported under WSL - set PDC_TOKEN and re-run init")
		}
		clog.Warn().Str("reason", "unsupported on WSL").Msg("keyring skipped")
	}

	if envToken != "" {
		email, err := validateWithSpinner(envToken, apiOpts)
		switch {
		case isAuthErr(err):
			return ic, "", err
		case err != nil:
			clog.Warn().Err(err).Msg("token validation failed - continuing with setup")
		default:
			ic.defaultEmail = email
		}

		if !wsl {
			var setupKeyring bool
			if err := huh.NewConfirm().
				Title("Also store a token in the OS keyring? (fallback when PDC_TOKEN is not set)").
				Value(&setupKeyring).
				Run(); err != nil {
				return ic, "", err
			}
			if setupKeyring {
				if err := maybeStoreKeyringToken(envToken); err != nil {
					clog.Warn().Err(err).
						Msg("could not store token in OS keyring")
				} else {
					ic.credentialSource = credential.SourceKeyring
				}
			}
		}
		return ic, envToken, nil
	}

	// No env token - prompt for keyring storage.
	token, err := promptAndStoreToken(&ic, apiOpts)
	if err != nil {
		return ic, "", err
	}
	return ic, token, nil
}

func runInitPreferencesForm(token string, ic *initConfig, apiOpts []api.Option) error {
	var teams []pagerduty.Team
	var services []pagerduty.Service

	if token != "" {
		if err := withInitTimeout(func(ctx context.Context) error {
			return spinner.New().
				Title("Fetching account data...").
				ActionWithErr(func(sCtx context.Context) error {
					client := api.NewClient(token, apiOpts...)

					var wg sync.WaitGroup
					var teamErr, svcErr error

					wg.Go(func() {
						t, err := client.ListTeams(sCtx, api.ListTeamsOpts{})
						if err != nil && !isPlanLimitation(err) {
							teamErr = err
							return
						}
						teams = t
					})
					wg.Go(func() {
						s, err := client.ListServices(sCtx, api.ListServicesOpts{})
						if err != nil && !isPlanLimitation(err) {
							svcErr = err
							return
						}
						services = s
					})
					wg.Wait()

					return errors.Join(teamErr, svcErr)
				}).
				Context(ctx).
				Run()
		}); err != nil {
			clog.Warn().Err(err).Msg("could not fetch account data - skipping defaults")
		}
	}

	// Build form with conditional groups.
	var (
		email       string
		teamID      string
		serviceID   string
		interactive bool
	)

	teamOpts := []huh.Option[string]{huh.NewOption("No default", "")}
	for _, t := range teams {
		teamOpts = append(teamOpts, huh.NewOption(t.Name, t.ID))
	}

	serviceOpts := []huh.Option[string]{huh.NewOption("No default", "")}
	for _, s := range services {
		serviceOpts = append(serviceOpts, huh.NewOption(s.Name, s.ID))
	}

	emailGroup := huh.NewGroup(
		huh.NewInput().
			Title("PagerDuty login email").
			Value(&email).
			Validate(func(s string) error {
				if s == "" {
					return nil // optional
				}
				if _, err := mail.ParseAddress(s); err != nil {
					return errors.New("invalid email address")
				}
				return nil
			}),
	).WithHide(ic.defaultEmail != "")

	teamGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Default team filter").
			Description("Used for incident and on-call views").
			Options(teamOpts...).
			Value(&teamID),
	).WithHide(len(teams) == 0)

	serviceGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Default service filter").
			Description("Used for incident views").
			Options(serviceOpts...).
			Value(&serviceID),
	).WithHide(len(services) == 0)

	tabs := config.DefaultTabs
	tabOpts := []huh.Option[string]{
		huh.NewOption("Incidents", "incidents"),
		huh.NewOption("Escalation Policies", "escalation-policies"),
		huh.NewOption("Services", "services"),
		huh.NewOption("Teams", "teams"),
	}

	tabsGroup := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("TUI tabs").
			Description("Select which tabs to show in the dashboard").
			Height(8).
			Options(tabOpts...).
			Value(&tabs),
	)

	prefsGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable interactive mode by default?").
			Description("Launches TUI dashboard when running pdc").
			Value(&interactive),
	)

	if err := huh.NewForm(emailGroup, teamGroup, serviceGroup, tabsGroup, prefsGroup).
		Run(); err != nil {
		return err
	}

	// Apply form results.
	if email != "" {
		ic.defaultEmail = email
	}
	ic.defaultTeamID = teamID
	ic.defaultServiceID = serviceID
	ic.interactive = interactive
	ic.tabs = tabs

	if ic.defaultEmail != "" {
		clog.Debug().Str("email", ic.defaultEmail).Msg("email set")
	}
	if ic.defaultTeamID != "" {
		clog.Debug().Str("team", ic.defaultTeamID).Msg("default team set")
	}
	if ic.defaultServiceID != "" {
		clog.Debug().Str("service", ic.defaultServiceID).Msg("default service set")
	}
	clog.Debug().Bool("enabled", ic.interactive).Msg("interactive mode")

	return nil
}

// authError indicates the token is definitively invalid (not a transient failure).
type authError struct {
	err error
}

func (e *authError) Error() string { return e.err.Error() }
func (e *authError) Unwrap() error { return e.err }

func isAuthErr(err error) bool {
	_, ok := errors.AsType[*authError](err)
	return ok
}

// validateTokenAPI checks a PagerDuty API token and returns the owner's email
// if available. It tries /users/me first (works with user tokens). If that
// returns 400 (account-level API key), it falls back to /abilities which
// works with both token types.
//
// Returns ("", nil) when the token is valid but no email could be resolved
// (account-level key). Returns ("", err) when the token is invalid.
func validateTokenAPI(ctx context.Context, token string, opts []api.Option) (string, error) {
	client := api.NewClient(token, opts...)

	user, userErr := client.GetCurrentUser(ctx)
	if userErr == nil {
		return user.Email, nil
	}

	if apiErr, ok := errors.AsType[*api.APIError](userErr); ok {
		switch apiErr.StatusCode {
		case 400:
			// Account-level API key - /users/me doesn't apply.
			// Fall back to /abilities to confirm the key is valid.
			clog.Debug().Msg("account-level API key detected, validating via /abilities")
			if _, abErr := client.ListAbilities(ctx); abErr != nil {
				return "", fmt.Errorf("token validation failed: %w", abErr)
			}
			return "", nil
		case 401, 403:
			return "", &authError{err: fmt.Errorf(
				"authentication failed (HTTP %d) - check your token is a valid REST API key, not an Events/integration key",
				apiErr.StatusCode)}
		default:
			return "", fmt.Errorf(
				"PagerDuty API returned HTTP %d - check your token and try again",
				apiErr.StatusCode)
		}
	}
	return "", fmt.Errorf("could not reach PagerDuty API: %w", userErr)
}

func validateWithSpinner(token string, opts []api.Option) (string, error) {
	suffix := token
	if len(token) > 3 {
		suffix = token[len(token)-3:]
	}

	var email string
	err := withInitTimeout(func(ctx context.Context) error {
		return spinner.New().
			Title("Validating token ..." + suffix).
			ActionWithErr(func(sCtx context.Context) error {
				var vErr error
				email, vErr = validateTokenAPI(sCtx, token, opts)
				return vErr
			}).
			Context(ctx).
			Run()
	})
	return email, err
}

// maybeStoreKeyringToken stores a token in the OS keyring, prompting
// before overwriting an existing entry.
func maybeStoreKeyringToken(token string) error {
	if existing, err := keyring.Get(credential.ServiceName, credential.AccountName); err == nil && existing != "" {
		var overwrite bool
		if err := huh.NewConfirm().
			Title("A token is already stored in the OS keyring. Overwrite it?").
			Value(&overwrite).
			Run(); err != nil {
			return err
		}
		if !overwrite {
			return nil
		}
	}
	return keyring.Set(credential.ServiceName, credential.AccountName, token)
}

func promptAndStoreToken(ic *initConfig, apiOpts []api.Option) (string, error) {
	// Check for existing keyring token.
	if existing, err := keyring.Get(credential.ServiceName, credential.AccountName); err == nil && existing != "" {
		var overwrite bool
		if err := huh.NewConfirm().
			Title("A token is already stored in the OS keyring. Overwrite it?").
			Value(&overwrite).
			Run(); err != nil {
			return "", err
		}
		if !overwrite {
			email, err := validateWithSpinner(existing, apiOpts)
			switch {
			case isAuthErr(err):
				clog.Warn().Err(err).Msg("stored keyring token is invalid - enter a new one")
			case err != nil:
				clog.Warn().Err(err).Msg("token validation failed - continuing with setup")
				ic.credentialSource = credential.SourceKeyring
				return existing, nil
			default:
				ic.defaultEmail = email
				ic.credentialSource = credential.SourceKeyring
				return existing, nil
			}
		}
	}

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
		return "", err
	}

	email, err := validateWithSpinner(rawToken, apiOpts)
	switch {
	case isAuthErr(err):
		return "", err
	case err != nil:
		clog.Warn().Err(err).Msg("token validation failed - storing anyway")
	default:
		ic.defaultEmail = email
	}

	if err := keyring.Set(credential.ServiceName, credential.AccountName, rawToken); err != nil {
		clog.Warn().Err(err).Msg("could not store token in OS keyring")
	} else {
		ic.credentialSource = credential.SourceKeyring
	}
	return rawToken, nil
}

// writeInitConfig writes config.toml to configDir with mode 0600.
func writeInitConfig(configDir string, ic initConfig) error {
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# pdc configuration - generated by pdc config init\n\n")

	switch ic.credentialSource {
	case credential.SourceKeyring:
		_, _ = fmt.Fprintf(&sb, "credential_source = %q\n", ic.credentialSource)
		sb.WriteString("# Note: PDC_TOKEN env var takes precedence over keyring when set\n")
	default:
		sb.WriteString("# No credential source configured - pdc reads PDC_TOKEN from the environment\n")
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

	if len(ic.tabs) > 0 {
		sb.WriteString("\n[tui]\n")
		quoted := make([]string, len(ic.tabs))
		for i, t := range ic.tabs {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		_, _ = fmt.Fprintf(&sb, "tabs = [%s]\n", strings.Join(quoted, ", "))
	}

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

// isWSL reports whether the process is running under Windows Subsystem for Linux.
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
