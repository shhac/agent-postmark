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
	AccountToken bool `json:"account_token"`
	ServerToken  bool `json:"server_token"`
}

type NotFoundError struct {
	Name string
	Kind TokenKind
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s token for profile %q not found", e.Kind, e.Name)
}

func Store(name string, kind TokenKind, token string) (string, error) {
	if err := keychainStore(keychainName(name, kind), token); err != nil {
		return "", err
	}
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current := index[name]
	switch kind {
	case AccountToken:
		current.AccountToken = true
	case ServerToken:
		current.ServerToken = true
	default:
		return "", fmt.Errorf("unknown token kind %q", kind)
	}
	index[name] = current
	if err := writeIndex(index); err != nil {
		return "", err
	}
	return "keychain", nil
}

func Get(name string, kind TokenKind) (string, error) {
	index, err := readIndex()
	if err != nil {
		return "", err
	}
	current, ok := index[name]
	if !ok || (kind == AccountToken && !current.AccountToken) || (kind == ServerToken && !current.ServerToken) {
		return "", &NotFoundError{Name: name, Kind: kind}
	}
	return keychainGet(keychainName(name, kind))
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
		_ = keychainDelete(keychainName(name, AccountToken))
	}
	if current.ServerToken {
		_ = keychainDelete(keychainName(name, ServerToken))
	}
	delete(index, name)
	return writeIndex(index)
}

func Summary(name string) map[string]bool {
	index, err := readIndex()
	if err != nil {
		return map[string]bool{}
	}
	current := index[name]
	return map[string]bool{
		"account_token": current.AccountToken,
		"server_token":  current.ServerToken,
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

func keychainName(profile string, kind TokenKind) string {
	return profile + ":" + string(kind)
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
