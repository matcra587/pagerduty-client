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
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/output"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/spf13/cobra"
)

// isAgentMode detects agent mode from env vars and the --agent flag
// for commands that bypass PersistentPreRunE (config, version).
func isAgentMode(cmd *cobra.Command) bool {
	flag, _ := cmd.Root().PersistentFlags().GetBool("agent")
	return agent.DetectWithFlag(flag).Active
}

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
	"defaults.update_channel",
	"ui.theme",
	"ui.tui.show_resolved",
	"ui.tui.page_size",
	"ui.tui.tabs",
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
	"defaults.update_channel":   "Update channel (stable/dev)",
	"ui.theme":                  "Colour theme",
	"ui.tui.show_resolved":      "Show resolved incidents",
	"ui.tui.page_size":          "TUI page size",
	"ui.tui.tabs":               "TUI tab selection",
}

// envVarMapping maps config keys to their environment variable equivalents.
// Only keys with env var support are listed.
var envVarMapping = map[string]string{
	"base_url":                "PDC_BASE_URL",
	"defaults.email":          "PDC_EMAIL",
	"defaults.team":           "PDC_TEAM",
	"defaults.service":        "PDC_SERVICE",
	"defaults.format":         "PDC_FORMAT",
	"defaults.interactive":    "PDC_INTERACTIVE",
	"defaults.update_channel": "PDC_UPDATE_CHANNEL",
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

			type configEntry struct {
				Key         string `json:"key"`
				Value       string `json:"value"`
				Source      string `json:"source"`
				Description string `json:"description"`
			}

			entries := make([]configEntry, 0, len(configKeys))
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
				entries = append(entries, configEntry{
					Key:         key,
					Value:       val,
					Source:      source,
					Description: configDescriptions[key],
				})
			}

			w := cmd.OutOrStdout()
			if isAgentMode(cmd) {
				return output.RenderAgentJSON(w, "config list", compact.ResourceNone, entries, nil, nil)
			}

			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, []string{e.Key, e.Value, e.Source, e.Description})
			}

			th := pdctheme.Resolve(cfg.UI.Theme)
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

			fmt.Fprintln(w, t.Render())
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

			w := cmd.OutOrStdout()
			if isAgentMode(cmd) {
				data := map[string]string{"key": args[0], "value": val}
				return output.RenderAgentJSON(w, "config get", compact.ResourceNone, data, nil, nil)
			}

			fmt.Fprintln(w, val)
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
		ValidArgsFunction: completeConfigSetValues,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			cfgPath := configPathFromFlags(cmd)
			if err := modifyConfigFile(cfgPath, key, typed); err != nil {
				return err
			}
			if isAgentMode(cmd) {
				data := map[string]string{"key": key, "value": val}
				return output.RenderAgentJSON(cmd.OutOrStdout(), "config set", compact.ResourceNone, data, nil, nil)
			}
			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if !slices.Contains(configKeys, args[0]) {
				return fmt.Errorf("unknown config key %s - valid keys: %s", args[0], strings.Join(configKeys, ", "))
			}
			cfgPath := configPathFromFlags(cmd)
			if err := removeConfigKey(cfgPath, args[0]); err != nil {
				return err
			}
			if isAgentMode(cmd) {
				data := map[string]string{"key": args[0]}
				return output.RenderAgentJSON(cmd.OutOrStdout(), "config unset", compact.ResourceNone, data, nil, nil)
			}
			return nil
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
			path := configPathFromFlags(cmd)
			w := cmd.OutOrStdout()

			_, statErr := os.Stat(path)
			if isAgentMode(cmd) {
				data := map[string]any{
					"path":   path,
					"exists": statErr == nil,
				}
				return output.RenderAgentJSON(w, "config path", compact.ResourceNone, data, nil, nil)
			}

			if statErr != nil {
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
		"defaults.update_channel":   cfg.UpdateChannel,
		"ui.theme":                  cfg.UI.Theme,
		"ui.tui.show_resolved":      strconv.FormatBool(cfg.UI.TUI.ShowResolved),
		"ui.tui.page_size":          strconv.Itoa(cfg.UI.TUI.PageSize),
		"ui.tui.tabs":               strings.Join(cfg.UI.TUI.Tabs, ","),
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
// in dot notation (e.g. "defaults.format", "ui.tui.tabs").
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
	case "ui.tui.page_size":
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("page_size must be a positive integer, got %s", val)
		}
		return n, nil
	case "defaults.interactive", "ui.tui.show_resolved":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("%s must be true or false, got %s", key, val)
		}
		return b, nil
	case "ui.tui.tabs":
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
// at the correct nested path and writes back.
func modifyConfigFile(cfgPath, key string, value any) error {
	raw, err := readTOMLFile(cfgPath)
	if err != nil {
		return err
	}

	parts := splitConfigKey(key)
	setNestedKey(raw, parts, value)

	return writeTOMLFile(cfgPath, raw)
}

// setNestedKey walks a dot-notation path through nested maps, creating
// intermediate maps as needed, and sets the leaf value.
func setNestedKey(m map[string]any, parts []string, value any) {
	for i, p := range parts {
		if i == len(parts)-1 {
			m[p] = value
			return
		}
		sub, ok := m[p].(map[string]any)
		if !ok {
			sub = make(map[string]any)
			m[p] = sub
		}
		m = sub
	}
}

// removeConfigKey removes a key from the TOML file and cleans up empty sections.
// If the file becomes empty, it is deleted.
func removeConfigKey(cfgPath string, key string) error {
	raw, err := readTOMLFile(cfgPath)
	if err != nil {
		return err
	}

	parts := splitConfigKey(key)
	deleteNestedKey(raw, parts)

	if len(raw) == 0 {
		if err := os.Remove(cfgPath); err == nil {
			clog.Info().Path("path", cfgPath).Msg("config file removed (empty)")
		}
		return nil
	}

	return writeTOMLFile(cfgPath, raw)
}

// deleteNestedKey walks a dot-notation path, deletes the leaf, and
// removes empty parent maps on the way back up.
func deleteNestedKey(m map[string]any, parts []string) {
	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		return
	}
	deleteNestedKey(sub, parts[1:])
	if len(sub) == 0 {
		delete(m, parts[0])
	}
}

// splitConfigKey splits a user-facing dot-notation key into path
// segments. For example "ui.tui.tabs" returns ["ui", "tui", "tabs"].
func splitConfigKey(key string) []string {
	return strings.Split(key, ".")
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

// completeConfigSetValues provides tab completion for config set.
// First arg completes key names; second arg completes valid values
// for keys that have a known set (e.g. ui.theme).
func completeConfigSetValues(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		completions := make([]string, len(configKeys))
		for i, key := range configKeys {
			completions[i] = key + "\t" + configDescriptions[key]
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		switch args[0] {
		case "ui.theme":
			return pdctheme.PresetNames(), cobra.ShellCompDirectiveNoFileComp
		case "defaults.format":
			return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
		case "defaults.interactive", "ui.tui.show_resolved":
			return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
		case "defaults.update_channel":
			return []string{"stable", "dev"}, cobra.ShellCompDirectiveNoFileComp
		}
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
