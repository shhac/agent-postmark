package credential

import (
	"github.com/shhac/lib-agent-cli/creds"
)

// keychainService is this CLI's reverse-domain identity in the macOS login
// keychain. The shared creds package is deliberately app-agnostic, so the name
// lives here.
const keychainService = "app.paulie.agent-postmark"

// MCPKeychainService is the Keychain service for the MCP server's local-OAuth
// secrets — the CLI's service plus a ".mcp" namespace, separate from the API creds.
func MCPKeychainService() string { return keychainService + ".mcp" }

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
