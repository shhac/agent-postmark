package credential

import (
	"errors"
	"path/filepath"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/lib-agent-cli/creds"
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

// secretsStore is credentials.secrets.json's file: 0600 writes into a 0700
// parent, atomic replacement, and Update for a locked read-modify-write. It
// holds raw tokens, so a lost update here loses the secret itself, not just a
// pointer to it. Its lock is separate from the index's — the two are different
// files — and is only ever taken while already holding the index lock (never the
// other way round), so the ordering cannot deadlock.
func secretsStore() creds.Store {
	return creds.Store{Path: secretsPath()}
}

func readSecrets() (map[string]string, error) {
	m := map[string]string{}
	if err := secretsStore().Load(&m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

// errSkipWrite lets a mutate callback decline to persist anything without
// updateSecrets treating it as a real failure — see removeFileSecret, which must
// not conjure a secrets file for a keychain-backed install.
var errSkipWrite = errors.New("credential: skip write")

func updateSecrets(mutate func(secrets map[string]string) error) error {
	m := map[string]string{}
	err := secretsStore().Update(&m, func() error {
		if m == nil {
			m = map[string]string{}
		}
		return mutate(m)
	})
	if errors.Is(err, errSkipWrite) {
		return nil
	}
	return err
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
	if err := updateSecrets(func(secrets map[string]string) error {
		secrets[name] = token
		return nil
	}); err != nil {
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
	_ = updateSecrets(func(secrets map[string]string) error {
		if _, ok := secrets[name]; !ok {
			// Nothing file-backed under this name — the common case on a
			// keychain host, where writing would create an empty secrets file
			// that never existed before.
			return errSkipWrite
		}
		delete(secrets, name)
		return nil
	})
}
