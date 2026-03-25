package cmd

import (
	"context"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/credential"
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
	token, err := resolveToken(context.Background(), cfg, testToken)
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_EnvVarWins(t *testing.T) {
	t.Setenv("PDC_TOKEN", testToken)

	cfg := &config.Config{CredentialSource: credential.SourceKeyring} // keyring configured but env should win
	keyring.MockInit()                                                // empty keyring

	token, err := resolveToken(context.Background(), cfg, "") // no flag
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_Keyring(t *testing.T) {
	t.Setenv("PDC_TOKEN", "") // clear env
	keyring.MockInit()
	require.NoError(t, keyring.Set(credential.ServiceName, credential.AccountName, testToken))

	cfg := &config.Config{CredentialSource: credential.SourceKeyring}
	token, err := resolveToken(context.Background(), cfg, "")
	require.NoError(t, err)
	assert.Equal(t, testToken, token)
}

func TestResolveToken_KeyringNotFound_ReturnsError(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")
	keyring.MockInit() // empty store

	cfg := &config.Config{CredentialSource: credential.SourceKeyring}
	token, err := resolveToken(context.Background(), cfg, "")
	assert.Empty(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pdc init")
}

func TestResolveToken_EmptyFallthrough(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")

	// No credential source configured - falls through to empty token.
	// Validate() catches the missing token downstream.
	cfg := &config.Config{}
	token, err := resolveToken(context.Background(), cfg, "")
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestResolveToken_UnknownSource(t *testing.T) {
	t.Setenv("PDC_TOKEN", "")

	cfg := &config.Config{CredentialSource: "vault"}
	token, err := resolveToken(context.Background(), cfg, "")
	assert.Empty(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown credential_source")
}
