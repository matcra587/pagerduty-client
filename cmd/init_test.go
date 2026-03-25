package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenViaAPI_Success(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/users/me", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Token token="+testToken, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"user":{"id":"U1","name":"Test User","email":"test@example.com"}}`))
	})

	err := validateTokenViaAPI(context.Background(), testToken, []api.Option{api.WithBaseURL(srv.URL)})
	require.NoError(t, err)
}

func TestValidateTokenViaAPI_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/users/me", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Unauthorized","code":2006}}`))
	})

	err := validateTokenViaAPI(context.Background(), testToken, []api.Option{api.WithBaseURL(srv.URL)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestWriteInitConfig_KeyringSource(t *testing.T) {
	dir := t.TempDir()
	ic := initConfig{credentialSource: credential.SourceKeyring}

	require.NoError(t, writeInitConfig(dir, ic))

	cfg, err := config.Load(config.WithPath(filepath.Join(dir, "config.toml")))
	require.NoError(t, err)
	assert.Equal(t, credential.SourceKeyring, cfg.CredentialSource)
	assert.Empty(t, cfg.Token)
}

func TestWriteInitConfig_KeyringSourceWithDefaults(t *testing.T) {
	dir := t.TempDir()
	ic := initConfig{
		credentialSource: credential.SourceKeyring,
		defaultTeamID:    "T1",
		defaultServiceID: "S1",
	}

	require.NoError(t, writeInitConfig(dir, ic))

	cfg, err := config.Load(config.WithPath(filepath.Join(dir, "config.toml")))
	require.NoError(t, err)
	assert.Equal(t, credential.SourceKeyring, cfg.CredentialSource)
	assert.Equal(t, "T1", cfg.Team)
	assert.Equal(t, "S1", cfg.Service)
	assert.Empty(t, cfg.Token)
}

func TestWriteInitConfig_EmptySource(t *testing.T) {
	dir := t.TempDir()
	ic := initConfig{} // PDC_TOKEN flow - no credential source written

	require.NoError(t, writeInitConfig(dir, ic))

	cfg, err := config.Load(config.WithPath(filepath.Join(dir, "config.toml")))
	require.NoError(t, err)
	assert.Empty(t, cfg.CredentialSource)
	assert.Empty(t, cfg.Token)
}

func TestWriteInitConfig_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "path")
	ic := initConfig{credentialSource: credential.SourceKeyring}

	require.NoError(t, writeInitConfig(dir, ic))
	_, err := os.Stat(filepath.Join(dir, "config.toml"))
	require.NoError(t, err)
}

func TestWriteInitConfig_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions not enforced the same way on Windows")
	}

	dir := t.TempDir()
	ic := initConfig{credentialSource: credential.SourceKeyring}

	require.NoError(t, writeInitConfig(dir, ic))

	info, err := os.Stat(filepath.Join(dir, "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
