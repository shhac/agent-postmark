package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func registerInvestigate(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "investigate", Short: "Higher-level Postmark triage workflows"}
	registerInvestigateDelivery(cmd, globals)
	registerInvestigateBounce(cmd, globals)
	registerInvestigateDomainHealth(cmd, globals)
	registerInvestigateStreamHealth(cmd, globals)
	registerInvestigateWebhookHealth(cmd, globals)
	root.AddCommand(cmd)
}

func registerInvestigateDelivery(parent *cobra.Command, globals *GlobalFlags) {
	var email, fromDate, toDate string
	delivery := &cobra.Command{
		Use:   "delivery --email <address>",
		Short: "Collect message and bounce evidence for a recipient",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				output.WriteError(output.Stderr(), agenterrors.New("missing --email", agenterrors.FixableByAgent).
					WithHint("Provide the recipient address to search Postmark messages and bounces."))
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{"count": {"20"}, "offset": {"0"}, "messagestream": {resolved.MessageStream}}
				addIfSet(q, "recipient", email)
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				msgRaw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/outbound", q)
				if err != nil {
					return err
				}
				bq := url.Values{"count": {"20"}, "offset": {"0"}, "messagestream": {resolved.MessageStream}}
				addIfSet(bq, "emailFilter", email)
				addIfSet(bq, "fromdate", fromDate)
				addIfSet(bq, "todate", toDate)
				bounceRaw, err := resolved.Client.Get(ctx, api.ServerToken, "/bounces", bq)
				if err != nil {
					return err
				}
				evidence := deliveryEvidence(msgRaw, bounceRaw)
				records := []evidenceRecord{
					entityRecord("messages_search", nil, evidence.Messages),
					entityRecord("bounces_search", nil, evidence.Bounces),
				}
				records = append(records, deliveryFindings(email, resolved.MessageStream, evidence)...)
				return writeEvidence(records)
			})
		},
	}
	delivery.Flags().StringVar(&email, "email", "", "Recipient email address")
	delivery.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	delivery.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	parent.AddCommand(delivery)
}

func registerInvestigateBounce(parent *cobra.Command, globals *GlobalFlags) {
	parent.AddCommand(&cobra.Command{
		Use:   "bounce <bounce-id>",
		Short: "Explain a bounce and its related message state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/bounces/"+args[0], url.Values{})
				if err != nil {
					return err
				}
				bounce := rawObject(raw)
				records := []evidenceRecord{entityRecord("bounce", bounce["ID"], bounce)}
				if messageID := firstString(bounce, "MessageID"); messageID != "" {
					msgRaw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/messages/outbound/%s/details", messageID), url.Values{})
					if err == nil {
						records = append(records, entityRecord("message", messageID, compactMap("Messages", msgRaw)))
					}
				}
				records = append(records, bounceFindings(bounce)...)
				records = append(records, nextCommandRecord("agent-postmark suppressions check <email>", "Check whether the bounced recipient is currently suppressed."))
				return writeEvidence(records)
			})
		},
	})
}

func registerInvestigateDomainHealth(parent *cobra.Command, globals *GlobalFlags) {
	parent.AddCommand(&cobra.Command{
		Use:   "domain-health <domain-id-or-name>",
		Short: "Inspect sender domain and signature health",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				domainRaw, err := domainByIDOrName(ctx, resolved, args[0])
				if err != nil {
					return err
				}
				domain := rawObject(domainRaw)
				signatureRaw, err := resolved.Client.Get(ctx, api.AccountToken, "/senders", url.Values{"count": {"100"}, "offset": {"0"}})
				if err != nil {
					return err
				}
				signatures := signaturesForDomain(signatureRaw, firstString(domain, "Name", "Domain"))
				records := []evidenceRecord{
					entityRecord("domain", domain["ID"], compactMap("Domains", domainRaw)),
					entityRecord("sender_signatures", nil, signatures),
				}
				records = append(records, domainHealthFindings(domain, signatures)...)
				return writeEvidence(records)
			})
		},
	})
}

func registerInvestigateStreamHealth(parent *cobra.Command, globals *GlobalFlags) {
	var stream string
	cmd := &cobra.Command{
		Use:   "stream-health",
		Short: "Summarize delivery, bounce, webhook, and suppression health for a stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if stream == "" {
					stream = resolved.MessageStream
				}
				statsRaw, err := resolved.Client.Get(ctx, api.ServerToken, "/deliverystats", url.Values{})
				if err != nil {
					return err
				}
				bounceRaw, err := resolved.Client.Get(ctx, api.ServerToken, "/bounces", url.Values{"count": {"20"}, "offset": {"0"}, "messagestream": {stream}})
				if err != nil {
					return err
				}
				webhooksRaw, err := resolved.Client.Get(ctx, api.ServerToken, "/webhooks", url.Values{})
				if err != nil {
					return err
				}
				suppressionRaw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/dump", stream), url.Values{"count": {"20"}, "offset": {"0"}})
				if err != nil {
					return err
				}
				records := []evidenceRecord{
					entityRecord("delivery_stats", stream, rawObject(statsRaw)),
					entityRecord("recent_bounces", stream, compactRows("Bounces", rawEnvelopeList(bounceRaw, "Bounces"))),
					entityRecord("webhooks", stream, compactRows("Webhooks", rawEnvelopeList(webhooksRaw, "Webhooks"))),
					entityRecord("suppressions", stream, compactRows("Suppressions", rawEnvelopeList(suppressionRaw, "Suppressions"))),
				}
				records = append(records, streamHealthFindings(stream, statsRaw, bounceRaw, webhooksRaw, suppressionRaw)...)
				return writeEvidence(records)
			})
		},
	}
	cmd.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	parent.AddCommand(cmd)
}

func registerInvestigateWebhookHealth(parent *cobra.Command, globals *GlobalFlags) {
	parent.AddCommand(&cobra.Command{
		Use:   "webhook-health",
		Short: "Summarize webhook delivery/bounce/inbound trigger coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/webhooks", url.Values{})
				if err != nil {
					return err
				}
				return writeWebhookEvidence(raw)
			})
		},
	})
}

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

func deliveryFindings(email, stream string, evidence deliverySearchEvidence) []evidenceRecord {
	findings := []evidenceRecord{}
	if evidence.MessageTotal == 0 && evidence.BounceTotal == 0 {
		return append(findings,
			findingRecord("warning", "No outbound messages or bounces were found for this recipient in the selected window.", map[string]any{
				"email":  email,
				"stream": stream,
			}),
			nextCommandRecord("agent-postmark streams list --server <server-id>", "Confirm the stream used for this message."),
			nextCommandRecord("agent-postmark messages search --to <email> --stream <other-stream>", "Retry in another likely message stream."),
		)
	}
	if evidence.MessageTotal > 0 {
		findings = append(findings, findingRecord("info", "Outbound message activity was found for this recipient.", map[string]any{
			"message_total": evidence.MessageTotal,
		}))
		findings = append(findings, nextCommandRecord("agent-postmark messages get <message-id>", "Inspect the most relevant message details."))
	}
	if evidence.BounceTotal > 0 {
		severity := "warning"
		summary := "Bounce activity was found for this recipient."
		for _, bounce := range evidence.Bounces {
			if inactive, ok := bounce["Inactive"].(bool); ok && inactive {
				severity = "critical"
				summary = "Recipient appears inactive due to a bounce; future delivery may be suppressed."
				break
			}
		}
		findings = append(findings, findingRecord(severity, summary, map[string]any{
			"bounce_total": evidence.BounceTotal,
		}))
		findings = append(findings, nextCommandRecord("agent-postmark bounces get <bounce-id>", "Inspect the bounce type and inactive state."))
		findings = append(findings, nextCommandRecord("agent-postmark suppressions check <email>", "Check whether the recipient is currently suppressed."))
	}
	return findings
}

func bounceFindings(bounce map[string]any) []evidenceRecord {
	severity := "info"
	summary := "Bounce record found."
	if inactive, _ := bounce["Inactive"].(bool); inactive {
		severity = "critical"
		summary = "Recipient is inactive because of this bounce; future delivery may be suppressed."
	} else if bounce["Type"] != nil {
		severity = "warning"
		summary = "Bounce record found; inspect type and can-activate state before retrying delivery."
	}
	return []evidenceRecord{findingRecord(severity, summary, map[string]any{
		"type":         bounce["Type"],
		"name":         bounce["Name"],
		"inactive":     bounce["Inactive"],
		"can_activate": bounce["CanActivate"],
	})}
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

func domainHealthFindings(domain map[string]any, signatures []map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	checks := map[string]bool{
		"dkim":        boolValue(domain["DKIMVerified"]),
		"spf":         boolValue(domain["SPFVerified"]),
		"return_path": boolValue(domain["ReturnPathDomainVerified"]),
	}
	if checks["dkim"] && checks["spf"] && checks["return_path"] {
		records = append(records, findingRecord("ok", "Domain authentication checks are verified.", map[string]any{"checks": checks}))
	} else {
		records = append(records, findingRecord("warning", "One or more domain authentication checks are not verified.", map[string]any{"checks": checks}))
		if !checks["dkim"] {
			records = append(records, nextCommandRecord("agent-postmark domains verify-dkim <domain-id> --yes", "Ask Postmark to re-check DKIM after DNS is corrected."))
		}
		if !checks["spf"] {
			records = append(records, nextCommandRecord("agent-postmark domains verify-spf <domain-id> --yes", "Ask Postmark to re-check SPF after DNS is corrected."))
		}
	}
	if len(signatures) == 0 {
		records = append(records, findingRecord("warning", "No sender signatures were found for this domain.", nil))
	}
	return records
}

func streamHealthFindings(stream string, statsRaw, bounceRaw, webhooksRaw, suppressionRaw json.RawMessage) []evidenceRecord {
	records := []evidenceRecord{}
	stats := rawObject(statsRaw)
	inactiveMails := numberValue(stats["InactiveMails"])
	if inactiveMails > 0 {
		records = append(records, findingRecord("warning", "Stream has inactive mail count in delivery stats.", map[string]any{"stream": stream, "inactive_mails": inactiveMails}))
	}
	if bounceTotal := rawTotal(bounceRaw); bounceTotal > 0 {
		records = append(records, findingRecord("warning", "Recent bounces were found for this stream.", map[string]any{"stream": stream, "bounce_total": bounceTotal}))
		records = append(records, nextCommandRecord("agent-postmark bounces list --stream "+stream, "Inspect recent bounces by type and inactive state."))
	}
	if suppressionTotal := rawTotal(suppressionRaw); suppressionTotal > 0 {
		records = append(records, findingRecord("info", "Suppressions exist for this stream.", map[string]any{"stream": stream, "suppression_total": suppressionTotal}))
	}
	coverage := webhookCoverage(rawEnvelopeList(webhooksRaw, "Webhooks"))
	if coverage["delivery"] == 0 || coverage["bounce"] == 0 {
		records = append(records, findingRecord("warning", "Webhook coverage is missing delivery or bounce triggers.", map[string]any{"coverage": coverage}))
		records = append(records, nextCommandRecord("agent-postmark webhooks health", "Inspect webhook trigger coverage."))
	} else {
		records = append(records, findingRecord("ok", "Webhook coverage includes delivery and bounce triggers.", map[string]any{"coverage": coverage}))
	}
	if len(records) == 0 {
		records = append(records, findingRecord("ok", "No obvious stream health issues found in delivery stats, bounces, suppressions, or webhooks.", map[string]any{"stream": stream}))
	}
	return records
}

func writeWebhookEvidence(raw json.RawMessage) error {
	rows := rawEnvelopeList(raw, "Webhooks")
	records := []evidenceRecord{entityRecord("webhooks", nil, compactRows("Webhooks", rows))}
	coverage := webhookCoverage(rows)
	if coverage["delivery"] == 0 || coverage["bounce"] == 0 {
		records = append(records, findingRecord("warning", "Webhook coverage is missing delivery or bounce triggers.", map[string]any{"coverage": coverage}))
	} else {
		records = append(records, findingRecord("ok", "Webhook coverage includes delivery and bounce triggers.", map[string]any{"coverage": coverage}))
	}
	return writeEvidence(records)
}

func compactRows(resource string, rows []json.RawMessage) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, compactMap(resource, row))
	}
	return out
}

func webhookCoverage(rows []json.RawMessage) map[string]int {
	coverage := map[string]int{"delivery": 0, "bounce": 0, "inbound": 0, "spam_complaint": 0}
	for _, row := range rows {
		obj := rawObject(row)
		triggers, _ := obj["Triggers"].(map[string]any)
		for key, value := range triggers {
			if !boolValue(value) {
				continue
			}
			switch key {
			case "Delivery":
				coverage["delivery"]++
			case "Bounce":
				coverage["bounce"]++
			case "Inbound":
				coverage["inbound"]++
			case "SpamComplaint":
				coverage["spam_complaint"]++
			}
		}
	}
	return coverage
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
