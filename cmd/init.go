package cmd

import (
	"context"
	"errors"
	"fmt"
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
	defaultTeamID    string
	defaultServiceID string
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
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted - existing config unchanged.")
			return nil
		}
	}

	ic := initConfig{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resolvedToken string

	// If PDC_TOKEN is set, validate it and skip keyring setup.
	if envToken := os.Getenv("PDC_TOKEN"); envToken != "" {
		suffix := envToken
		if len(envToken) > 3 {
			suffix = envToken[len(envToken)-3:]
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PDC_TOKEN detected in environment (...%s)\n", suffix)

		if err := validateTokenViaAPI(ctx, envToken, apiOpts); err != nil {
			return fmt.Errorf("%w\nRun \"pdc init\" to try again", err)
		}
		clog.Info().Msg("Token verified")
		resolvedToken = envToken

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Using environment variable for authentication - no keyring needed.")
	} else {
		// No env token - prompt for keyring storage.
		if existing, err := keyring.Get(credential.ServiceName, credential.AccountName); err == nil && existing != "" {
			var overwriteKey bool
			if err := huh.NewConfirm().
				Title("A token is already stored in the OS keyring. Overwrite it?").
				Value(&overwriteKey).
				Run(); err != nil {
				return err
			}
			if !overwriteKey {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Keeping existing keyring token.")
				resolvedToken = existing
			}
		}

		if resolvedToken == "" {
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
				return fmt.Errorf("%w\nRun \"pdc init\" to try again", err)
			}
			clog.Info().Msg("Token verified")

			if err := keyring.Set(credential.ServiceName, credential.AccountName, rawToken); err != nil {
				return fmt.Errorf("failed to store token in OS keyring: %w", err)
			}
			resolvedToken = rawToken
		} else {
			// Validate existing keyring token.
			if err := validateTokenViaAPI(ctx, resolvedToken, apiOpts); err != nil {
				return fmt.Errorf("%w\nRun \"pdc init\" to try again", err)
			}
			clog.Info().Msg("Token verified")
		}

		ic.credentialSource = credential.SourceKeyring
	}

	// Default team/service selection (only when we have a validated token).
	if resolvedToken != "" {
		if err := runTeamSelection(ctx, cmd, resolvedToken, &ic, apiOpts); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: team selection failed (%v). Skipping.\n", err)
		}
		if err := runServiceSelection(ctx, cmd, resolvedToken, &ic, apiOpts); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: service selection failed (%v). Skipping.\n", err)
		}
	}

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
		if apiErr, ok := errors.AsType[*api.APIError](err); ok && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			return fmt.Errorf("invalid token: authentication failed (HTTP %d)", apiErr.StatusCode)
		}
		return fmt.Errorf("token validation failed: %w", err)
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
	}

	sb.WriteString("\n[defaults]\n")
	sb.WriteString("format           = \"table\"\n")
	sb.WriteString("refresh_interval = 30\n")
	if ic.defaultTeamID != "" {
		_, _ = fmt.Fprintf(&sb, "team             = %q\n", ic.defaultTeamID)
	}
	if ic.defaultServiceID != "" {
		_, _ = fmt.Fprintf(&sb, "service          = %q\n", ic.defaultServiceID)
	}

	path := filepath.Join(configDir, "config.toml")
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// runTeamSelection runs the optional team picker step.
func runTeamSelection(ctx context.Context, cmd *cobra.Command, token string, ic *initConfig, opts []api.Option) error {
	c := api.NewClient(token, opts...)
	teams, err := c.ListTeams(ctx, api.ListTeamsOpts{})
	if err != nil || len(teams) == 0 {
		if len(teams) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No teams found - skipping default team.")
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
func runServiceSelection(ctx context.Context, cmd *cobra.Command, token string, ic *initConfig, opts []api.Option) error {
	c := api.NewClient(token, opts...)
	services, err := c.ListServices(ctx, api.ListServicesOpts{})
	if err != nil || len(services) == 0 {
		if len(services) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No services found - skipping default service.")
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
