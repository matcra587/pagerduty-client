package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// testToken is the Stoplight mock API token used as a fixture value across tests.
// It verifies the correct token value flows through each resolution step.
const testToken = "y_NbAkKc66ryYTWUXYEu" //nolint:gosec // test fixture, not a real credential

func TestResolveToken_FlagWins(t *testing.T) {
	t.Setenv("PDC_TOKEN", "env-token") // env set but flag should win
	keyring.MockInit()
	require.NoError(t, keyring.Set(credential.ServiceName, credential.AccountName, "keyring-token"))

	cfg := &config.Config{CredentialSource: credential.SourceKeyring}
	token, err := resolveToken(context.Background(), cfg, testToken, "")
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_EnvVarWins(t *testing.T) {
	t.Setenv("PDC_TOKEN", testToken)

	cfg := &config.Config{CredentialSource: credential.SourceKeyring} // keyring configured but env should win
	keyring.MockInit()                                                // empty keyring

	token, err := resolveToken(context.Background(), cfg, "", "") // no flag
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_Keyring(t *testing.T) {
	t.Setenv("PDC_TOKEN", "") // clear env
	keyring.MockInit()
	require.NoError(t, keyring.Set(credential.ServiceName, credential.AccountName, testToken))

	cfg := &config.Config{CredentialSource: credential.SourceKeyring}
	token, err := resolveToken(context.Background(), cfg, "", "")
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_KeyringNotFound_ReturnsError(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")
	keyring.MockInit() // empty store

	cfg := &config.Config{CredentialSource: credential.SourceKeyring}
	token, err := resolveToken(context.Background(), cfg, "", "")
	assert.Empty(t, token)
	require.Error(t, err)
	assert.ErrorContains(t, err, "pdc config init")
}

func TestResolveToken_EmptyFallthrough(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")

	// No credential source configured - falls through to empty token.
	// Validate() catches the missing token downstream.
	cfg := &config.Config{}
	token, err := resolveToken(context.Background(), cfg, "", "")
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestResolveToken_UnknownSource(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")

	cfg := &config.Config{CredentialSource: "vault"}
	token, err := resolveToken(context.Background(), cfg, "", "")
	assert.Empty(t, token)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown credential_source")
}

func TestResolveToken_FromFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(f, []byte("file-token\n"), 0o600))

	token, err := resolveToken(context.Background(), &config.Config{}, "", f)
	require.NoError(t, err)
	assert.Equal(t, "file-token", token)
}

func TestResolveToken_TokenAndFileAreMutuallyExclusive(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(f, []byte("file-token\n"), 0o600))

	_, err := resolveToken(context.Background(), &config.Config{}, "flag-token", f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "mutually exclusive")
}

func TestResolveToken_FileNotFound(t *testing.T) {
	_, err := resolveToken(context.Background(), &config.Config{}, "", "/nonexistent/token")
	require.Error(t, err)
	assert.ErrorContains(t, err, "reading token file")
}

func TestRunStartupUpdateCheck_TTYHumanStoresResult(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("PDC_NO_UPDATE_CHECK", "")

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/matcra587/pagerduty-client/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.4.0"})
	})

	t.Setenv("PDC_UPDATE_URL", server.URL)

	cmd := &cobra.Command{Use: "pdc"}
	cmd.SetContext(context.Background())

	state := preRunState{
		cfg: config.Default(),
		det: agent.DetectionResult{},
	}

	var notified []update.CheckResult
	runStartupUpdateCheck(cmd, state, true, func(result update.CheckResult) {
		notified = append(notified, result)
	})

	require.Len(t, notified, 1)

	got := UpdateResultFromContext(cmd)
	assert.Equal(t, notified[0], got)
	assert.Equal(t, "0.4.0", got.LatestRef)
	assert.Equal(t, update.ChannelStable, got.Channel)
}

func TestRunStartupUpdateCheck_SkipsForAgentOrNonTTY(t *testing.T) {
	tests := []struct {
		name      string
		agentMode bool
		isTTY     bool
	}{
		{name: "agent mode", agentMode: true, isTTY: true},
		{name: "not tty", agentMode: false, isTTY: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PDC_NO_UPDATE_CHECK", "")

			cmd := &cobra.Command{Use: "pdc"}
			cmd.SetContext(context.Background())

			state := preRunState{
				cfg: config.Default(),
				det: agent.DetectionResult{Active: tt.agentMode},
			}

			called := 0
			runStartupUpdateCheck(cmd, state, tt.isTTY, func(update.CheckResult) {
				called++
			})

			assert.Zero(t, called)
			assert.Equal(t, update.CheckResult{}, UpdateResultFromContext(cmd))
		})
	}
}
