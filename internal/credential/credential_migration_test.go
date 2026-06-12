package credential

import (
	"errors"
	"testing"

	"github.com/shhac/agent-postmark/internal/config"
)

func TestKeychainMigrationReadsNewServiceFirst(t *testing.T) {
	store := testKeychain(t)
	store[keychainService][accountKeychainName("prod")] = "account_new"
	store[legacyKeychainService][accountKeychainName("prod")] = "account_legacy"

	token, err := GetAccountWithMigration("prod", true)
	if err != nil {
		t.Fatal(err)
	}
	if token != "account_new" {
		t.Fatalf("token = %q, want account_new", token)
	}
}

func TestKeychainMigrationRequiresExplicitMigrationForLegacyOnly(t *testing.T) {
	store := testKeychain(t)
	store[legacyKeychainService][accountKeychainName("prod")] = "account_legacy"

	_, err := GetAccountWithMigration("prod", true)
	var migrationErr *MigrationRequiredError
	if !errors.As(err, &migrationErr) {
		t.Fatalf("err = %v, want MigrationRequiredError", err)
	}
	if migrationErr.Hint() != "Run 'agent-postmark profiles --migrate' to migrate stored credentials." {
		t.Fatalf("hint = %q", migrationErr.Hint())
	}
}

func TestKeychainMigrationNoMigrateFallsBackSilently(t *testing.T) {
	store := testKeychain(t)
	store[legacyKeychainService][serverKeychainName("prod", "primary")] = "server_legacy"

	token, err := GetServerWithMigration("prod", "primary", false)
	if err != nil {
		t.Fatal(err)
	}
	if token != "server_legacy" {
		t.Fatalf("token = %q, want server_legacy", token)
	}
}

func TestKeychainMigrationMovesLegacyCredentials(t *testing.T) {
	store := testKeychain(t)
	store[legacyKeychainService][legacyKeychainName("prod", AccountToken)] = "account_legacy"
	store[legacyKeychainService][legacyKeychainName("prod", ServerToken)] = "server_legacy"
	index := map[string]entry{
		"prod": {
			AccountToken: true,
			ServerToken:  true,
		},
	}
	if err := writeIndex(index); err != nil {
		t.Fatal(err)
	}

	migrated, err := MigrateLegacyCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if migrated != 2 {
		t.Fatalf("migrated = %d, want 2", migrated)
	}
	if got := store[keychainService][accountKeychainName("prod")]; got != "account_legacy" {
		t.Fatalf("new account token = %q, want account_legacy", got)
	}
	if got := store[keychainService][serverKeychainName("prod", "default")]; got != "server_legacy" {
		t.Fatalf("new server token = %q, want server_legacy", got)
	}
	if _, ok := store[legacyKeychainService][legacyKeychainName("prod", AccountToken)]; ok {
		t.Fatal("legacy account token was not deleted")
	}
	if _, ok := store[legacyKeychainService][legacyKeychainName("prod", ServerToken)]; ok {
		t.Fatal("legacy server token was not deleted")
	}
}

func testKeychain(t *testing.T) map[string]map[string]string {
	t.Helper()
	t.Cleanup(func() {
		config.SetConfigDir("")
		config.ClearCache()
		keychainStoreForService = platformKeychainStore
		keychainGetForService = platformKeychainGet
		keychainDeleteForService = platformKeychainDelete
	})
	config.SetConfigDir(t.TempDir())
	store := map[string]map[string]string{
		keychainService:       {},
		legacyKeychainService: {},
	}
	keychainStoreForService = func(service, name, token string) error {
		if store[service] == nil {
			store[service] = map[string]string{}
		}
		store[service][name] = token
		return nil
	}
	keychainGetForService = func(service, name string) (string, error) {
		if token, ok := store[service][name]; ok {
			return token, nil
		}
		return "", errors.New("not found")
	}
	keychainDeleteForService = func(service, name string) error {
		delete(store[service], name)
		return nil
	}
	return store
}
