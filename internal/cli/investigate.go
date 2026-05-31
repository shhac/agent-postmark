package cli

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func registerInvestigate(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "investigate", Short: "Higher-level Postmark triage workflows"}
	var email, fromDate, toDate string
	cmd.AddCommand(&cobra.Command{
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
				writer := output.NewNDJSONWriter(output.Stdout())
				_ = writer.WriteItem(map[string]any{"type": "entity", "object": "messages_search", "data": evidence.Messages})
				_ = writer.WriteItem(map[string]any{"type": "entity", "object": "bounces_search", "data": evidence.Bounces})
				for _, finding := range deliveryFindings(email, resolved.MessageStream, evidence) {
					_ = writer.WriteItem(finding)
				}
				return nil
			})
		},
	})
	cmd.Commands()[0].Flags().StringVar(&email, "email", "", "Recipient email address")
	cmd.Commands()[0].Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	cmd.Commands()[0].Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	root.AddCommand(cmd)
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
	return out
}

func deliveryFindings(email, stream string, evidence deliverySearchEvidence) []map[string]any {
	findings := []map[string]any{}
	if evidence.MessageTotal == 0 && evidence.BounceTotal == 0 {
		return append(findings, finding("warning", "No outbound messages or bounces were found for this recipient in the selected window.", map[string]any{
			"email":  email,
			"stream": stream,
			"next_commands": []string{
				"agent-postmark streams list --server <server-id>",
				"agent-postmark messages search --to <email> --stream <other-stream>",
			},
		}))
	}
	if evidence.MessageTotal > 0 {
		findings = append(findings, finding("info", "Outbound message activity was found for this recipient.", map[string]any{
			"message_total": evidence.MessageTotal,
			"next_commands": []string{"agent-postmark messages get <message-id>"},
		}))
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
		findings = append(findings, finding(severity, summary, map[string]any{
			"bounce_total": evidence.BounceTotal,
			"next_commands": []string{
				"agent-postmark bounces get <bounce-id>",
				"agent-postmark suppressions check <email>",
			},
		}))
	}
	return findings
}

func finding(severity, summary string, data map[string]any) map[string]any {
	return map[string]any{
		"type":     "finding",
		"severity": severity,
		"summary":  summary,
		"data":     data,
	}
}
