package credential

import (
	"context"
	"errors"
)

// Source identifies a credential backend.
type Source string

const (
	// SourceKeyring uses the OS keyring (Keychain, Credential Manager, Secret Service).
	SourceKeyring Source = "keyring"
)

// CredentialProvider retrieves an API token from a credential store.
type CredentialProvider interface {
	Provide(ctx context.Context) (string, error)
}

// ErrNotFound indicates the credential is not configured in this source.
// All CredentialProvider implementations must return ErrNotFound (never nil)
// when the credential is absent. Returning ("", nil) is a programming error.
var ErrNotFound = errors.New("credential not found")
