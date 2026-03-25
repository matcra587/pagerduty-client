package credential_test

import (
	"context"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestKeyringProvider_Constants(t *testing.T) {
	assert.Equal(t, "pagerduty-client", credential.ServiceName)
	assert.Equal(t, "api-token", credential.AccountName)
}

func TestKeyringProvider_Provide_ReturnsToken(t *testing.T) {
	keyring.MockInit()
	require.NoError(t, keyring.Set(credential.ServiceName, credential.AccountName, "my-secret-token"))

	p := credential.KeyringProvider{}
	token, err := p.Provide(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "my-secret-token", token)
}

func TestKeyringProvider_Provide_NotFound(t *testing.T) {
	keyring.MockInit() // fresh empty store

	p := credential.KeyringProvider{}
	token, err := p.Provide(context.Background())
	assert.Empty(t, token)
	assert.ErrorIs(t, err, credential.ErrNotFound)
}
