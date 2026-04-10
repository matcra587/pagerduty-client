package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateConfig_RewritesTUIToUI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	old := `base_url = "https://api.pd.example.com"

[defaults]
format = "json"

[tui]
theme = "dark"
show_resolved = true
page_size = 50
tabs = ["incidents", "services"]
`
	require.NoError(t, os.WriteFile(path, []byte(old), 0o600))

	require.NoError(t, migrateConfig(path))

	raw, err := readTOMLFile(path)
	require.NoError(t, err)

	// Old [tui] section should be gone.
	_, hasTUI := raw["tui"]
	assert.False(t, hasTUI, "old [tui] section should be removed")

	// Version should be set.
	assert.Equal(t, int64(currentConfigVersion), raw["config_version"])

	// New [ui] section should exist. "dark" maps to default (omitted).
	ui, ok := raw["ui"].(map[string]any)
	require.True(t, ok, "missing [ui] section")
	assert.Nil(t, ui["theme"], "dark should be omitted (it was the default)")

	// TUI-specific keys nested under [ui.tui].
	tui, ok := ui["tui"].(map[string]any)
	require.True(t, ok, "missing [ui.tui] section")
	assert.Equal(t, true, tui["show_resolved"])
	assert.Equal(t, int64(50), tui["page_size"])
	assert.Equal(t, []any{"incidents", "services"}, tui["tabs"])
}

func TestMigrateConfig_LightThemeMapped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	old := `[tui]
theme = "light"
`
	require.NoError(t, os.WriteFile(path, []byte(old), 0o600))

	require.NoError(t, migrateConfig(path))

	raw, err := readTOMLFile(path)
	require.NoError(t, err)
	ui, ok := raw["ui"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "catppuccin-latte", ui["theme"])
	assert.Equal(t, int64(2), raw["config_version"])
}

func TestMigrateConfig_HighContrastMapped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	old := `[tui]
theme = "high-contrast"
`
	require.NoError(t, os.WriteFile(path, []byte(old), 0o600))

	require.NoError(t, migrateConfig(path))

	raw, err := readTOMLFile(path)
	require.NoError(t, err)
	ui, ok := raw["ui"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "monochrome", ui["theme"])
}

func TestMigrateConfig_UnknownThemePassesThrough(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	old := `[tui]
theme = "dracula"
`
	require.NoError(t, os.WriteFile(path, []byte(old), 0o600))

	require.NoError(t, migrateConfig(path))

	raw, err := readTOMLFile(path)
	require.NoError(t, err)
	ui, ok := raw["ui"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "dracula", ui["theme"])
}

func TestMigrateConfig_AlreadyCurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	data := `config_version = 2

[ui]
theme = "dracula"
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))

	before, err := os.Stat(path)
	require.NoError(t, err)

	require.NoError(t, migrateConfig(path))

	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "file should not be rewritten")
}

func TestMigrateConfig_NoTUISection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	data := `base_url = "https://api.pd.example.com"
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))

	require.NoError(t, migrateConfig(path))

	raw, err := readTOMLFile(path)
	require.NoError(t, err)
	assert.Equal(t, "https://api.pd.example.com", raw["base_url"])
	assert.Equal(t, int64(2), raw["config_version"])
}

func TestMigrateConfig_FileNotFound(t *testing.T) {
	t.Parallel()

	err := migrateConfig("/nonexistent/config.toml")
	require.NoError(t, err, "missing file should be a no-op")
}
