package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })
	return dir
}

// Concurrent StoreProfile calls must not lose each other's entries.
//
// Before updateConfig routed through creds.Store.Update, StoreProfile did
// Read() -> mutate -> Write(). Two concurrent invocations — in-process sharing
// the package cache, or across processes sharing config.json — each built their
// write from a snapshot taken before the other's landed, so all but the last
// writer's profile were silently erased.
func TestConcurrentStoreProfileDoesNotLoseEntries(t *testing.T) {
	setupTestDir(t)

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := StoreProfile(fmt.Sprintf("profile-%02d", i), Profile{}); err != nil {
				t.Errorf("StoreProfile: %v", err)
			}
		}(i)
	}
	wg.Wait()

	ClearCache()
	cfg := Read()
	if len(cfg.Profiles) != writers {
		t.Fatalf("%d of %d concurrent StoreProfile calls survived — updates were lost", len(cfg.Profiles), writers)
	}
	for i := range writers {
		alias := fmt.Sprintf("profile-%02d", i)
		if _, ok := cfg.Profiles[alias]; !ok {
			t.Errorf("%s was lost from config.json", alias)
		}
	}
}

// StoreServer composes a profile lookup with a nested-map mutation, so it is the
// mutator most likely to lose a sibling's write: every caller reads and rewrites
// the SAME profile entry. `profiles setup` fans out exactly this way, one call
// per --server.
func TestConcurrentStoreServerDoesNotLoseEntries(t *testing.T) {
	setupTestDir(t)

	if err := StoreProfile("prod", Profile{}); err != nil {
		t.Fatalf("StoreProfile: %v", err)
	}

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			alias := fmt.Sprintf("server-%02d", i)
			if err := StoreServer("prod", alias, ServerProfile{ServerID: 9000 + i}); err != nil {
				t.Errorf("StoreServer: %v", err)
			}
		}(i)
	}
	wg.Wait()

	ClearCache()
	servers := Read().Profiles["prod"].Servers
	if len(servers) != writers {
		t.Fatalf("%d of %d concurrent StoreServer calls survived — updates were lost", len(servers), writers)
	}
	for i := range writers {
		alias := fmt.Sprintf("server-%02d", i)
		if got := servers[alias].ServerID; got != 9000+i {
			t.Errorf("%s round-tripped with server_id %d, want %d", alias, got, 9000+i)
		}
	}
}

// config.json now goes through creds.Store, which writes every file 0600
// regardless of content — one audited place to get file permissions right rather
// than a per-file policy. That is a tightening from the previous 0644, not a
// regression: this directory is shared with credentials.secrets.json, which
// holds raw tokens whenever the keychain is unavailable.
func TestConfigFilePerms(t *testing.T) {
	dir := setupTestDir(t)
	if err := Write(&Config{Profiles: map[string]Profile{}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file perms = %o, want 600", perm)
	}
}
