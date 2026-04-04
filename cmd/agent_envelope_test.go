package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetRootCmd resets the global rootCmd output and args to prevent
// state leaking between sequential tests.
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})
}

// TestAgentMode_JSONEnvelope verifies that non-API commands produce
// a valid JSON envelope when running in agent mode. Tests run
// sequentially because they share the global rootCmd.
func TestAgentMode_JSONEnvelope(t *testing.T) {
	// Sequential - rootCmd is shared global state.

	tests := []struct {
		name    string
		args    []string
		command string
	}{
		{name: "version", args: []string{"version"}, command: "version"},
		{name: "config list", args: []string{"config", "list"}, command: "config list"},
		{name: "config get", args: []string{"config", "get", "defaults.format"}, command: "config get"},
		{name: "config path", args: []string{"config", "path"}, command: "config path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRootCmd(t)

			cfgFile := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(cfgFile, nil, 0o600))

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetArgs(append(tt.args, "--agent", "--config", cfgFile))

			err := rootCmd.Execute()
			require.NoError(t, err)

			var env map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
				"output must be valid JSON envelope, got: %s", buf.String())
			assert.Equal(t, true, env["success"])
			assert.Equal(t, tt.command, env["command"])
			assert.NotNil(t, env["data"])
		})
	}
}

// TestAgentMode_ConfigSet_JSONEnvelope verifies config set produces
// an envelope confirming the write.
func TestAgentMode_ConfigSet_JSONEnvelope(t *testing.T) {
	resetRootCmd(t)

	cfgFile := filepath.Join(t.TempDir(), "config.toml")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"config", "set", "defaults.format", "json", "--agent", "--config", cfgFile})

	err := rootCmd.Execute()
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
		"output must be valid JSON envelope, got: %s", buf.String())
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "config set", env["command"])
	assert.NotNil(t, env["data"])
}

// TestAgentMode_ConfigUnset_JSONEnvelope verifies config unset
// produces an envelope confirming the removal.
func TestAgentMode_ConfigUnset_JSONEnvelope(t *testing.T) {
	resetRootCmd(t)

	cfgFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("[defaults]\nformat = \"json\"\n"), 0o600))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"config", "unset", "defaults.format", "--agent", "--config", cfgFile})

	err := rootCmd.Execute()
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env),
		"output must be valid JSON envelope, got: %s", buf.String())
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "config unset", env["command"])
	assert.NotNil(t, env["data"])
}
