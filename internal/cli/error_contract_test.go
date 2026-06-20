package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/shhac/agent-postmark/internal/output"
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

// TestExecuteRendersBubbledErrorStructured guards the top-level error sink:
// errors that escape every command (here, cobra's unknown-command error) must
// reach stderr as a single structured {error,fixable_by} line, never plain
// text. This path is only reachable through the package-level Execute, which
// reads os.Args, so the other tests' runCLI helper does not cover it.
func TestExecuteRendersBubbledErrorStructured(t *testing.T) {
	var stderr bytes.Buffer
	restore := output.SetWriters(nil, &stderr)
	t.Cleanup(restore)

	oldArgs := os.Args
	os.Args = []string{"agent-postmark", "boguscmd"}
	t.Cleanup(func() { os.Args = oldArgs })

	if err := Execute("test"); err == nil {
		t.Fatal("Execute(boguscmd) = nil error, want non-nil")
	}

	out := strings.TrimRight(stderr.String(), "\n")
	if strings.Contains(out, "\n") {
		t.Fatalf("stderr is not a single line:\n%s", out)
	}
	payload := parseErrorPayload(t, out)
	if payload.FixableBy != "agent" {
		t.Fatalf("fixable_by = %q, want agent", payload.FixableBy)
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
