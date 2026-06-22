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
	raw := redactRaw([]byte(`{"account_token":true,"server_token":false,"token_count":3,"token_ratio":1.5,"maybe_token":null}`))
	got := string(raw)
	if strings.Contains(got, "[REDACTED]") || strings.Contains(got, "@redacted") {
		t.Fatalf("scalar token metadata should not be redacted: %s", got)
	}
	if !strings.Contains(got, `"account_token":true`) || !strings.Contains(got, `"server_token":false`) || !strings.Contains(got, `"token_count":3`) || !strings.Contains(got, `"token_ratio":1.5`) || !strings.Contains(got, `"maybe_token":null`) {
		t.Fatalf("scalar token metadata changed: %s", got)
	}
}

func TestRedactAllowsConfiguredTokenMetadataOnlyForBooleanShape(t *testing.T) {
	raw := redactRaw([]byte(`{"account_token_configured":"yes","server_token_configured":"yes","server_tokens_configured":{"prod":"yes","dev":true}}`))
	got := string(raw)
	if strings.Contains(got, "yes") {
		t.Fatalf("unsafe configured token metadata leaked: %s", got)
	}
	if !strings.Contains(got, `"account_token_configured":"[REDACTED]"`) || !strings.Contains(got, `"server_token_configured":"[REDACTED]"`) || !strings.Contains(got, `"server_tokens_configured":"[REDACTED]"`) {
		t.Fatalf("unsafe configured token metadata was not redacted: %s", got)
	}
}

func TestRedactStillRedactsTokenValues(t *testing.T) {
	raw := redactRaw([]byte(`{"account_token":"pm-account-value","server_token":"pm-server-value","api_secret":"pm-secret-value","ApiToken":"pm-api-value"}`))
	got := string(raw)
	if strings.Contains(got, "pm-account-value") || strings.Contains(got, "pm-server-value") || strings.Contains(got, "pm-secret-value") || strings.Contains(got, "pm-api-value") {
		t.Fatalf("token value leaked: %s", got)
	}
	if !strings.Contains(got, `"account_token":"[REDACTED]"`) || !strings.Contains(got, `"server_token":"[REDACTED]"`) || !strings.Contains(got, `"api_secret":"[REDACTED]"`) || !strings.Contains(got, `"ApiToken":"[REDACTED]"`) {
		t.Fatalf("token keys were not redacted: %s", got)
	}
}

func TestRedactStillRedactsTokenContainers(t *testing.T) {
	raw := redactRaw([]byte(`{"token":{"value":"pm-token-value"},"secrets":["pm-secret-value"],"Nested":{"server_token":["pm-server-value"]}}`))
	got := string(raw)
	if strings.Contains(got, "pm-token-value") || strings.Contains(got, "pm-secret-value") || strings.Contains(got, "pm-server-value") {
		t.Fatalf("token container leaked: %s", got)
	}
	if !strings.Contains(got, `"token":"[REDACTED]"`) || !strings.Contains(got, `"secrets":"[REDACTED]"`) || !strings.Contains(got, `"server_token":"[REDACTED]"`) {
		t.Fatalf("token containers were not redacted: %s", got)
	}
}

func TestRedactAllowsEmailContentContainers(t *testing.T) {
	raw := redactRaw([]byte(`{"HtmlBody":"<p>Hello</p>","TextBody":"Hello","Subject":"Welcome","From":"sender@example.com","To":"user@example.com","Cc":"cc@example.com","Bcc":"bcc@example.com","Headers":[{"Name":"X-Campaign","Value":"spring"}],"Metadata":{"campaign":"spring"},"Attachments":[{"Name":"invoice.pdf","Content":"base64-body"}]}`))
	got := string(raw)
	if strings.Contains(got, "[REDACTED]") || strings.Contains(got, "@redacted") {
		t.Fatalf("email content containers should not be redacted by default: %s", got)
	}
	if !strings.Contains(got, `"HtmlBody":"<p>Hello</p>"`) || !strings.Contains(got, `"Subject":"Welcome"`) || !strings.Contains(got, `"To":"user@example.com"`) || !strings.Contains(got, `"Headers":[`) || !strings.Contains(got, `"Attachments":[`) || !strings.Contains(got, `"Metadata":{`) {
		t.Fatalf("email content containers changed: %s", got)
	}
}

func TestRedactStillRedactsNestedSecretsInsideContent(t *testing.T) {
	raw := redactRaw([]byte(`{"Metadata":{"campaign":"spring","api_token":"secret"},"Attachments":[{"Name":"invoice.pdf","secret":"attachment-secret"}]}`))
	got := string(raw)
	if strings.Contains(got, "attachment-secret") || strings.Contains(got, `"api_token":"secret"`) {
		t.Fatalf("nested secret leaked: %s", got)
	}
	if !strings.Contains(got, `"api_token":"[REDACTED]"`) || !strings.Contains(got, `"secret":"[REDACTED]"`) {
		t.Fatalf("nested secret was not redacted: %s", got)
	}
	if !strings.Contains(got, `"campaign":"spring"`) || !strings.Contains(got, `"Name":"invoice.pdf"`) {
		t.Fatalf("safe content metadata changed: %s", got)
	}
}

func TestRedactWebhookURLUserinfo(t *testing.T) {
	raw := redactRaw([]byte(`{"Url":"https://postmark:password@example.com/hooks?stream=outbound","CallbackUrl":"https://postmark:password@example.com/other","Website":"https://postmark:password@example.com/public"}`))
	got := string(raw)
	if strings.Contains(got, "postmark:password") {
		t.Fatalf("url userinfo leaked: %s", got)
	}
	if !strings.Contains(got, `"Url":"https://[REDACTED]@example.com/hooks?stream=outbound"`) {
		t.Fatalf("url userinfo was not redacted cleanly: %s", got)
	}
	if !strings.Contains(got, `"CallbackUrl":"https://[REDACTED]@example.com/other"`) {
		t.Fatalf("callback url userinfo was not redacted cleanly: %s", got)
	}
	if !strings.Contains(got, `"Website":"https://[REDACTED]@example.com/public"`) {
		t.Fatalf("url-ish value userinfo was not redacted cleanly: %s", got)
	}
}

func TestRedactKeySensitivityWinsOverURLUserinfo(t *testing.T) {
	raw := redactRaw([]byte(`{"ApiToken":"https://postmark:password@example.com/token","Url":"https://postmark:password@example.com/hook"}`))
	got := string(raw)
	if strings.Contains(got, "postmark:password") || strings.Contains(got, "/token") {
		t.Fatalf("sensitive URL-shaped token leaked: %s", got)
	}
	if !strings.Contains(got, `"ApiToken":"[REDACTED]"`) {
		t.Fatalf("token key should be fully redacted: %s", got)
	}
	if !strings.Contains(got, `"Url":"https://[REDACTED]@example.com/hook"`) {
		t.Fatalf("non-sensitive URL userinfo should be redacted as URL: %s", got)
	}
}

func TestRedactRecordsArrayElementPaths(t *testing.T) {
	raw := redactRaw([]byte(`{"Items":[{"ApiToken":"a"},{"ApiToken":"b"}]}`))
	got := string(raw)
	if strings.Contains(got, `"ApiToken":"a"`) || strings.Contains(got, `"ApiToken":"b"`) {
		t.Fatalf("token leaked: %s", got)
	}
	if strings.Count(got, `"ApiToken":"[REDACTED]"`) != 2 {
		t.Fatalf("both array element tokens should be redacted: %s", got)
	}
	// The shared redactor records one note per masked occurrence, keyed by the
	// array-element path (a "[]" segment marks the array).
	if !strings.Contains(got, `"path":"Items[].ApiToken"`) {
		t.Fatalf("redacted array path not recorded: %s", got)
	}
}
