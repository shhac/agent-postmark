package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/shhac/agent-postmark/internal/config"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
)

func TestResolveUnknownServerNamesConfiguredAliases(t *testing.T) {
	config.SetConfigDir(t.TempDir())
	t.Setenv("AGENT_POSTMARK_SERVER", "")
	t.Setenv("AGENT_POSTMARK_SERVER_TOKEN", "")
	t.Setenv("POSTMARK_SERVER_TOKEN", "")
	if err := config.Write(&config.Config{Profiles: map[string]config.Profile{
		"acme": {Servers: map[string]config.ServerProfile{
			"prod":    {ServerID: 1},
			"sandbox": {ServerID: 2},
		}},
	}}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := resolve(&GlobalFlags{Profile: "acme", Server: "staging"})
	if err == nil {
		t.Fatal("resolve(--server staging) = nil error, want unknown-server error")
	}

	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is not an APIError: %v", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("fixable_by = %q, want agent", apiErr.FixableBy)
	}
	if !strings.Contains(apiErr.Message, "staging") {
		t.Fatalf("message %q does not name the bad alias", apiErr.Message)
	}
	for _, alias := range []string{"prod", "sandbox"} {
		if !strings.Contains(apiErr.Hint, alias) {
			t.Fatalf("hint %q does not list configured alias %q", apiErr.Hint, alias)
		}
	}
}

// A directly supplied --server-token means the alias is just a label, so an
// unconfigured name must not be rejected.
func TestResolveUnknownServerAllowedWithDirectToken(t *testing.T) {
	config.SetConfigDir(t.TempDir())
	t.Setenv("AGENT_POSTMARK_SERVER", "")
	t.Setenv("AGENT_POSTMARK_SERVER_TOKEN", "")
	t.Setenv("POSTMARK_SERVER_TOKEN", "")
	if err := config.Write(&config.Config{Profiles: map[string]config.Profile{
		"acme": {Servers: map[string]config.ServerProfile{"prod": {ServerID: 1}}},
	}}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := resolve(&GlobalFlags{Profile: "acme", Server: "staging", ServerToken: "tok"}); err != nil {
		t.Fatalf("resolve with direct server token: %v", err)
	}
}
