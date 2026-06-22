package credential

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/shhac/agent-postmark/internal/config"
)

// credentials.secrets.json holds raw tokens when the macOS keychain is
// unavailable — non-macOS hosts, or an explicit opt-out via
// AGENT_POSTMARK_NO_KEYCHAIN / LIB_AGENT_NO_KEYCHAIN. It mirrors the keychain's
// key space: keys are the same composite names used for keychain items
// ("<profile>/account", "<profile>/server/<server>"), so the index format in
// credentials.json is unchanged and existing installs keep working.
func secretsPath() string {
	return filepath.Join(config.ConfigDir(), "credentials.secrets.json")
}

func readSecrets() (map[string]string, error) {
	data, err := os.ReadFile(secretsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

func writeSecrets(m map[string]string) error {
	if err := os.MkdirAll(config.ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(secretsPath(), append(data, '\n'), 0o600)
}

// storeSecret persists a token under name, preferring the macOS keychain and
// falling back to the 0600 secrets file when the keychain is unavailable.
// Returns "keychain" or "file".
func storeSecret(name, token string) (string, error) {
	if err := keychainStore(name, token); err == nil {
		// A prior file-backed secret for this name is now stale.
		removeFileSecret(name)
		return "keychain", nil
	}
	secrets, err := readSecrets()
	if err != nil {
		return "", err
	}
	secrets[name] = token
	if err := writeSecrets(secrets); err != nil {
		return "", err
	}
	return "file", nil
}

// secretBackend reports where the secret for name lives: "keychain", "file", or
// "" when no secret is stored under that name. Read-only. The file store is only
// populated on keychain fallback (storeSecret removes the file copy once the
// keychain accepts the write), so a name present in the file is file-backed and
// anything readable from the keychain is keychain-backed.
func secretBackend(name string) string {
	if secrets, err := readSecrets(); err == nil {
		if _, ok := secrets[name]; ok {
			return "file"
		}
	}
	if _, err := keychainGet(name); err == nil {
		return "keychain"
	}
	return ""
}

// getSecret reads a token, trying the keychain first, then the secrets file.
func getSecret(name string) (string, error) {
	if token, err := keychainGet(name); err == nil {
		return token, nil
	}
	secrets, err := readSecrets()
	if err != nil {
		return "", err
	}
	if token, ok := secrets[name]; ok {
		return token, nil
	}
	return "", &NotFoundError{Name: name}
}

// deleteSecret removes a token from both the keychain and the secrets file.
func deleteSecret(name string) {
	_ = keychainDelete(name)
	removeFileSecret(name)
}

func removeFileSecret(name string) {
	secrets, err := readSecrets()
	if err != nil {
		return
	}
	if _, ok := secrets[name]; !ok {
		return
	}
	delete(secrets, name)
	_ = writeSecrets(secrets)
}
