package credential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/agent-postmark/internal/config"
)

// TestStore_Headless_FileFallback exercises the credential-WRITE path
// non-interactively. The per-CLI keychain opt-out (derived by lib-agent-cli from
// the "app.paulie.agent-postmark" service) makes the keychain report
// unavailable, so account and server tokens deterministically land in the 0600
// credentials.secrets.json on every platform — including darwin, where they would
// otherwise reach the `security` GUI prompt. Before the file fallback existed,
// Store simply failed under the opt-out (and on any non-macOS host).
func TestStore_Headless_FileFallback(t *testing.T) {
	t.Setenv("AGENT_POSTMARK_NO_KEYCHAIN", "1")
	dir := t.TempDir()
	config.SetConfigDir(dir)
	t.Cleanup(func() { config.SetConfigDir("") })

	storage, err := StoreAccount("prof", "acct-headless-token")
	if err != nil {
		t.Fatalf("StoreAccount: %v", err)
	}
	if storage != "file" {
		t.Fatalf("account storage=%q, want \"file\" (keychain opt-out should force the file path)", storage)
	}
	storage, err = StoreServer("prof", "default", "server-headless-token")
	if err != nil {
		t.Fatalf("StoreServer: %v", err)
	}
	if storage != "file" {
		t.Fatalf("server storage=%q, want \"file\"", storage)
	}

	secretsFile := filepath.Join(dir, "credentials.secrets.json")
	info, err := os.Stat(secretsFile)
	if err != nil {
		t.Fatalf("secrets file not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("secrets mode=%o, want 0600", mode)
	}
	secretsData, _ := os.ReadFile(secretsFile)
	for _, want := range []string{"acct-headless-token", "server-headless-token"} {
		if !strings.Contains(string(secretsData), want) {
			t.Errorf("secrets file missing %q; got %s", want, secretsData)
		}
	}
	// The index records only booleans — secrets must never leak into it.
	indexData, _ := os.ReadFile(filepath.Join(dir, "credentials.json"))
	for _, leak := range []string{"acct-headless-token", "server-headless-token"} {
		if strings.Contains(string(indexData), leak) {
			t.Errorf("index file leaked secret %q; got %s", leak, indexData)
		}
	}

	if got, err := GetAccount("prof"); err != nil || got != "acct-headless-token" {
		t.Errorf("GetAccount=%q,%v; want acct-headless-token,nil", got, err)
	}
	if got, err := GetServer("prof", "default"); err != nil || got != "server-headless-token" {
		t.Errorf("GetServer=%q,%v; want server-headless-token,nil", got, err)
	}

	if err := Remove("prof"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := GetAccount("prof"); err == nil {
		t.Error("expected NotFound after Remove")
	}
	if rest, _ := readSecrets(); len(rest) != 0 {
		t.Errorf("secrets file still has entries after Remove: %v", rest)
	}
}
