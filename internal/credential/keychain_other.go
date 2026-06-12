//go:build !darwin

package credential

import "fmt"

const keychainService = "app.paulie.agent-postmark"

// KEYCHAIN-MIGRATION: Remove this legacy service constant after the migration window.
const legacyKeychainService = "agent-postmark"

// KEYCHAIN-MIGRATION: Service-specific helpers let tests and migration code share the darwin shape.
var (
	keychainStoreForService  = platformKeychainStore
	keychainGetForService    = platformKeychainGet
	keychainDeleteForService = platformKeychainDelete
)

func keychainStore(name, token string) error {
	return keychainStoreForService(keychainService, name, token)
}

func platformKeychainStore(service, name, token string) error {
	return fmt.Errorf("Keychain credential storage is only implemented on macOS")
}

func keychainGet(name string) (string, error) {
	return keychainGetForService(keychainService, name)
}

func platformKeychainGet(service, name string) (string, error) {
	return "", fmt.Errorf("Keychain credential storage is only implemented on macOS")
}

func keychainDelete(name string) error {
	return keychainDeleteForService(keychainService, name)
}

func platformKeychainDelete(service, name string) error { return nil }
