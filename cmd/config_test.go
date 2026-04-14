package cmd

import (
	"os"
	"path/filepath"
	"testing"

	clibcobra "github.com/gechr/clib/cli/cobra"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// configToMap
// ---------------------------------------------------------------------------

func TestConfigToMap_FullyPopulated(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		BaseURL:          "https://api.pd.example.com",
		CredentialSource: credential.SourceKeyring,
		Format:           "json",
		RefreshInterval:  60,
		Email:            "user@example.com",
		Team:             "PTEAM01",
		Service:          "PSVC01",
		Interactive:      true,
		UI: config.UI{
			Theme: "dark",
			TUI: config.TUI{
				ShowResolved: true,
				PageSize:     50,
				Tabs:         []string{"incidents", "services"},
			},
		},
	}

	m := configToMap(cfg)

	assert.Equal(t, "https://api.pd.example.com", m["base_url"])
	assert.Equal(t, "keyring", m["credential_source"])
	assert.Equal(t, "json", m["defaults.format"])
	assert.Equal(t, "60", m["defaults.refresh_interval"])
	assert.Equal(t, "user@example.com", m["defaults.email"])
	assert.Equal(t, "PTEAM01", m["defaults.team"])
	assert.Equal(t, "PSVC01", m["defaults.service"])
	assert.Equal(t, "true", m["defaults.interactive"])
	assert.Equal(t, "dark", m["ui.theme"])
	assert.Equal(t, "true", m["ui.tui.show_resolved"])
	assert.Equal(t, "50", m["ui.tui.page_size"])
	assert.Equal(t, "incidents,services", m["ui.tui.tabs"])
}

func TestConfigToMap_Defaults(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	m := configToMap(cfg)

	assert.Equal(t, "table", m["defaults.format"])
	assert.Equal(t, "30", m["defaults.refresh_interval"])
	assert.Empty(t, m["base_url"])
}

// ---------------------------------------------------------------------------
// parseConfigValue
// ---------------------------------------------------------------------------

func TestParseConfigValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		val     string
		want    any
		wantErr bool
	}{
		{name: "string value", key: "base_url", val: "https://api.pd.example.com", want: "https://api.pd.example.com"},
		{name: "format string", key: "defaults.format", val: "json", want: "json"},
		{name: "refresh_interval valid", key: "defaults.refresh_interval", val: "60", want: 60},
		{name: "refresh_interval zero", key: "defaults.refresh_interval", val: "0", wantErr: true},
		{name: "refresh_interval negative", key: "defaults.refresh_interval", val: "-1", wantErr: true},
		{name: "refresh_interval not a number", key: "defaults.refresh_interval", val: "abc", wantErr: true},
		{name: "page_size valid", key: "ui.tui.page_size", val: "25", want: 25},
		{name: "page_size zero", key: "ui.tui.page_size", val: "0", wantErr: true},
		{name: "interactive true", key: "defaults.interactive", val: "true", want: true},
		{name: "interactive false", key: "defaults.interactive", val: "false", want: false},
		{name: "interactive invalid", key: "defaults.interactive", val: "yes", wantErr: true},
		{name: "show_resolved true", key: "ui.tui.show_resolved", val: "true", want: true},
		{name: "tabs single", key: "ui.tui.tabs", val: "incidents", want: []string{"incidents"}},
		{name: "tabs multiple", key: "ui.tui.tabs", val: "incidents,services,teams", want: []string{"incidents", "services", "teams"}},
		{name: "tabs with spaces", key: "ui.tui.tabs", val: "incidents, services", want: []string{"incidents", "services"}},
		{name: "tabs empty", key: "ui.tui.tabs", val: "", wantErr: true},
		{name: "theme valid", key: "ui.theme", val: "dracula", want: "dracula"},
		{name: "theme invalid", key: "ui.theme", val: "tokyonightasd", wantErr: true},
		{name: "theme empty", key: "ui.theme", val: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseConfigValue(tt.key, tt.val)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// modifyConfigFile - nested TOML sections
// ---------------------------------------------------------------------------

func TestModifyConfigFile_TopLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "base_url", "https://api.pd.example.com"))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Equal(t, "https://api.pd.example.com", cfg.BaseURL)
}

func TestModifyConfigFile_DefaultsSection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "defaults.format", "json"))
	require.NoError(t, modifyConfigFile(path, "defaults.refresh_interval", 60))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, 60, cfg.RefreshInterval)
}

func TestModifyConfigFile_UISection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "ui.theme", "light"))
	require.NoError(t, modifyConfigFile(path, "ui.tui.show_resolved", true))
	require.NoError(t, modifyConfigFile(path, "ui.tui.page_size", 50))
	require.NoError(t, modifyConfigFile(path, "ui.tui.tabs", []string{"incidents", "services"}))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Equal(t, "light", cfg.UI.Theme)
	assert.True(t, cfg.UI.TUI.ShowResolved)
	assert.Equal(t, 50, cfg.UI.TUI.PageSize)
	assert.Equal(t, []string{"incidents", "services"}, cfg.UI.TUI.Tabs)
}

func TestModifyConfigFile_PreservesExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write initial values.
	require.NoError(t, modifyConfigFile(path, "base_url", "https://api.pd.example.com"))
	require.NoError(t, modifyConfigFile(path, "defaults.format", "json"))

	// Add a new key - existing values should survive.
	require.NoError(t, modifyConfigFile(path, "defaults.email", "user@example.com"))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Equal(t, "https://api.pd.example.com", cfg.BaseURL)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "user@example.com", cfg.Email)
}

// ---------------------------------------------------------------------------
// removeConfigKey
// ---------------------------------------------------------------------------

func TestRemoveConfigKey_TopLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "base_url", "https://api.pd.example.com"))
	require.NoError(t, modifyConfigFile(path, "defaults.format", "json"))

	require.NoError(t, removeConfigKey(path, "base_url"))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Empty(t, cfg.BaseURL)
	assert.Equal(t, "json", cfg.Format)
}

func TestRemoveConfigKey_NestedCleansSection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "ui.theme", "dark"))

	require.NoError(t, removeConfigKey(path, "ui.theme"))

	// Section should be gone; file should still exist if other sections remain.
	// But in this case, the file was tui-only, so it should be deleted.
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "empty config file should be removed")
}

func TestRemoveConfigKey_FileRemovedWhenEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "base_url", "https://example.com"))
	require.NoError(t, removeConfigKey(path, "base_url"))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "empty config file should be removed")
}

// ---------------------------------------------------------------------------
// configSourcesFromFile
// ---------------------------------------------------------------------------

func TestConfigSourcesFromFile_NestedKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `credential_source = "keyring"

[defaults]
format = "json"
email  = "user@example.com"

[ui]
theme = "dark"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	sources := configSourcesFromFile(path)
	assert.Contains(t, sources, "credential_source")
	assert.Contains(t, sources, "defaults.format")
	assert.Contains(t, sources, "defaults.email")
	assert.Contains(t, sources, "ui.theme")
	assert.NotContains(t, sources, "defaults.team")
}

func TestConfigSourcesFromFile_MissingFile(t *testing.T) {
	t.Parallel()

	sources := configSourcesFromFile("/nonexistent/path/config.toml")
	assert.Nil(t, sources)
}

// ---------------------------------------------------------------------------
// splitConfigKey
// ---------------------------------------------------------------------------

func TestSplitConfigKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want []string
	}{
		{key: "base_url", want: []string{"base_url"}},
		{key: "defaults.format", want: []string{"defaults", "format"}},
		{key: "ui.theme", want: []string{"ui", "theme"}},
		{key: "ui.tui.tabs", want: []string{"ui", "tui", "tabs"}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, splitConfigKey(tt.key))
		})
	}
}

// ---------------------------------------------------------------------------
// Round-trip: set then Load
// ---------------------------------------------------------------------------

func TestRoundTrip_SetThenLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, modifyConfigFile(path, "defaults.format", "json"))
	require.NoError(t, modifyConfigFile(path, "defaults.refresh_interval", 45))
	require.NoError(t, modifyConfigFile(path, "defaults.interactive", true))
	require.NoError(t, modifyConfigFile(path, "ui.tui.tabs", []string{"incidents", "teams"}))

	cfg, err := config.Load(config.WithPath(path))
	require.NoError(t, err)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, 45, cfg.RefreshInterval)
	assert.True(t, cfg.Interactive)
	assert.Equal(t, []string{"incidents", "teams"}, cfg.UI.TUI.Tabs)
}

// ---------------------------------------------------------------------------
// completeConfigSetValues
// ---------------------------------------------------------------------------

func TestCompleteConfigSetValues_ThemePresets(t *testing.T) {
	t.Parallel()

	completions, directive := completeConfigSetValues(nil, []string{"ui.theme"}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, completions, "dracula")
	assert.Contains(t, completions, "monochrome")
	assert.Contains(t, completions, "catppuccin-mocha")
}

func TestConfigSetCmd_SubcommandsExposeDynamicArgs(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "config"}
	root.AddCommand(configSetCmd())

	subs := clibcobra.Subcommands(root)
	require.Len(t, subs, 1)
	assert.Equal(t, "set", subs[0].Name)
	assert.Equal(t, []string{"config_key", "config_value"}, subs[0].DynamicArgs)
}

func TestConfigGetCmd_SubcommandsExposeDynamicArgs(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "config"}
	root.AddCommand(configGetCmd())

	subs := clibcobra.Subcommands(root)
	require.Len(t, subs, 1)
	assert.Equal(t, "get", subs[0].Name)
	assert.Equal(t, []string{"config_key"}, subs[0].DynamicArgs)
}

func TestConfigUnsetCmd_SubcommandsExposeDynamicArgs(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "config"}
	root.AddCommand(configUnsetCmd())

	subs := clibcobra.Subcommands(root)
	require.Len(t, subs, 1)
	assert.Equal(t, "unset", subs[0].Name)
	assert.Equal(t, []string{"config_key"}, subs[0].DynamicArgs)
}
