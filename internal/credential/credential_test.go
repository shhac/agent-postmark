package credential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// setupHeadless points the credential store at a temp dir and forces the file
// fallback, so the concurrency tests below exercise credentials.json AND
// credentials.secrets.json without touching the host keychain (which would also
// mean a GUI prompt on darwin).
func setupHeadless(t *testing.T) string {
	t.Helper()
	t.Setenv("AGENT_POSTMARK_NO_KEYCHAIN", "1")
	dir := t.TempDir()
	config.SetConfigDir(dir)
	t.Cleanup(func() { config.SetConfigDir("") })
	return dir
}

// Concurrent StoreAccount calls must not lose each other's entries, in either
// file they touch.
//
// This is the failure that matters most for the index: the secret write has
// already succeeded by the time credentials.json is written, so an entry lost to
// a racing writer leaves a live token that nothing references — invisible to
// `profiles list` and unreachable by `profiles remove`, which looks the name up
// in the index first. Under the keychain opt-out the same race also hit
// credentials.secrets.json, where the lost update is the token itself.
func TestConcurrentStoreAccountDoesNotLoseEntries(t *testing.T) {
	setupHeadless(t)

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			profile := fmt.Sprintf("profile-%02d", i)
			if _, err := StoreAccount(profile, fmt.Sprintf("acct-token-%02d", i)); err != nil {
				t.Errorf("StoreAccount: %v", err)
			}
		}(i)
	}
	wg.Wait()

	index, err := readIndex()
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	if len(index) != writers {
		t.Errorf("%d of %d entries survived in credentials.json — updates were lost", len(index), writers)
	}
	secrets, err := readSecrets()
	if err != nil {
		t.Fatalf("readSecrets: %v", err)
	}
	if len(secrets) != writers {
		t.Errorf("%d of %d secrets survived in credentials.secrets.json — updates were lost", len(secrets), writers)
	}
	for i := range writers {
		profile := fmt.Sprintf("profile-%02d", i)
		got, err := GetAccount(profile)
		if err != nil {
			t.Errorf("%s was lost: %v", profile, err)
			continue
		}
		if want := fmt.Sprintf("acct-token-%02d", i); got != want {
			t.Errorf("%s round-tripped as %q, want %q", profile, got, want)
		}
	}
}

// StoreServer merges into a nested map under a SINGLE index entry, so every
// concurrent writer here reads and rewrites the same key — the shape most likely
// to drop a sibling's write. `profiles setup` fans out exactly this way, one
// call per --server.
func TestConcurrentStoreServerDoesNotLoseEntries(t *testing.T) {
	setupHeadless(t)

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			server := fmt.Sprintf("server-%02d", i)
			if _, err := StoreServer("prod", server, fmt.Sprintf("server-token-%02d", i)); err != nil {
				t.Errorf("StoreServer: %v", err)
			}
		}(i)
	}
	wg.Wait()

	index, err := readIndex()
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	if got := len(index["prod"].Servers); got != writers {
		t.Errorf("%d of %d servers survived in credentials.json — updates were lost", got, writers)
	}
	for i := range writers {
		server := fmt.Sprintf("server-%02d", i)
		got, err := GetServer("prod", server)
		if err != nil {
			t.Errorf("%s was lost: %v", server, err)
			continue
		}
		if want := fmt.Sprintf("server-token-%02d", i); got != want {
			t.Errorf("%s round-tripped as %q, want %q", server, got, want)
		}
	}
}
