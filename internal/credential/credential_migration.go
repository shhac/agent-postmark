package credential

import (
	"fmt"
	"strings"
)

const migrateCommand = "agent-postmark profiles --migrate"

// MigrationRequiredError reports that a token can only be read from the legacy
// Keychain service name until the user explicitly runs the migration command.
type MigrationRequiredError struct {
	Name string
	Kind TokenKind
}

func (e *MigrationRequiredError) Error() string {
	return fmt.Sprintf("%s token for %q was found under old Keychain service %q and must be migrated to %q", e.Kind, e.Name, legacyKeychainService, keychainService)
}

func (e *MigrationRequiredError) Hint() string {
	return fmt.Sprintf("Run '%s' to migrate stored credentials.", migrateCommand)
}

func getAccountWithMigration(profile string, requireMigration bool) (string, error) {
	return getWithMigration(profile, AccountToken, []string{
		accountKeychainName(profile),
		legacyKeychainName(profile, AccountToken),
	}, requireMigration)
}

// GetAccountWithMigration reads an account token with the temporary service-name migration path.
func GetAccountWithMigration(profile string, requireMigration bool) (string, error) {
	return getAccountWithMigration(profile, requireMigration)
}

func getServerWithMigration(profile, server string, requireMigration bool) (string, error) {
	names := []string{serverKeychainName(profile, server)}
	if server == "default" {
		names = append(names, legacyKeychainName(profile, ServerToken))
	}
	return getWithMigration(profile+"/server/"+server, ServerToken, names, requireMigration)
}

// GetServerWithMigration reads a server token with the temporary service-name migration path.
func GetServerWithMigration(profile, server string, requireMigration bool) (string, error) {
	return getServerWithMigration(profile, server, requireMigration)
}

func getWithMigration(label string, kind TokenKind, names []string, requireMigration bool) (string, error) {
	var firstErr error
	for _, name := range names {
		token, err := keychainGetForService(keychainService, name)
		if validCredential(token) {
			return token, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	for _, name := range names {
		token, err := keychainGetForService(legacyKeychainService, name)
		if !validCredential(token) {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if requireMigration {
			return "", &MigrationRequiredError{Name: label, Kind: kind}
		}
		return token, nil
	}
	return "", firstErr
}

// MigrateLegacyCredentials copies legacy-service tokens for every indexed
// profile/server to the current service and deletes each migrated legacy entry.
func MigrateLegacyCredentials() (int, error) {
	index, err := readIndex()
	if err != nil {
		return 0, err
	}
	migrated := 0
	for profile, current := range index {
		count, err := migrateLegacyToken(accountKeychainName(profile), []string{
			accountKeychainName(profile),
			legacyKeychainName(profile, AccountToken),
		}, current.AccountToken)
		if err != nil {
			return migrated, err
		}
		migrated += count

		defaultNeeded := current.ServerToken || (current.Servers != nil && current.Servers["default"])
		count, err = migrateLegacyToken(serverKeychainName(profile, "default"), []string{
			serverKeychainName(profile, "default"),
			legacyKeychainName(profile, ServerToken),
		}, defaultNeeded)
		if err != nil {
			return migrated, err
		}
		migrated += count

		for server, configured := range current.Servers {
			if server == "default" {
				continue
			}
			count, err = migrateLegacyToken(serverKeychainName(profile, server), []string{
				serverKeychainName(profile, server),
			}, configured)
			if err != nil {
				return migrated, err
			}
			migrated += count
		}
	}
	return migrated, nil
}

func migrateLegacyToken(targetName string, legacyNames []string, configured bool) (int, error) {
	if !configured {
		return 0, nil
	}
	if token, err := keychainGetForService(keychainService, targetName); err == nil && validCredential(token) {
		return 0, nil
	}
	for _, legacyName := range legacyNames {
		token, err := keychainGetForService(legacyKeychainService, legacyName)
		if err != nil || !validCredential(token) {
			continue
		}
		if err := keychainStoreForService(keychainService, targetName, token); err != nil {
			return 0, err
		}
		if err := keychainDeleteForService(legacyKeychainService, legacyName); err != nil {
			return 0, err
		}
		return 1, nil
	}
	return 0, nil
}

func validCredential(token string) bool {
	return strings.TrimSpace(token) != ""
}
