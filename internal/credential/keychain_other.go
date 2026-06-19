//go:build !darwin

package credential

import "fmt"

const keychainService = "app.paulie.agent-postmark"

func keychainStore(name, token string) error {
	return fmt.Errorf("Keychain credential storage is only implemented on macOS")
}

func keychainGet(name string) (string, error) {
	return "", fmt.Errorf("Keychain credential storage is only implemented on macOS")
}

func keychainDelete(name string) error { return nil }
