// Package credential provides credential store backends.
package credential

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keyring service identifier.
	ServiceName = "pagerduty-client"
	// AccountName is the keyring account key for the PagerDuty API token.
	AccountName = "api-token"
)

// KeyringProvider retrieves the API token from the OS keyring.
// On macOS it reads from Keychain; on Windows, Credential Manager;
// on Linux, Secret Service (requires a running D-Bus secret service).
type KeyringProvider struct{}

var _ CredentialProvider = KeyringProvider{}

func (p KeyringProvider) Provide(_ context.Context) (string, error) {
	token, err := keyring.Get(ServiceName, AccountName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("keyring lookup: %w", err)
	}
	if token == "" {
		return "", ErrNotFound
	}
	return token, nil
}
