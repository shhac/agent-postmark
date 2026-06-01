package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shhac/agent-postmark/internal/output"
)

func TestWriteEnvelopeListWithPageAppliesLocalPaginationWithoutTotalCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWriters(&stdout, &stderr)
	defer restore()

	raw := json.RawMessage(`{"Suppressions":[{"EmailAddress":"a@example.com"},{"EmailAddress":"b@example.com"},{"EmailAddress":"c@example.com"}]}`)
	if err := writeEnvelopeListWithPage(raw, "Suppressions", 1, 1, "", false); err != nil {
		t.Fatalf("writeEnvelopeListWithPage: %v", err)
	}

	got := stdout.String()
	if strings.Count(strings.TrimSpace(got), "\n")+1 != 2 {
		t.Fatalf("expected one item plus pagination, got %q", got)
	}
	if strings.Contains(got, "a@example.com") || strings.Contains(got, "c@example.com") {
		t.Fatalf("local pagination leaked items outside requested page: %s", got)
	}
	if !strings.Contains(got, `"EmailAddress":"[REDACTED]"`) || !strings.Contains(got, `"total_items":3`) || !strings.Contains(got, `"next_offset":2`) {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestWriteEnvelopeListWithPageTrustsRemotePaginationWithTotalCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWriters(&stdout, &stderr)
	defer restore()

	raw := json.RawMessage(`{"TotalCount":3,"Suppressions":[{"EmailAddress":"b@example.com"}]}`)
	if err := writeEnvelopeListWithPage(raw, "Suppressions", 1, 1, "", false); err != nil {
		t.Fatalf("writeEnvelopeListWithPage: %v", err)
	}

	got := stdout.String()
	if strings.Count(strings.TrimSpace(got), "\n")+1 != 2 {
		t.Fatalf("expected one remote-paginated item plus pagination, got %q", got)
	}
	if !strings.Contains(got, `"total_items":3`) || !strings.Contains(got, `"next_offset":2`) {
		t.Fatalf("unexpected pagination metadata: %s", got)
	}
}
