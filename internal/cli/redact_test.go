package cli

import (
	"strings"
	"testing"
)

func TestRedactAllowsConfiguredTokenMetadata(t *testing.T) {
	raw := redactRaw([]byte(`{"account_token_configured":true,"server_tokens_configured":{"prod":true},"server_token_configured":true}`))
	got := string(raw)
	if strings.Contains(got, "[REDACTED]") || strings.Contains(got, "@redacted") {
		t.Fatalf("configured token metadata should not be redacted: %s", got)
	}
}

func TestRedactAllowsScalarTokenPresenceMetadata(t *testing.T) {
	raw := redactRaw([]byte(`{"account_token":true,"server_token":false,"token_count":3}`))
	got := string(raw)
	if strings.Contains(got, "[REDACTED]") || strings.Contains(got, "@redacted") {
		t.Fatalf("scalar token metadata should not be redacted: %s", got)
	}
	if !strings.Contains(got, `"account_token":true`) || !strings.Contains(got, `"server_token":false`) || !strings.Contains(got, `"token_count":3`) {
		t.Fatalf("scalar token metadata changed: %s", got)
	}
}

func TestRedactStillRedactsTokenValues(t *testing.T) {
	raw := redactRaw([]byte(`{"account_token":"pm-account-value","server_token":"pm-server-value","api_secret":"pm-secret-value"}`))
	got := string(raw)
	if strings.Contains(got, "pm-account-value") || strings.Contains(got, "pm-server-value") || strings.Contains(got, "pm-secret-value") {
		t.Fatalf("token value leaked: %s", got)
	}
	if !strings.Contains(got, `"account_token":"[REDACTED]"`) || !strings.Contains(got, `"server_token":"[REDACTED]"`) || !strings.Contains(got, `"api_secret":"[REDACTED]"`) {
		t.Fatalf("token keys were not redacted: %s", got)
	}
}

func TestRedactStillRedactsSensitiveContainers(t *testing.T) {
	raw := redactRaw([]byte(`{"Headers":[{"Name":"X-Secret","Value":"secret"}],"Metadata":{"token":"secret"},"Attachments":[{"Name":"invoice.pdf"}]}`))
	got := string(raw)
	if strings.Contains(got, "secret") || strings.Contains(got, "invoice.pdf") {
		t.Fatalf("sensitive container leaked: %s", got)
	}
	if !strings.Contains(got, `"Headers":"[REDACTED]"`) || !strings.Contains(got, `"Metadata":"[REDACTED]"`) || !strings.Contains(got, `"Attachments":"[REDACTED]"`) {
		t.Fatalf("sensitive containers were not redacted: %s", got)
	}
}
