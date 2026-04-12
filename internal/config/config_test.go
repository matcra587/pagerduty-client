package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := config.Default()
	assert.Empty(t, cfg.BaseURL)
	assert.Equal(t, "table", cfg.Format)
	assert.Equal(t, 30, cfg.RefreshInterval)
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(config.WithPath(""))
	require.NoError(t, err)
	assert.Empty(t, cfg.BaseURL)
	assert.Equal(t, "table", cfg.Format)
	assert.Equal(t, 30, cfg.RefreshInterval)
}

func TestLoad_TOMLFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
base_url = "https://custom.pagerduty.com"

[defaults]
team = "PTEAM01"
service = "PSVC01"
format = "json"
refresh_interval = 60

[ui]
theme = "dark"

[ui.tui]
show_resolved = true
page_size = 50
tabs = ["incidents", "services"]
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.Equal(t, "https://custom.pagerduty.com", cfg.BaseURL)
	assert.Equal(t, "PTEAM01", cfg.Team)
	assert.Equal(t, "PSVC01", cfg.Service)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, 60, cfg.RefreshInterval)
	assert.Equal(t, "dark", cfg.UI.Theme)
	assert.True(t, cfg.UI.TUI.ShowResolved)
	assert.Equal(t, 50, cfg.UI.TUI.PageSize)
	assert.Equal(t, []string{"incidents", "services"}, cfg.UI.TUI.Tabs)
}

func TestLoad_UISection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	data := `
[ui]
theme = "dracula"

[ui.tui]
show_resolved = true
page_size = 25
tabs = ["incidents", "teams"]
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(data), 0o600))

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.Equal(t, "dracula", cfg.UI.Theme)
	assert.True(t, cfg.UI.TUI.ShowResolved)
	assert.Equal(t, 25, cfg.UI.TUI.PageSize)
	assert.Equal(t, []string{"incidents", "teams"}, cfg.UI.TUI.Tabs)
}

func TestLoad_EnvOverlay(t *testing.T) {
	t.Setenv("PDC_TEAM", "PENVTEAM")
	t.Setenv("PDC_SERVICE", "PENVSVC")
	t.Setenv("PDC_BASE_URL", "https://env.pagerduty.com")
	t.Setenv("PDC_FORMAT", "json")
	t.Setenv("PDC_DEBUG", "true")

	cfg, err := config.Load(config.WithPath(""))
	require.NoError(t, err)
	assert.Equal(t, "PENVTEAM", cfg.Team)
	assert.Equal(t, "PENVSVC", cfg.Service)
	assert.Equal(t, "https://env.pagerduty.com", cfg.BaseURL)
	assert.Equal(t, "json", cfg.Format)
	assert.True(t, cfg.Debug)
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
[defaults]
team = "PFILETEAM"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))
	t.Setenv("PDC_TEAM", "PENVTEAM")

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.Equal(t, "PENVTEAM", cfg.Team)
}

func TestLoad_ThemeEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
[ui]
theme = "catppuccin-mocha"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))
	t.Setenv("PDC_THEME", "dracula")

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.Equal(t, "dracula", cfg.UI.Theme)
}

func TestLoad_OptionOverridesEnv(t *testing.T) {
	t.Setenv("PDC_TEAM", "PENVTEAM")

	cfg, err := config.Load(config.WithPath(""), config.WithTeam("POPTTEAM"))
	require.NoError(t, err)
	assert.Equal(t, "POPTTEAM", cfg.Team)
}

func TestLoad_WithToken(t *testing.T) {
	cfg, err := config.Load(config.WithPath(""), config.WithToken("test-token"))
	require.NoError(t, err)
	assert.Equal(t, "test-token", cfg.Token)
}

func TestValidate_TokenOptional(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Format: "table", RefreshInterval: 30}
	cfg.SetTokenOptional()
	require.NoError(t, cfg.Validate())
}

func TestValidate_MissingToken(t *testing.T) {
	cfg := config.Default()
	err := cfg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "token is required")
	assert.ErrorContains(t, err, "pdc config init")
}

func TestValidate_BadFormat(t *testing.T) {
	cfg := config.Default()
	cfg.Token = "tok"
	cfg.Format = "xml"
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid format")
}

func TestValidate_BadRefreshInterval(t *testing.T) {
	cfg := config.Default()
	cfg.Token = "tok"
	cfg.RefreshInterval = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "refresh interval must be positive")
}

func TestValidate_OK(t *testing.T) {
	cfg := config.Default()
	cfg.Token = "tok"
	require.NoError(t, cfg.Validate())
}

func TestDefaultConfigPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgtest")
	got := config.DefaultConfigPath()
	assert.Equal(t, "/tmp/xdgtest/pagerduty-client/config.toml", got)
}

func TestDefaultConfigPath_Home(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got := config.DefaultConfigPath()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "pagerduty-client", "config.toml"), got)
}

func TestLoad_CustomFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
[[custom_fields]]
label = "Region"
path = "details.region"
display = "inline"

[[custom_fields]]
label = "Tier"
path = "details.tier"
display = "badge"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	require.Len(t, cfg.CustomFields, 2)
	assert.Equal(t, "Region", cfg.CustomFields[0].Label)
	assert.Equal(t, "details.region", cfg.CustomFields[0].Path)
	assert.Equal(t, "Tier", cfg.CustomFields[1].Label)
}

func TestLoad_CredentialSource(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
credential_source = "keyring"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.Equal(t, credential.SourceKeyring, cfg.CredentialSource)
}

func TestLoad_MissingFileFallsBackToDefaults(t *testing.T) {
	cfg, err := config.Load(config.WithPath("/nonexistent/config.toml"))
	require.NoError(t, err)
	assert.Empty(t, cfg.BaseURL)
	assert.Equal(t, "table", cfg.Format)
	assert.Equal(t, 30, cfg.RefreshInterval)
}

func TestLoad_InteractiveFromTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
[defaults]
interactive = true
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(toml), 0o600))

	cfg, err := config.Load(config.WithPath(cfgPath))
	require.NoError(t, err)
	assert.True(t, cfg.Interactive)
}

func TestLoad_DotenvFile(t *testing.T) {
	// Write a .env file in a temp directory and chdir there so
	// godotenv.Load() picks it up.
	dir := t.TempDir()
	dotenv := "PDC_TEAM=PDOTENV\nPDC_BASE_URL=https://dotenv.pagerduty.com\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))

	t.Chdir(dir)

	// godotenv.Load does not overwrite existing variables, so
	// they must be fully unset (not just empty).
	origTeam, teamSet := os.LookupEnv("PDC_TEAM")
	origBase, baseSet := os.LookupEnv("PDC_BASE_URL")
	require.NoError(t, os.Unsetenv("PDC_TEAM"))
	require.NoError(t, os.Unsetenv("PDC_BASE_URL"))
	t.Cleanup(func() {
		if teamSet {
			_ = os.Setenv("PDC_TEAM", origTeam) //nolint:errcheck,usetesting // cleanup restores pre-test state
		}
		if baseSet {
			_ = os.Setenv("PDC_BASE_URL", origBase) //nolint:errcheck,usetesting // cleanup restores pre-test state
		}
	})

	cfg, err := config.Load(config.WithPath(""))
	require.NoError(t, err)
	assert.Equal(t, "PDOTENV", cfg.Team)
	assert.Equal(t, "https://dotenv.pagerduty.com", cfg.BaseURL)
}

func TestLoad_InteractiveFromEnv(t *testing.T) {
	t.Setenv("PDC_INTERACTIVE", "1")

	cfg, err := config.Load(config.WithPath(""))
	require.NoError(t, err)
	assert.True(t, cfg.Interactive)
}
