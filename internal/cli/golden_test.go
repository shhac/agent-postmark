package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenMessagesSearch(t *testing.T) {
	stdout, stderr, _ := runCLI(t, "messages", "search", "--to", "user@example.com")
	if stderr != "" {
		t.Fatalf("stderr = %s", stderr)
	}
	assertGolden(t, "messages_search.golden", stdout)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", name, string(want), got)
	}
}
