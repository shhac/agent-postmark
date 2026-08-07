package credential

import (
	"fmt"
	"path/filepath"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/lib-agent-cli/creds"
)

type TokenKind string

const (
	AccountToken TokenKind = "account"
	ServerToken  TokenKind = "server"
)

type entry struct {
	AccountToken bool            `json:"account_token"`
	ServerToken  bool            `json:"server_token"`
	Servers      map[string]bool `json:"servers,omitempty"`
}

type NotFoundError struct {
	Name string
	Kind TokenKind
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s token for profile %q not found", e.Kind, e.Name)
}

func Store(name string, kind TokenKind, token string) (string, error) {
	switch kind {
	case AccountToken:
		return StoreAccount(name, token)
	case ServerToken:
		return StoreServer(name, "default", token)
	default:
		return "", fmt.Errorf("unknown token kind %q", kind)
	}
}

func StoreAccount(profile string, token string) (string, error) {
	storage, err := storeSecret(accountKeychainName(profile), token)
	if err != nil {
		return "", err
	}
	// The index write is the step that must not race: the token is already
	// stored by now, so an entry lost to a concurrent writer leaves that token
	// referenced by nothing.
	if err := updateIndex(func(index map[string]entry) error {
		current := index[profile]
		current.AccountToken = true
		index[profile] = current
		return nil
	}); err != nil {
		return "", err
	}
	return storage, nil
}

func StoreServer(profile, server string, token string) (string, error) {
	storage, err := storeSecret(serverKeychainName(profile, server), token)
	if err != nil {
		return "", err
	}
	if err := updateIndex(func(index map[string]entry) error {
		current := index[profile]
		if current.Servers == nil {
			current.Servers = map[string]bool{}
		}
		current.Servers[server] = true
		if server == "default" {
			current.ServerToken = true
		}
		index[profile] = current
		return nil
	}); err != nil {
		return "", err
	}
	return storage, nil
}

func Get(name string, kind TokenKind) (string, error) {
	switch kind {
	case AccountToken:
		return GetAccount(name)
	case ServerToken:
		return GetServer(name, "default")
	default:
		return "", fmt.Errorf("unknown token kind %q", kind)
	}
}

func GetAccount(profile string) (string, error) {
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current, ok := index[profile]
	if !ok || !current.AccountToken {
		return "", &NotFoundError{Name: profile, Kind: AccountToken}
	}
	return getSecret(accountKeychainName(profile))
}

func GetServer(profile, server string) (string, error) {
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current, ok := index[profile]
	hasServer := ok && current.Servers != nil && current.Servers[server]
	hasLegacyDefault := ok && server == "default" && current.ServerToken
	if !hasServer && !hasLegacyDefault {
		return "", &NotFoundError{Name: profile + "/server/" + server, Kind: ServerToken}
	}
	token, err := getSecret(serverKeychainName(profile, server))
	if err != nil && server != "default" {
		return "", &NotFoundError{Name: profile + "/server/" + server, Kind: ServerToken}
	}
	return token, err
}

func Remove(name string) error {
	return updateIndex(func(index map[string]entry) error {
		current, ok := index[name]
		if !ok {
			return &NotFoundError{Name: name}
		}
		if current.AccountToken {
			deleteSecret(accountKeychainName(name))
		}
		if current.ServerToken {
			deleteSecret(serverKeychainName(name, "default"))
		}
		for server := range current.Servers {
			deleteSecret(serverKeychainName(name, server))
		}
		delete(index, name)
		return nil
	})
}

func RemoveServer(profile, server string) error {
	return updateIndex(func(index map[string]entry) error {
		current, ok := index[profile]
		hasServer := ok && current.Servers != nil && current.Servers[server]
		hasLegacyDefault := ok && server == "default" && current.ServerToken
		if !hasServer && !hasLegacyDefault {
			return &NotFoundError{Name: profile + "/server/" + server, Kind: ServerToken}
		}
		deleteSecret(serverKeychainName(profile, server))
		if current.Servers != nil {
			delete(current.Servers, server)
		}
		if server == "default" {
			current.ServerToken = false
		}
		index[profile] = current
		return nil
	})
}

func Summary(name string) map[string]any {
	index, err := readIndex()
	if err != nil {
		return map[string]any{}
	}
	current := index[name]
	servers := map[string]bool{}
	for alias, ok := range current.Servers {
		servers[alias] = ok
	}
	if current.ServerToken {
		servers["default"] = true
	}
	return map[string]any{
		"account_token_configured": current.AccountToken,
		"server_tokens_configured": servers,
	}
}

// AccountStorage reports where a profile's account token lives: "keychain",
// "file", or "" when none is stored. Read-only.
func AccountStorage(profile string) string {
	return secretBackend(accountKeychainName(profile))
}

// ServerStorage reports where a profile/server token lives: "keychain", "file",
// or "" when none is stored. Read-only.
func ServerStorage(profile, server string) string {
	return secretBackend(serverKeychainName(profile, server))
}

// ProfileStorage reports the backend for a profile's credentials as a whole. It
// returns the account token's backend when present, otherwise the backend of
// any stored server token, or "" when the profile has no stored secret. When
// backends differ across a profile's secrets it reports "mixed". Read-only.
func ProfileStorage(profile string) string {
	index, err := readIndex()
	if err != nil {
		return ""
	}
	current := index[profile]
	backend := ""
	note := func(b string) {
		if b == "" {
			return
		}
		if backend == "" {
			backend = b
			return
		}
		if backend != b {
			backend = "mixed"
		}
	}
	if current.AccountToken {
		note(AccountStorage(profile))
	}
	if current.ServerToken {
		note(ServerStorage(profile, "default"))
	}
	for server := range current.Servers {
		note(ServerStorage(profile, server))
	}
	return backend
}

func Type(token string) string {
	switch token {
	case "":
		return "missing"
	case "POSTMARK_API_TEST":
		return "test_server_token"
	default:
		return "postmark_token"
	}
}

func accountKeychainName(profile string) string {
	return profile + "/account"
}

func serverKeychainName(profile, server string) string {
	return profile + "/server/" + server
}

func credentialsPath() string {
	return filepath.Join(config.ConfigDir(), "credentials.json")
}

// indexStore is credentials.json's file: 0600 writes into a 0700 parent, atomic
// replacement, and Update for a locked read-modify-write. This used to be
// hand-rolled with os.ReadFile/os.WriteFile, which carried a lost-update race —
// two concurrent writers each built their write from a stale snapshot, and the
// loser's entry vanished while its token stayed in the keychain, unreferenced
// and un-removable (`profiles list` can't show it, `profiles remove` can't look
// it up).
func indexStore() creds.Store {
	return creds.Store{Path: credentialsPath()}
}

func readIndex() (map[string]entry, error) {
	out := map[string]entry{}
	if err := indexStore().Load(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]entry{}
	}
	return out, nil
}

// updateIndex applies mutate to the index under an exclusive lock, so two
// concurrent `profiles add`/`profiles remove` invocations serialize instead of
// clobbering each other. An error from mutate aborts without writing.
func updateIndex(mutate func(index map[string]entry) error) error {
	index := map[string]entry{}
	return indexStore().Update(&index, func() error {
		if index == nil {
			index = map[string]entry{}
		}
		return mutate(index)
	})
}
