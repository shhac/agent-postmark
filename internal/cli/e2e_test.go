package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shhac/agent-postmark/internal/api"
	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/mockpostmark"
	"github.com/shhac/agent-postmark/internal/output"
)

type mockDoer struct {
	handler http.Handler
	calls   int
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	m.calls++
	rec := httptest.NewRecorder()
	m.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func runCLI(t *testing.T, args ...string) (string, string, *mockDoer) {
	t.Helper()
	config.SetConfigDir(t.TempDir())
	t.Setenv("AGENT_POSTMARK_PROFILE", "")
	t.Setenv("AGENT_POSTMARK_ACCOUNT_TOKEN", "")
	t.Setenv("AGENT_POSTMARK_SERVER_TOKEN", "")
	t.Setenv("POSTMARK_ACCOUNT_TOKEN", "")
	t.Setenv("POSTMARK_SERVER_TOKEN", "")

	doer := &mockDoer{handler: mockpostmark.NewServer()}
	oldFactory := newAPIClient
	newAPIClient = func(baseURL, accountToken, serverToken string) *api.Client {
		client := api.New(baseURL, accountToken, serverToken)
		client.HTTPClient = doer
		client.Sleep = func(d time.Duration) {}
		return client
	}
	t.Cleanup(func() { newAPIClient = oldFactory })

	var stdout, stderr bytes.Buffer
	restore := output.SetWriters(&stdout, &stderr)
	t.Cleanup(restore)

	cmd := newRootCmd("test")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{
		"--host", "http://mock.postmark",
		"--account-token", "account_mock",
		"--server-token", "server_mock",
	}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return stdout.String(), stderr.String(), doer
}

func TestHelpDoesNotAdvertiseAuthCommand(t *testing.T) {
	cmd := newRootCmd("test")
	var foundProfiles, foundAuth bool
	for _, child := range cmd.Commands() {
		switch child.Name() {
		case "profiles":
			foundProfiles = true
		case "auth":
			foundAuth = true
		}
	}
	if !foundProfiles {
		t.Fatal("profiles command not registered")
	}
	if foundAuth {
		t.Fatal("auth should remain hidden and not be registered as a visible command")
	}
}

func TestHiddenAuthAliasExecutesProfiles(t *testing.T) {
	config.SetConfigDir(t.TempDir())
	var stdout, stderr bytes.Buffer
	restore := output.SetWriters(&stdout, &stderr)
	defer restore()
	oldArgs := os.Args
	os.Args = []string{"agent-postmark", "auth", "list"}
	defer func() { os.Args = oldArgs }()

	if err := Execute("test"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %s", got)
	}
}

func TestServersListStreamsCompactNDJSON(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "servers", "list")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"ID":101`) || !strings.Contains(stdout, `"Name":"Production"`) {
		t.Fatalf("stdout = %s", stdout)
	}
	if strings.Contains(stdout, "HtmlBody") {
		t.Fatalf("server list should be compact, got %s", stdout)
	}
}

func TestMessagesSearchCompactsAndRedacts(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "messages", "search", "--to", "user@example.com")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"MessageID":"msg-1"`) || !strings.Contains(stdout, `"Status":"Sent"`) {
		t.Fatalf("stdout = %s", stdout)
	}
	if strings.Contains(stdout, "user@example.com") || strings.Contains(stdout, "secret body") || strings.Contains(stdout, "Welcome") {
		t.Fatalf("sensitive content leaked: %s", stdout)
	}
}

func TestInvestigateDeliveryEmitsCriticalBounceFinding(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "investigate", "delivery", "--email", "user@example.com")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"severity":"critical"`) || !strings.Contains(stdout, "Recipient appears inactive") {
		t.Fatalf("stdout = %s", stdout)
	}
	if strings.Contains(stdout, "user@example.com") || strings.Contains(stdout, "secret body") {
		t.Fatalf("sensitive content leaked: %s", stdout)
	}
}

func TestMutationRequiresYes(t *testing.T) {
	_, stderr, doer := runCLI(t, "domains", "verify-dkim", "501")
	if !strings.Contains(stderr, `"mutation requires --yes"`) {
		t.Fatalf("stderr = %s", stderr)
	}
	if doer.calls != 0 {
		t.Fatalf("mutation should not call API without --yes, got %d calls", doer.calls)
	}
}

func TestWebhookHealthFinding(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "webhooks", "health")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"type":"finding"`) || !strings.Contains(stdout, `"severity":"ok"`) {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestInvestigateBounce(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "investigate", "bounce", "9001")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"object":"bounce"`) || !strings.Contains(stdout, `"severity":"critical"`) {
		t.Fatalf("stdout = %s", stdout)
	}
	if strings.Contains(stdout, "user@example.com") {
		t.Fatalf("sensitive content leaked: %s", stdout)
	}
}

func TestInvestigateDomainHealth(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "investigate", "domain-health", "example.com")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"object":"domain"`) || !strings.Contains(stdout, `"severity":"ok"`) {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestInvestigateStreamHealth(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "investigate", "stream-health", "--stream", "outbound")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"object":"delivery_stats"`) || !strings.Contains(stdout, `"object":"suppressions"`) {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestInvestigateWebhookHealth(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "investigate", "webhook-health")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	if !strings.Contains(stdout, `"object":"webhooks"`) || !strings.Contains(stdout, `"severity":"ok"`) {
		t.Fatalf("stdout = %s", stdout)
	}
}
