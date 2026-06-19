package credential

import (
	"github.com/shhac/lib-agent-cli/creds"
)

// keychainService is this CLI's reverse-domain identity in the macOS login
// keychain. The shared creds package is deliberately app-agnostic, so the name
// lives here.
const keychainService = "app.paulie.agent-postmark"

var keychain = creds.NewKeychain(keychainService)

func keychainStore(name, token string) error {
	if !keychain.Available() {
		return creds.ErrKeychainUnavailable
	}
	return keychain.Set(name, token)
}

func keychainGet(name string) (string, error) {
	if !keychain.Available() {
		return "", creds.ErrKeychainUnavailable
	}
	token, ok := keychain.Get(name)
	if !ok {
		return "", &NotFoundError{Name: name}
	}
	return token, nil
}

func keychainDelete(name string) error {
	if !keychain.Available() {
		return nil
	}
	return keychain.Delete(name)
}
