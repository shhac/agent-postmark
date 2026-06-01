package credential

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shhac/agent-postmark/internal/config"
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
	if err := keychainStore(accountKeychainName(profile), token); err != nil {
		return "", err
	}
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current := index[profile]
	current.AccountToken = true
	index[profile] = current
	if err := writeIndex(index); err != nil {
		return "", err
	}
	return "keychain", nil
}

func StoreServer(profile, server string, token string) (string, error) {
	if err := keychainStore(serverKeychainName(profile, server), token); err != nil {
		return "", err
	}
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current := index[profile]
	if current.Servers == nil {
		current.Servers = map[string]bool{}
	}
	current.Servers[server] = true
	if server == "default" {
		current.ServerToken = true
	}
	index[profile] = current
	if err := writeIndex(index); err != nil {
		return "", err
	}
	return "keychain", nil
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
	if token, err := keychainGet(accountKeychainName(profile)); err == nil {
		return token, nil
	}
	return keychainGet(legacyKeychainName(profile, AccountToken))
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
	if token, err := keychainGet(serverKeychainName(profile, server)); err == nil {
		return token, nil
	}
	if server == "default" {
		return keychainGet(legacyKeychainName(profile, ServerToken))
	}
	return "", &NotFoundError{Name: profile + "/server/" + server, Kind: ServerToken}
}

func Remove(name string) error {
	index, err := readIndex()
	if err != nil {
		return err
	}
	current, ok := index[name]
	if !ok {
		return &NotFoundError{Name: name}
	}
	if current.AccountToken {
		_ = keychainDelete(accountKeychainName(name))
		_ = keychainDelete(legacyKeychainName(name, AccountToken))
	}
	if current.ServerToken {
		_ = keychainDelete(legacyKeychainName(name, ServerToken))
		_ = keychainDelete(serverKeychainName(name, "default"))
	}
	for server := range current.Servers {
		_ = keychainDelete(serverKeychainName(name, server))
	}
	delete(index, name)
	return writeIndex(index)
}

func RemoveServer(profile, server string) error {
	index, err := readIndex()
	if err != nil {
		return err
	}
	current, ok := index[profile]
	hasServer := ok && current.Servers != nil && current.Servers[server]
	hasLegacyDefault := ok && server == "default" && current.ServerToken
	if !hasServer && !hasLegacyDefault {
		return &NotFoundError{Name: profile + "/server/" + server, Kind: ServerToken}
	}
	_ = keychainDelete(serverKeychainName(profile, server))
	if current.Servers != nil {
		delete(current.Servers, server)
	}
	if server == "default" {
		current.ServerToken = false
		_ = keychainDelete(legacyKeychainName(profile, ServerToken))
	}
	index[profile] = current
	return writeIndex(index)
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

func legacyKeychainName(profile string, kind TokenKind) string {
	return profile + ":" + string(kind)
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

func readIndex() (map[string]entry, error) {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]entry{}, nil
		}
		return nil, err
	}
	var out map[string]entry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]entry{}
	}
	return out, nil
}

func writeIndex(index map[string]entry) error {
	if err := os.MkdirAll(config.ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credentialsPath(), append(data, '\n'), 0o600)
}
