package cli

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
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
				return agenterrors.New("missing --email", agenterrors.FixableByAgent).
					WithHint("Provide the recipient address to search Postmark messages and bounces.")
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
				suppressionRaw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/list", stream), url.Values{"count": {"20"}, "offset": {"0"}})
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
