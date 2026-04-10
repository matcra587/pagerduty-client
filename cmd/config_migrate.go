package cmd

import (
	"os"

	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/spf13/cobra"
)

// currentConfigVersion is the latest config format version.
const currentConfigVersion = 2

// configMigration is a step that transforms a raw TOML map from one
// version to the next. It returns the new version after applying.
type configMigration struct {
	from    int64
	to      int64
	migrate func(raw map[string]any)
}

// configMigrations is the ordered list of all config migrations.
var configMigrations = []configMigration{
	{from: 1, to: 2, migrate: migrateV1ToV2},
}

func configMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Update config file to the latest format",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath := configPathFromFlags(cmd)
			if err := migrateConfig(cfgPath); err != nil {
				return err
			}
			if isAgentMode(cmd) {
				return output.RenderAgentJSON(cmd.OutOrStdout(), "config migrate", compact.ResourceNone,
					map[string]string{"status": "ok", "path": cfgPath}, nil, nil)
			}
			return nil
		},
	}
}

// migrateConfig applies all pending migrations to a config file.
// If the file does not exist or is already current, it is a no-op.
func migrateConfig(cfgPath string) error {
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return nil
	}

	raw, err := readTOMLFile(cfgPath)
	if err != nil {
		return err
	}

	version := configVersion(raw)
	if version >= currentConfigVersion {
		clog.Info().Msg("config is already up to date")
		return nil
	}

	for _, m := range configMigrations {
		if version < m.to {
			m.migrate(raw)
			version = m.to
		}
	}

	raw["config_version"] = version

	if err := writeTOMLFile(cfgPath, raw); err != nil {
		return err
	}

	clog.Info().Int("version", int(version)).Path("path", cfgPath).Msg("config migrated")
	return nil
}

// configVersion returns the config_version from a raw TOML map.
// Absent means v1 (pre-versioning).
func configVersion(raw map[string]any) int64 {
	if v, ok := raw["config_version"].(int64); ok {
		return v
	}
	return 1
}

// migrateV1ToV2 restructures [tui] to [ui] / [ui.tui] and maps old
// theme preset names to clib equivalents.
func migrateV1ToV2(raw map[string]any) {
	tuiRaw, ok := raw["tui"].(map[string]any)
	if !ok {
		return
	}

	ui, _ := raw["ui"].(map[string]any)
	if ui == nil {
		ui = make(map[string]any)
	}

	if theme, ok := tuiRaw["theme"].(string); ok {
		if mapped := migrateThemeName(theme); mapped != "" {
			ui["theme"] = mapped
		}
		delete(tuiRaw, "theme")
	}

	if len(tuiRaw) > 0 {
		ui["tui"] = tuiRaw
	}

	raw["ui"] = ui
	delete(raw, "tui")
}

// migrateThemeName maps old custom preset names to clib equivalents.
// Unknown names pass through unchanged.
func migrateThemeName(name string) string {
	switch name {
	case "dark":
		return "" // was the default, omit to use clib default
	case "light":
		return "catppuccin-latte"
	case "high-contrast":
		return "monochrome"
	default:
		return name
	}
}
