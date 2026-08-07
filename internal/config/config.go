package config

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/shhac/lib-agent-cli/creds"
	"github.com/shhac/lib-agent-cli/xdg"
)

const DefaultHost = "https://api.postmarkapp.com"

type Config struct {
	DefaultProfile string             `json:"default_profile,omitempty"`
	Defaults       Defaults           `json:"defaults,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Defaults struct {
	TimeoutMS  *int `json:"timeout_ms,omitempty"`
	MaxRetries *int `json:"max_retries,omitempty"`
}

type Profile struct {
	Host           string                   `json:"host,omitempty"`
	DefaultServer  string                   `json:"default_server,omitempty"`
	Servers        map[string]ServerProfile `json:"servers,omitempty"`
	AccountTokenID string                   `json:"account_token_id,omitempty"`

	LegacyDefaultServerID int    `json:"default_server_id,omitempty"`
	LegacyMessageStream   string `json:"message_stream,omitempty"`
	LegacyServerTokenID   string `json:"server_token_id,omitempty"`
}

type ServerProfile struct {
	ServerID      int    `json:"server_id,omitempty"`
	MessageStream string `json:"message_stream,omitempty"`
	ServerTokenID string `json:"server_token_id,omitempty"`
}

func ErrProfileNotConfigured(alias string) error {
	return fmt.Errorf("profile %q is not configured", alias)
}

var (
	cache       *Config
	cacheMu     sync.Mutex
	overrideDir string
)

func SetConfigDir(dir string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	overrideDir = dir
	cache = nil
}

func ConfigDir() string {
	if overrideDir != "" {
		return overrideDir
	}
	return xdg.ConfigDir("agent-postmark")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

// store is config.json's file: 0600 writes into a 0700 parent, atomic
// replacement, and Update for a locked read-modify-write. This used to be
// hand-rolled with os.ReadFile/os.WriteFile, which carried a lost-update race —
// two concurrent CLI invocations (e.g. `profiles add` racing `profiles servers
// add`) could each build their write from a snapshot taken before the other
// landed, silently erasing one of them.
func store() creds.Store {
	return creds.Store{Path: ConfigPath()}
}

func Read() *Config {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cache != nil {
		return cache
	}
	cache = loadConfig()
	return cache
}

// loadConfig reads config.json fresh from disk, bypassing the package cache. It
// is the single definition of "what a from-scratch read looks like", shared by
// Read (cached) and updateConfig, which must never hand a mutate callback the
// stale in-memory cache while holding the store's lock.
func loadConfig() *Config {
	var cfg Config
	if err := store().Load(&cfg); err != nil {
		return defaultConfig()
	}
	hydrate(&cfg)
	return &cfg
}

// hydrate fills in what a freshly decoded Config needs before anything reads or
// mutates it: a non-nil profile map, and per-profile normalization (default
// host, legacy single-server fields folded into the servers map).
func hydrate(cfg *Config) {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	for alias, profile := range cfg.Profiles {
		cfg.Profiles[alias] = normalizeProfile(profile)
	}
}

func Write(cfg *Config) error {
	if err := store().Save(cfg); err != nil {
		return err
	}
	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()
	return nil
}

// updateConfig applies mutate to a freshly loaded config under ONE exclusive
// lock spanning read, mutate, and write, so two concurrent invocations serialize
// instead of each building its write from a stale snapshot. The package-level
// cache is bypassed entirely while the lock is held — mutate always sees what
// store().Update just loaded from disk — and is invalidated afterward so a later
// Read cannot hand back the pre-write value.
//
// An error from mutate aborts the Update without writing, so the callers that
// reject an unknown profile/server/key leave config.json untouched exactly as
// they did before.
func updateConfig(mutate func(cfg *Config) error) error {
	var cfg Config
	err := store().Update(&cfg, func() error {
		hydrate(&cfg)
		return mutate(&cfg)
	})

	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()

	return err
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = nil
}

func StoreProfile(alias string, profile Profile) error {
	return updateConfig(func(cfg *Config) error {
		cfg.Profiles[alias] = normalizeProfile(profile)
		if cfg.DefaultProfile == "" {
			cfg.DefaultProfile = alias
		}
		return nil
	})
}

func StoreServer(profileAlias, serverAlias string, server ServerProfile) error {
	return updateConfig(func(cfg *Config) error {
		return applyToProfile(cfg, profileAlias, func(profile Profile) Profile {
			if profile.Servers == nil {
				profile.Servers = map[string]ServerProfile{}
			}
			profile.Servers[serverAlias] = normalizeServer(server)
			if profile.DefaultServer == "" {
				profile.DefaultServer = serverAlias
			}
			return profile
		})
	})
}

func UpdateProfile(alias string, update func(Profile) Profile) error {
	return updateConfig(func(cfg *Config) error {
		return applyToProfile(cfg, alias, update)
	})
}

// applyToProfile is UpdateProfile's body without the store lock, so mutators
// that compose it (StoreServer, RemoveServer) stay a single locked
// read-modify-write. Calling UpdateProfile from inside another updateConfig
// would deadlock: flock is per-open-file-description, so a second acquisition
// from the same process blocks on the first.
func applyToProfile(cfg *Config, alias string, update func(Profile) Profile) error {
	profile, ok := cfg.Profiles[alias]
	if !ok {
		return ErrProfileNotConfigured(alias)
	}
	cfg.Profiles[alias] = normalizeProfile(update(profile))
	return nil
}

func UpdateServer(profileAlias, serverAlias string, update func(ServerProfile) ServerProfile) error {
	return updateConfig(func(cfg *Config) error {
		profile, ok := cfg.Profiles[profileAlias]
		if !ok {
			return ErrProfileNotConfigured(profileAlias)
		}
		server, ok := profile.Servers[serverAlias]
		if !ok {
			return fmt.Errorf("server %q is not configured for profile %q", serverAlias, profileAlias)
		}
		profile.Servers[serverAlias] = normalizeServer(update(server))
		cfg.Profiles[profileAlias] = profile
		return nil
	})
}

func RemoveServer(profileAlias, serverAlias string) error {
	return updateConfig(func(cfg *Config) error {
		return applyToProfile(cfg, profileAlias, func(profile Profile) Profile {
			delete(profile.Servers, serverAlias)
			if profile.DefaultServer == serverAlias {
				profile.DefaultServer = ""
				for alias := range profile.Servers {
					profile.DefaultServer = alias
					break
				}
			}
			return profile
		})
	})
}

func RemoveProfile(alias string) error {
	return updateConfig(func(cfg *Config) error {
		delete(cfg.Profiles, alias)
		if cfg.DefaultProfile == alias {
			cfg.DefaultProfile = ""
			for name := range cfg.Profiles {
				cfg.DefaultProfile = name
				break
			}
		}
		return nil
	})
}

func SetDefaultServer(profileAlias, serverAlias string) error {
	return updateConfig(func(cfg *Config) error {
		profile, ok := cfg.Profiles[profileAlias]
		if !ok {
			return ErrProfileNotConfigured(profileAlias)
		}
		if _, ok := profile.Servers[serverAlias]; !ok {
			return fmt.Errorf("server %q is not configured for profile %q", serverAlias, profileAlias)
		}
		profile.DefaultServer = serverAlias
		cfg.Profiles[profileAlias] = profile
		return nil
	})
}

func SetDefault(alias string) error {
	return updateConfig(func(cfg *Config) error {
		if _, ok := cfg.Profiles[alias]; !ok {
			return fmt.Errorf("profile %q is not configured", alias)
		}
		cfg.DefaultProfile = alias
		return nil
	})
}

func SetDefaultValue(key string, value int) error {
	return updateConfig(func(cfg *Config) error {
		switch key {
		case "timeout_ms":
			cfg.Defaults.TimeoutMS = intPtr(value)
		case "max_retries":
			cfg.Defaults.MaxRetries = intPtr(value)
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return nil
	})
}

func UnsetDefaultValue(key string) error {
	return updateConfig(func(cfg *Config) error {
		switch key {
		case "timeout_ms":
			cfg.Defaults.TimeoutMS = nil
		case "max_retries":
			cfg.Defaults.MaxRetries = nil
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return nil
	})
}

func defaultConfig() *Config {
	return &Config{Profiles: map[string]Profile{}}
}

func normalizeProfile(profile Profile) Profile {
	if profile.Host == "" {
		profile.Host = DefaultHost
	}
	if profile.Servers == nil {
		profile.Servers = map[string]ServerProfile{}
	}
	if len(profile.Servers) == 0 && (profile.LegacyDefaultServerID != 0 || profile.LegacyMessageStream != "") {
		profile.DefaultServer = creds.FirstNonEmpty(profile.DefaultServer, "default")
		profile.Servers[profile.DefaultServer] = ServerProfile{
			ServerID:      profile.LegacyDefaultServerID,
			MessageStream: profile.LegacyMessageStream,
			ServerTokenID: profile.LegacyServerTokenID,
		}
	}
	for alias, server := range profile.Servers {
		profile.Servers[alias] = normalizeServer(server)
	}
	return profile
}

func normalizeServer(server ServerProfile) ServerProfile {
	if server.MessageStream == "" {
		server.MessageStream = "outbound"
	}
	return server
}

func intPtr(value int) *int { return &value }
