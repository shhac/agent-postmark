package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
)

type deliverySearchEvidence struct {
	MessageTotal int              `json:"message_total"`
	BounceTotal  int              `json:"bounce_total"`
	Messages     []map[string]any `json:"messages"`
	Bounces      []map[string]any `json:"bounces"`
}

func deliveryEvidence(msgRaw, bounceRaw json.RawMessage) deliverySearchEvidence {
	var msgPayload struct {
		TotalCount int               `json:"TotalCount"`
		Messages   []json.RawMessage `json:"Messages"`
	}
	var bouncePayload struct {
		TotalCount int               `json:"TotalCount"`
		Bounces    []json.RawMessage `json:"Bounces"`
	}
	_ = json.Unmarshal(msgRaw, &msgPayload)
	_ = json.Unmarshal(bounceRaw, &bouncePayload)
	out := deliverySearchEvidence{MessageTotal: msgPayload.TotalCount, BounceTotal: bouncePayload.TotalCount}
	for _, raw := range msgPayload.Messages {
		out.Messages = append(out.Messages, compactMap("Messages", raw))
	}
	for _, raw := range bouncePayload.Bounces {
		out.Bounces = append(out.Bounces, compactMap("Bounces", raw))
	}
	return out
}

func compactMap(resource string, raw json.RawMessage) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(compactListItem(resource, raw, false), &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func domainByIDOrName(ctx context.Context, resolved *resolvedContext, value string) (json.RawMessage, error) {
	if _, err := strconv.Atoi(value); err == nil {
		return resolved.Client.Get(ctx, api.AccountToken, "/domains/"+value, url.Values{})
	}
	raw, err := resolved.Client.Get(ctx, api.AccountToken, "/domains", url.Values{"count": {"100"}, "offset": {"0"}})
	if err != nil {
		return nil, err
	}
	for _, item := range rawEnvelopeList(raw, "Domains") {
		obj := rawObject(item)
		if strings.EqualFold(firstString(obj, "Name", "Domain"), value) {
			return item, nil
		}
	}
	return nil, agenterrors.New("domain not found: "+value, agenterrors.FixableByAgent).
		WithHint("Run 'agent-postmark domains list' to find the domain ID or exact domain name.")
}

func signaturesForDomain(raw json.RawMessage, domain string) []map[string]any {
	rows := []map[string]any{}
	for _, item := range rawEnvelopeList(raw, "SenderSignatures") {
		obj := compactMap("SenderSignatures", item)
		email := firstString(obj, "EmailAddress")
		if domain == "" || strings.HasSuffix(strings.ToLower(email), "@"+strings.ToLower(domain)) {
			rows = append(rows, obj)
		}
	}
	return rows
}

func compactRows(resource string, rows []json.RawMessage) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, compactMap(resource, row))
	}
	return out
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := obj[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

func numberValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}
