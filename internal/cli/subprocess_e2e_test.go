package cli

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSubprocessE2EAgainstMockpostmark(t *testing.T) {
	if os.Getenv("AGENT_POSTMARK_RUN_SUBPROCESS_E2E") != "1" {
		t.Skip("set AGENT_POSTMARK_RUN_SUBPROCESS_E2E=1 to build binaries and run mockpostmark subprocess e2e")
	}

	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	binDir := t.TempDir()
	agentBin := filepath.Join(binDir, "agent-postmark")
	mockBin := filepath.Join(binDir, "mockpostmark")

	runCmd(t, repoRoot, "go", "build", "-buildvcs=false", "-o", agentBin, "./cmd/agent-postmark")
	runCmd(t, repoRoot, "go", "build", "-buildvcs=false", "-o", mockBin, "./cmd/mockpostmark")

	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mock := exec.CommandContext(ctx, mockBin, "--addr", addr)
	mock.Dir = repoRoot
	mock.Env = os.Environ()
	if err := mock.Start(); err != nil {
		t.Fatalf("start mockpostmark: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		_ = mock.Wait()
	})
	waitForHealth(t, "http://"+addr+"/healthz")

	out := runAgent(t, repoRoot, agentBin, addr, "messages", "search", "--to", "user@example.com")
	if !strings.Contains(out, `"MessageID":"msg-1"`) || strings.Contains(out, "user@example.com") {
		t.Fatalf("unexpected message search output: %s", out)
	}
	out = runAgent(t, repoRoot, agentBin, addr, "investigate", "delivery", "--email", "user@example.com")
	if !strings.Contains(out, `"severity":"critical"`) {
		t.Fatalf("expected critical delivery finding, got: %s", out)
	}
}

func runAgent(t *testing.T, repoRoot, agentBin, addr string, args ...string) string {
	t.Helper()
	allArgs := append([]string{
		"--host", "http://" + addr,
		"--account-token", "account_mock",
		"--server-token", "server_mock",
	}, args...)
	return runCmd(t, repoRoot, agentBin, allArgs...)
}

func runCmd(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(os.TempDir(), "agent-postmark-go-build"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().String()
}

func waitForHealth(t *testing.T, url string) {
	t.Helper()
	client := http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("mockpostmark did not become healthy at %s", url)
}
