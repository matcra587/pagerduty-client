package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/BurntSushi/toml"
	"github.com/gechr/clib/theme"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/spf13/cobra"
)

// configKeys is the ordered list of valid configuration keys.
var configKeys = []string{
	"base_url",
	"credential_source",
	"defaults.format",
	"defaults.refresh_interval",
	"defaults.email",
	"defaults.team",
	"defaults.service",
	"defaults.interactive",
	"tui.theme",
	"tui.show_resolved",
	"tui.page_size",
	"tui.tabs",
}

// configDescriptions maps each key to a human-readable description.
var configDescriptions = map[string]string{ //nolint:gosec // G101 false positive - no credentials here
	"base_url":                  "PagerDuty API base URL",
	"credential_source":         "Credential backend (read-only)",
	"defaults.format":           "Output format (table/json)",
	"defaults.refresh_interval": "TUI refresh seconds",
	"defaults.email":            "Acting user email",
	"defaults.team":             "Default team filter (see: pdc team list)",
	"defaults.service":          "Default service filter (see: pdc service list)",
	"defaults.interactive":      "Launch TUI by default",
	"tui.theme":                 "TUI colour theme",
	"tui.show_resolved":         "Show resolved incidents",
	"tui.page_size":             "TUI page size",
	"tui.tabs":                  "TUI tab selection",
}

// envVarMapping maps config keys to their environment variable equivalents.
// Only keys with env var support are listed.
var envVarMapping = map[string]string{
	"base_url":             "PDC_BASE_URL",
	"defaults.email":       "PDC_EMAIL",
	"defaults.team":        "PDC_TEAM",
	"defaults.service":     "PDC_SERVICE",
	"defaults.format":      "PDC_FORMAT",
	"defaults.interactive": "PDC_INTERACTIVE",
}

// ------------------------------------------------------------------
// config list (alias: ls)
// ------------------------------------------------------------------

func configListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show current settings",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath := configPathFromFlags(cmd)
			cfg, err := config.Load(config.WithPath(cfgPath))
			if err != nil {
				return err
			}

			resolved := configToMap(cfg)
			fileSources := configSourcesFromFile(cfgPath)

			th := theme.Default()

			rows := make([][]string, 0, len(configKeys))
			for _, key := range configKeys {
				val := resolved[key]
				source := "default"
				if _, ok := fileSources[key]; ok {
					source = "file"
				}
				if envVar, ok := envVarMapping[key]; ok {
					if os.Getenv(envVar) != "" {
						source = envVar
					}
				}
				desc := configDescriptions[key]
				rows = append(rows, []string{key, val, source, desc})
			}

			headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
			keyStyle := th.Blue.Padding(0, 1)
			valueStyle := lipgloss.NewStyle().Padding(0, 1)
			dimStyle := th.Dim.Padding(0, 1)

			t := table.New().
				Border(lipgloss.HiddenBorder()).
				Headers("Key", "Value", "Source", "Description").
				StyleFunc(func(row, col int) lipgloss.Style {
					if row == table.HeaderRow {
						return headerStyle
					}
					switch col {
					case 0:
						return keyStyle
					case 2, 3:
						return dimStyle
					default:
						return valueStyle
					}
				}).
				Rows(rows...)

			fmt.Fprintln(cmd.OutOrStdout(), t.Render())
			return nil
		},
	}
}

// ------------------------------------------------------------------
// config get
// ------------------------------------------------------------------

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <key>",
		Short:             "Show a configuration value",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := configPathFromFlags(cmd)
			cfg, err := config.Load(config.WithPath(cfgPath))
			if err != nil {
				return err
			}

			m := configToMap(cfg)
			val, ok := m[args[0]]
			if !ok {
				return fmt.Errorf("unknown config key %s - run 'pdc config list' to see all keys", args[0])
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

// ------------------------------------------------------------------
// config set
// ------------------------------------------------------------------

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "set <key> <value>",
		Short:             "Add or update a setting",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(_ *cobra.Command, args []string) error {
			key, val := args[0], args[1]
			if !slices.Contains(configKeys, key) {
				return fmt.Errorf("unknown config key %s - valid keys: %s", key, strings.Join(configKeys, ", "))
			}
			if key == "credential_source" {
				return errors.New("credential_source is read-only - use 'pdc config init' to configure credentials")
			}
			typed, err := parseConfigValue(key, val)
			if err != nil {
				return err
			}
			cfgPath := config.DefaultConfigPath()
			return modifyConfigFile(cfgPath, key, typed)
		},
	}
}

// ------------------------------------------------------------------
// config unset (aliases: rm, remove)
// ------------------------------------------------------------------

func configUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "unset <key>",
		Aliases:           []string{"rm", "remove"},
		Short:             "Remove a setting",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeConfigKeys,
		RunE: func(_ *cobra.Command, args []string) error {
			if !slices.Contains(configKeys, args[0]) {
				return fmt.Errorf("unknown config key %s - valid keys: %s", args[0], strings.Join(configKeys, ", "))
			}
			cfgPath := config.DefaultConfigPath()
			return removeConfigKey(cfgPath, args[0])
		},
	}
}

// ------------------------------------------------------------------
// config path
// ------------------------------------------------------------------

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.DefaultConfigPath()
			w := cmd.OutOrStdout()
			if _, err := os.Stat(path); err != nil {
				fmt.Fprintf(w, "%s (not found)\n", path)
			} else {
				fmt.Fprintln(w, path)
			}
			return nil
		},
	}
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

// configPathFromFlags returns the config path from the --config flag or the default.
func configPathFromFlags(cmd *cobra.Command) string {
	if p, _ := cmd.Root().PersistentFlags().GetString("config"); p != "" {
		return p
	}
	return config.DefaultConfigPath()
}

// configToMap flattens a Config into a string map keyed by user-facing dot notation.
func configToMap(cfg *config.Config) map[string]string {
	m := map[string]string{
		"base_url":                  cfg.BaseURL,
		"credential_source":         string(cfg.CredentialSource),
		"defaults.format":           cfg.Format,
		"defaults.refresh_interval": strconv.Itoa(cfg.RefreshInterval),
		"defaults.email":            cfg.Email,
		"defaults.team":             cfg.Team,
		"defaults.service":          cfg.Service,
		"defaults.interactive":      strconv.FormatBool(cfg.Interactive),
		"tui.theme":                 cfg.TUI.Theme,
		"tui.show_resolved":         strconv.FormatBool(cfg.TUI.ShowResolved),
		"tui.page_size":             strconv.Itoa(cfg.TUI.PageSize),
		"tui.tabs":                  strings.Join(cfg.TUI.Tabs, ","),
	}
	return m
}

// configSourcesFromFile reads the raw TOML and returns a set of user-facing
// dot-notation keys that are explicitly set in the file.
func configSourcesFromFile(path string) map[string]struct{} {
	data, err := os.ReadFile(path) //nolint:gosec // path from --config or XDG default
	if err != nil {
		return nil
	}

	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil
	}

	result := make(map[string]struct{})
	flattenTOML("", raw, result)
	return result
}

// flattenTOML recursively walks a decoded TOML map and populates keys
// in dot notation (e.g. "defaults.format", "tui.tabs").
func flattenTOML(prefix string, m map[string]any, result map[string]struct{}) {
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			flattenTOML(fullKey, sub, result)
		} else {
			result[fullKey] = struct{}{}
		}
	}
}

// parseConfigValue converts a string value to the correct Go type for the given key.
func parseConfigValue(key, val string) (any, error) {
	switch key {
	case "defaults.refresh_interval":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("refresh_interval must be a positive integer, got %s", val)
		}
		return n, nil
	case "tui.page_size":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("page_size must be a positive integer, got %s", val)
		}
		return n, nil
	case "defaults.interactive", "tui.show_resolved":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("%s must be true or false, got %s", key, val)
		}
		return b, nil
	case "tui.tabs":
		tabs := strings.Split(val, ",")
		cleaned := make([]string, 0, len(tabs))
		for _, t := range tabs {
			t = strings.TrimSpace(t)
			if t != "" {
				cleaned = append(cleaned, t)
			}
		}
		if len(cleaned) == 0 {
			return nil, errors.New("tabs must contain at least one value")
		}
		return cleaned, nil
	default:
		return val, nil
	}
}

// modifyConfigFile reads a TOML file (creating it if absent), sets the key
// in the correct section and writes back.
func modifyConfigFile(cfgPath, key string, value any) error {
	raw, err := readTOMLFile(cfgPath)
	if err != nil {
		return err
	}

	section, leaf := splitConfigKey(key)
	if section == "" {
		raw[leaf] = value
	} else {
		sec, ok := raw[section].(map[string]any)
		if !ok {
			sec = make(map[string]any)
			raw[section] = sec
		}
		sec[leaf] = value
	}

	return writeTOMLFile(cfgPath, raw)
}

// removeConfigKey removes a key from the TOML file and cleans up empty sections.
// If the file becomes empty, it is deleted.
func removeConfigKey(cfgPath string, key string) error {
	raw, err := readTOMLFile(cfgPath)
	if err != nil {
		return err
	}

	section, leaf := splitConfigKey(key)
	if section == "" {
		delete(raw, leaf)
	} else {
		if sec, ok := raw[section].(map[string]any); ok {
			delete(sec, leaf)
			if len(sec) == 0 {
				delete(raw, section)
			}
		}
	}

	if len(raw) == 0 {
		if err := os.Remove(cfgPath); err == nil {
			clog.Info().Path("path", cfgPath).Msg("config file removed (empty)")
		}
		return nil
	}

	return writeTOMLFile(cfgPath, raw)
}

// splitConfigKey splits a user-facing dot-notation key into section and leaf.
// Top-level keys return ("", key). Dotted keys return ("section", "leaf").
func splitConfigKey(key string) (section, leaf string) {
	if before, after, ok := strings.Cut(key, "."); ok {
		return before, after
	}
	return "", key
}

// readTOMLFile reads and decodes a TOML file. Returns an empty map if the file
// does not exist.
func readTOMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from --config or XDG default
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	if raw == nil {
		raw = make(map[string]any)
	}
	return raw, nil
}

// writeTOMLFile encodes the document to a TOML file with mode 0600.
func writeTOMLFile(path string, doc map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // path from --config or XDG default
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(doc); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	clog.Info().Path("path", path).Msg("config updated")
	return nil
}

// completeConfigKeys provides tab completion for config key names.
func completeConfigKeys(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	completions := make([]string, len(configKeys))
	for i, key := range configKeys {
		completions[i] = key + "\t" + configDescriptions[key]
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
