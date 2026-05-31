package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

type errorPayload struct {
	Error     string `json:"error"`
	FixableBy string `json:"fixable_by"`
	Hint      string `json:"hint"`
}

func parseErrorPayload(t *testing.T, stderr string) errorPayload {
	t.Helper()
	var payload errorPayload
	if err := json.Unmarshal([]byte(stderr), &payload); err != nil {
		t.Fatalf("stderr is not JSON: %v\n%s", err, stderr)
	}
	if payload.Error == "" {
		t.Fatalf("missing error field: %s", stderr)
	}
	if payload.FixableBy == "" {
		t.Fatalf("missing fixable_by field: %s", stderr)
	}
	return payload
}

func TestErrorContractRawAPIPathHint(t *testing.T) {
	_, stderr, _ := runCLI(t, "api", "get", "bounces")
	payload := parseErrorPayload(t, stderr)
	if payload.FixableBy != "agent" {
		t.Fatalf("fixable_by = %q", payload.FixableBy)
	}
	if !strings.Contains(payload.Hint, "/bounces") {
		t.Fatalf("hint = %q", payload.Hint)
	}
}

func TestErrorContractAuthUsesProfilesHint(t *testing.T) {
	_, stderr, _ := runCLI(t, "--server-token", "invalid", "messages", "search", "--to", "user@example.com")
	payload := parseErrorPayload(t, stderr)
	if payload.FixableBy != "human" {
		t.Fatalf("fixable_by = %q", payload.FixableBy)
	}
	if !strings.Contains(payload.Hint, "profiles check") {
		t.Fatalf("hint = %q", payload.Hint)
	}
	if strings.Contains(payload.Hint, "auth check") {
		t.Fatalf("hint should not advertise auth alias: %q", payload.Hint)
	}
}

func TestErrorContractConfigHint(t *testing.T) {
	_, stderr, _ := runCLI(t, "config", "set", "max_retries", "nope")
	payload := parseErrorPayload(t, stderr)
	if payload.FixableBy != "agent" {
		t.Fatalf("fixable_by = %q", payload.FixableBy)
	}
	if !strings.Contains(payload.Hint, "must be integers") {
		t.Fatalf("hint = %q", payload.Hint)
	}
}
