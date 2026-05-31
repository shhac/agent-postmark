package cli

import (
	"encoding/json"
	"testing"
)

func TestDeliveryFindingsNoActivitySuggestsStreamSearch(t *testing.T) {
	records := deliveryFindings("user@example.com", "outbound", deliverySearchEvidence{})
	assertRecord(t, records, "finding", "severity", "warning")
	assertRecord(t, records, "next_command", "command", "agent-postmark streams list --server <server-id>")
	assertRecord(t, records, "next_command", "command", "agent-postmark messages search --to <email> --stream <other-stream>")
}

func TestDeliveryFindingsInactiveBounceIsCritical(t *testing.T) {
	records := deliveryFindings("user@example.com", "outbound", deliverySearchEvidence{
		MessageTotal: 1,
		BounceTotal:  1,
		Bounces:      []map[string]any{{"Inactive": true}},
	})
	assertRecord(t, records, "finding", "severity", "critical")
	assertRecord(t, records, "next_command", "command", "agent-postmark suppressions check <email>")
}

func TestDomainHealthFindingsSuggestsVerificationCommands(t *testing.T) {
	records := domainHealthFindings(map[string]any{
		"DKIMVerified":             false,
		"SPFVerified":              false,
		"ReturnPathDomainVerified": true,
	}, nil)
	assertRecord(t, records, "finding", "severity", "warning")
	assertRecord(t, records, "next_command", "command", "agent-postmark domains verify-dkim <domain-id> --yes")
	assertRecord(t, records, "next_command", "command", "agent-postmark domains verify-spf <domain-id> --yes")
}

func TestStreamHealthFindingsUsesSharedWebhookCoverageHint(t *testing.T) {
	stats := json.RawMessage(`{"InactiveMails":0}`)
	bounces := json.RawMessage(`{"TotalCount":0,"Bounces":[]}`)
	webhooks := json.RawMessage(`{"TotalCount":1,"Webhooks":[{"ID":701,"Triggers":{"Delivery":true}}]}`)
	suppressions := json.RawMessage(`{"TotalCount":0,"Suppressions":[]}`)

	records := streamHealthFindings("outbound", stats, bounces, webhooks, suppressions)
	assertRecord(t, records, "finding", "severity", "warning")
	assertRecord(t, records, "next_command", "command", "agent-postmark webhooks health")
}

func TestWebhookCoverageFindingOkAndWarning(t *testing.T) {
	ok := webhookCoverageFinding(map[string]int{"delivery": 1, "bounce": 1})
	if ok["severity"] != "ok" {
		t.Fatalf("severity = %v", ok["severity"])
	}
	warning := webhookCoverageFinding(map[string]int{"delivery": 1, "bounce": 0})
	if warning["severity"] != "warning" {
		t.Fatalf("severity = %v", warning["severity"])
	}
}

func assertRecord(t *testing.T, records []evidenceRecord, recordType, key string, value any) {
	t.Helper()
	for _, record := range records {
		if record["type"] == recordType && record[key] == value {
			return
		}
	}
	t.Fatalf("missing %s record with %s=%v in %#v", recordType, key, value, records)
}
