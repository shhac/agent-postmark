package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

func Read() *Config {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cache != nil {
		return cache
	}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		cache = defaultConfig()
		return cache
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		cache = defaultConfig()
		return cache
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	for alias, profile := range cfg.Profiles {
		cfg.Profiles[alias] = normalizeProfile(profile)
	}
	cache = &cfg
	return cache
}

func Write(cfg *Config) error {
	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()

	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), append(data, '\n'), 0o644)
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = nil
}

func StoreProfile(alias string, profile Profile) error {
	cfg := Read()
	cfg.Profiles[alias] = normalizeProfile(profile)
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = alias
	}
	return Write(cfg)
}

func StoreServer(profileAlias, serverAlias string, server ServerProfile) error {
	return UpdateProfile(profileAlias, func(profile Profile) Profile {
		if profile.Servers == nil {
			profile.Servers = map[string]ServerProfile{}
		}
		profile.Servers[serverAlias] = normalizeServer(server)
		if profile.DefaultServer == "" {
			profile.DefaultServer = serverAlias
		}
		return profile
	})
}

func UpdateProfile(alias string, update func(Profile) Profile) error {
	cfg := Read()
	profile, ok := cfg.Profiles[alias]
	if !ok {
		return ErrProfileNotConfigured(alias)
	}
	cfg.Profiles[alias] = normalizeProfile(update(profile))
	return Write(cfg)
}

func UpdateServer(profileAlias, serverAlias string, update func(ServerProfile) ServerProfile) error {
	cfg := Read()
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
	return Write(cfg)
}

func RemoveServer(profileAlias, serverAlias string) error {
	return UpdateProfile(profileAlias, func(profile Profile) Profile {
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
}

func RemoveProfile(alias string) error {
	cfg := Read()
	delete(cfg.Profiles, alias)
	if cfg.DefaultProfile == alias {
		cfg.DefaultProfile = ""
		for name := range cfg.Profiles {
			cfg.DefaultProfile = name
			break
		}
	}
	return Write(cfg)
}

func SetDefaultServer(profileAlias, serverAlias string) error {
	cfg := Read()
	profile, ok := cfg.Profiles[profileAlias]
	if !ok {
		return ErrProfileNotConfigured(profileAlias)
	}
	if _, ok := profile.Servers[serverAlias]; !ok {
		return fmt.Errorf("server %q is not configured for profile %q", serverAlias, profileAlias)
	}
	profile.DefaultServer = serverAlias
	cfg.Profiles[profileAlias] = profile
	return Write(cfg)
}

func SetDefault(alias string) error {
	cfg := Read()
	if _, ok := cfg.Profiles[alias]; !ok {
		return fmt.Errorf("profile %q is not configured", alias)
	}
	cfg.DefaultProfile = alias
	return Write(cfg)
}

func SetDefaultValue(key string, value int) error {
	cfg := Read()
	switch key {
	case "timeout_ms":
		cfg.Defaults.TimeoutMS = intPtr(value)
	case "max_retries":
		cfg.Defaults.MaxRetries = intPtr(value)
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return Write(cfg)
}

func UnsetDefaultValue(key string) error {
	cfg := Read()
	switch key {
	case "timeout_ms":
		cfg.Defaults.TimeoutMS = nil
	case "max_retries":
		cfg.Defaults.MaxRetries = nil
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return Write(cfg)
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
		profile.DefaultServer = firstNonEmpty(profile.DefaultServer, "default")
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func intPtr(value int) *int { return &value }
