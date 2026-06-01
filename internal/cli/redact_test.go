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
