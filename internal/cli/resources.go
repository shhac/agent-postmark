package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func registerResources(root *cobra.Command, globals *GlobalFlags) {
	registerServers(root, globals)
	registerStreams(root, globals)
	registerDomains(root, globals)
	registerSignatures(root, globals)
	registerWebhooks(root, globals)
	registerMessages(root, globals)
	registerBounces(root, globals)
	registerSuppressions(root, globals)
	registerStats(root, globals)
}

func registerServers(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "servers", Short: "List and get Postmark servers"}
	cmd.AddCommand(accountListCommand("list", "List servers", globals, "/servers", "Servers"))
	cmd.AddCommand(accountGetCommand("get <server-id>", "Get a server", globals, "/servers/%s"))
	root.AddCommand(cmd)
}

func registerStreams(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "streams", Short: "List and get message streams for a server"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List message streams",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if err := requireServer(resolved); err != nil {
					return err
				}
				raw, err := resolved.Client.Get(ctx, api.AccountToken, fmt.Sprintf("/servers/%d/message-streams", resolved.ServerID), url.Values{})
				if err != nil {
					return err
				}
				return writeEnvelopeList(raw, "MessageStreams", globals.Format, globals.Full)
			})
		},
	})
	cmd.AddCommand(accountGetCommand("get <stream-id>", "Get a message stream", globals, "/message-streams/%s"))
	root.AddCommand(cmd)
}

func registerDomains(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "domains", Short: "List and get sender domains"}
	cmd.AddCommand(accountListCommand("list", "List domains", globals, "/domains", "Domains"))
	cmd.AddCommand(accountGetCommand("get <domain-id>", "Get domain details", globals, "/domains/%s"))
	cmd.AddCommand(accountPostCommand("verify-dkim <domain-id>", "Verify DKIM for a domain", globals, "/domains/%s/verifyDkim", "Domain verification calls Postmark and may update verification state."))
	cmd.AddCommand(accountPostCommand("verify-spf <domain-id>", "Verify SPF for a domain", globals, "/domains/%s/verifySPF", "Domain verification calls Postmark and may update verification state."))
	root.AddCommand(cmd)
}

func registerSignatures(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "signatures", Short: "List and get sender signatures"}
	cmd.AddCommand(accountListCommand("list", "List sender signatures", globals, "/senders", "SenderSignatures"))
	cmd.AddCommand(accountGetCommand("get <signature-id>", "Get sender signature", globals, "/senders/%s"))
	root.AddCommand(cmd)
}

func registerWebhooks(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "webhooks", Short: "List and get webhooks"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List webhooks for the current server token",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/webhooks", url.Values{})
				if err != nil {
					return err
				}
				return writeEnvelopeList(raw, "Webhooks", globals.Format, globals.Full)
			})
		},
	})
	cmd.AddCommand(serverGetCommand("get <webhook-id>", "Get webhook", globals, "/webhooks/%s"))
	cmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Summarize webhook delivery/bounce/inbound trigger coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/webhooks", url.Values{})
				if err != nil {
					return err
				}
				return writeWebhookHealth(raw, globals.Format)
			})
		},
	})
	root.AddCommand(cmd)
}

func registerMessages(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "messages", Short: "Search and inspect outbound/inbound messages"}
	var recipient, from, tag, status, fromDate, toDate, stream string
	var count, offset int
	search := &cobra.Command{
		Use:   "search",
		Short: "Search outbound messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{}
				q.Set("count", strconv.Itoa(count))
				q.Set("offset", strconv.Itoa(offset))
				addIfSet(q, "recipient", recipient)
				addIfSet(q, "fromemail", from)
				addIfSet(q, "tag", tag)
				addIfSet(q, "status", status)
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				if stream == "" {
					stream = resolved.MessageStream
				}
				addIfSet(q, "messagestream", stream)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/outbound", q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Messages", offset, count, globals.Format, globals.Full)
			})
		},
	}
	search.Flags().StringVar(&recipient, "to", "", "Recipient email filter")
	search.Flags().StringVar(&from, "from", "", "From email filter")
	search.Flags().StringVar(&tag, "tag", "", "Tag filter")
	search.Flags().StringVar(&status, "status", "", "Outbound status filter")
	search.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	search.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	search.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	addCountOffsetFlags(search, &count, &offset)
	cmd.AddCommand(search)

	var inboundFrom, inboundRecipient, inboundSubject, mailboxHash string
	var inboundCount, inboundOffset int
	inboundSearch := &cobra.Command{
		Use:   "inbound-search",
		Short: "Search inbound messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{}
				q.Set("count", strconv.Itoa(inboundCount))
				q.Set("offset", strconv.Itoa(inboundOffset))
				addIfSet(q, "fromemail", inboundFrom)
				addIfSet(q, "recipient", inboundRecipient)
				addIfSet(q, "subject", inboundSubject)
				addIfSet(q, "mailboxhash", mailboxHash)
				addIfSet(q, "status", status)
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/inbound", q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "InboundMessages", inboundOffset, inboundCount, globals.Format, globals.Full)
			})
		},
	}
	inboundSearch.Flags().StringVar(&inboundFrom, "from", "", "Inbound from email filter")
	inboundSearch.Flags().StringVar(&inboundRecipient, "to", "", "Inbound recipient filter")
	inboundSearch.Flags().StringVar(&inboundSubject, "subject", "", "Subject filter")
	inboundSearch.Flags().StringVar(&mailboxHash, "mailbox-hash", "", "Mailbox hash filter")
	inboundSearch.Flags().StringVar(&status, "status", "", "Inbound status filter")
	inboundSearch.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	inboundSearch.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	addCountOffsetFlags(inboundSearch, &inboundCount, &inboundOffset)
	cmd.AddCommand(inboundSearch)

	var opensCount, opensOffset int
	opens := &cobra.Command{
		Use:   "opens",
		Short: "List outbound message opens",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{"count": {strconv.Itoa(opensCount)}, "offset": {strconv.Itoa(opensOffset)}}
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				addIfSet(q, "tag", tag)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/outbound/opens", q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Opens", opensOffset, opensCount, globals.Format, globals.Full)
			})
		},
	}
	opens.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	opens.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	opens.Flags().StringVar(&tag, "tag", "", "Tag filter")
	addCountOffsetFlags(opens, &opensCount, &opensOffset)
	cmd.AddCommand(opens)

	var clicksCount, clicksOffset int
	clicks := &cobra.Command{
		Use:   "clicks",
		Short: "List outbound message clicks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{"count": {strconv.Itoa(clicksCount)}, "offset": {strconv.Itoa(clicksOffset)}}
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				addIfSet(q, "tag", tag)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/outbound/clicks", q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Clicks", clicksOffset, clicksCount, globals.Format, globals.Full)
			})
		},
	}
	clicks.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	clicks.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	clicks.Flags().StringVar(&tag, "tag", "", "Tag filter")
	addCountOffsetFlags(clicks, &clicksCount, &clicksOffset)
	cmd.AddCommand(clicks)

	cmd.AddCommand(serverGetCommand("get <message-id>", "Get outbound message details", globals, "/messages/outbound/%s/details"))
	cmd.AddCommand(serverGetCommand("inbound-get <message-id>", "Get inbound message details", globals, "/messages/inbound/%s/details"))
	cmd.AddCommand(serverPutCommand("inbound-retry <message-id>", "Retry inbound message processing", globals, "/messages/inbound/%s/retry", "Retrying inbound processing can trigger downstream processing again."))
	cmd.AddCommand(serverPutCommand("inbound-bypass <message-id>", "Bypass inbound message rules", globals, "/messages/inbound/%s/bypass", "Bypassing inbound rules can deliver a message that rules previously blocked."))
	root.AddCommand(cmd)
}

func registerBounces(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "bounces", Short: "List and inspect bounces"}
	var email, bounceType, inactive, tag, messageID, fromDate, toDate, stream string
	var count, offset int
	list := &cobra.Command{
		Use:   "list",
		Short: "List bounces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{}
				q.Set("count", strconv.Itoa(count))
				q.Set("offset", strconv.Itoa(offset))
				addIfSet(q, "emailFilter", email)
				addIfSet(q, "type", bounceType)
				addIfSet(q, "inactive", inactive)
				addIfSet(q, "tag", tag)
				addIfSet(q, "messageID", messageID)
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				if stream == "" {
					stream = resolved.MessageStream
				}
				addIfSet(q, "messagestream", stream)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/bounces", q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Bounces", offset, count, globals.Format, globals.Full)
			})
		},
	}
	list.Flags().StringVar(&email, "email", "", "Email filter")
	list.Flags().StringVar(&bounceType, "type", "", "Bounce type filter")
	list.Flags().StringVar(&inactive, "inactive", "", "Inactive filter: true or false")
	list.Flags().StringVar(&tag, "tag", "", "Tag filter")
	list.Flags().StringVar(&messageID, "message-id", "", "Message ID filter")
	list.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	list.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	list.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	addCountOffsetFlags(list, &count, &offset)
	cmd.AddCommand(list)
	cmd.AddCommand(serverGetCommand("get <bounce-id>", "Get a bounce", globals, "/bounces/%s"))
	root.AddCommand(cmd)
}

func registerSuppressions(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "suppressions", Short: "Check and manage suppressions"}
	var email, reason, origin, stream string
	var count, offset int
	dump := &cobra.Command{
		Use:   "dump",
		Short: "Dump suppressions for a message stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if stream == "" {
					stream = resolved.MessageStream
				}
				q := url.Values{"count": {strconv.Itoa(count)}, "offset": {strconv.Itoa(offset)}}
				addIfSet(q, "EmailAddress", email)
				addIfSet(q, "SuppressionReason", reason)
				addIfSet(q, "Origin", origin)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/dump", stream), q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Suppressions", offset, count, globals.Format, globals.Full)
			})
		},
	}
	dump.Flags().StringVar(&email, "email", "", "Email address filter")
	dump.Flags().StringVar(&reason, "reason", "", "Suppression reason filter")
	dump.Flags().StringVar(&origin, "origin", "", "Suppression origin filter")
	dump.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	addCountOffsetFlags(dump, &count, &offset)
	cmd.AddCommand(dump)

	check := &cobra.Command{
		Use:   "check <email>",
		Short: "Check suppression status for an email address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if stream == "" {
					stream = resolved.MessageStream
				}
				q := url.Values{"count": {"10"}, "offset": {"0"}, "EmailAddress": {args[0]}}
				raw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/dump", stream), q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Suppressions", 0, 10, globals.Format, globals.Full)
			})
		},
	}
	check.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	cmd.AddCommand(check)

	cmd.AddCommand(suppressionMutationCommand("create <email>", "Create a manual suppression", globals, false, "Creating a suppression blocks future delivery to this recipient."))
	cmd.AddCommand(suppressionMutationCommand("delete <email>", "Delete a suppression", globals, true, "Deleting a suppression may allow future delivery to this recipient."))
	root.AddCommand(cmd)
}

func registerStats(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "stats", Short: "Get delivery and outbound stats"}
	cmd.AddCommand(&cobra.Command{
		Use:   "delivery",
		Short: "Get delivery stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/deliverystats", url.Values{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	})
	root.AddCommand(cmd)
}

func accountListCommand(use, short string, globals *GlobalFlags, path, envelope string) *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{"count": {strconv.Itoa(count)}, "offset": {strconv.Itoa(offset)}}
				raw, err := resolved.Client.Get(ctx, api.AccountToken, path, q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, envelope, offset, count, globals.Format, globals.Full)
			})
		},
	}
	addCountOffsetFlags(cmd, &count, &offset)
	return cmd
}

func accountGetCommand(use, short string, globals *GlobalFlags, pathFormat string) *cobra.Command {
	return getCommand(use, short, globals, api.AccountToken, pathFormat)
}

func serverGetCommand(use, short string, globals *GlobalFlags, pathFormat string) *cobra.Command {
	return getCommand(use, short, globals, api.ServerToken, pathFormat)
}

func getCommand(use, short string, globals *GlobalFlags, kind api.TokenKind, pathFormat string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, kind, fmt.Sprintf(pathFormat, args[0]), url.Values{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
}

func accountPostCommand(use, short string, globals *GlobalFlags, pathFormat string, warning string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				output.WriteError(output.Stderr(), agenterrors.New("mutation requires --yes", agenterrors.FixableByHuman).WithHint(warning))
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Post(ctx, api.AccountToken, fmt.Sprintf(pathFormat, args[0]), url.Values{}, map[string]any{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm this state-changing Postmark request")
	return cmd
}

func serverPutCommand(use, short string, globals *GlobalFlags, pathFormat string, warning string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				output.WriteError(output.Stderr(), agenterrors.New("mutation requires --yes", agenterrors.FixableByHuman).WithHint(warning))
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Put(ctx, api.ServerToken, fmt.Sprintf(pathFormat, args[0]), url.Values{}, map[string]any{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm this state-changing Postmark request")
	return cmd
}

func suppressionMutationCommand(use, short string, globals *GlobalFlags, deleteSuppression bool, warning string) *cobra.Command {
	var yes bool
	var stream string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				output.WriteError(output.Stderr(), agenterrors.New("mutation requires --yes", agenterrors.FixableByHuman).WithHint(warning))
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if stream == "" {
					stream = resolved.MessageStream
				}
				path := fmt.Sprintf("/message-streams/%s/suppressions", stream)
				if deleteSuppression {
					path += "/delete"
				}
				body := map[string]any{"Suppressions": []map[string]any{{"EmailAddress": args[0]}}}
				raw, err := resolved.Client.Post(ctx, api.ServerToken, path, url.Values{}, body)
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	cmd.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm this state-changing Postmark request")
	return cmd
}

func writeEnvelopeList(raw json.RawMessage, field, format string, full bool) error {
	return writeEnvelopeListWithPage(raw, field, 0, 0, format, full)
}

func writeEnvelopeListWithPage(raw json.RawMessage, field string, offset, count int, format string, full bool) error {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return writeRaw(raw, format)
	}
	var total int
	if totalRaw, ok := payload["TotalCount"]; ok {
		_ = json.Unmarshal(totalRaw, &total)
	}
	var items []json.RawMessage
	if listRaw, ok := payload[field]; ok {
		_ = json.Unmarshal(listRaw, &items)
	}
	if count == 0 {
		count = len(items)
	}
	if total == 0 {
		total = len(items)
	}
	return writeList(items, total, offset, count, field, format, full)
}

func addIfSet(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func writeWebhookHealth(raw json.RawMessage, format string) error {
	var payload struct {
		Webhooks []map[string]any `json:"Webhooks"`
	}
	_ = json.Unmarshal(raw, &payload)
	rows := make([]json.RawMessage, 0, len(payload.Webhooks)+1)
	coverage := map[string]int{"delivery": 0, "bounce": 0, "inbound": 0, "spam_complaint": 0}
	for _, hook := range payload.Webhooks {
		triggers, _ := hook["Triggers"].(map[string]any)
		for key, value := range triggers {
			enabled, _ := value.(bool)
			if !enabled {
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
		row, _ := json.Marshal(map[string]any{
			"type":           "entity",
			"object":         "webhook",
			"id":             hook["ID"],
			"message_stream": hook["MessageStream"],
			"triggers":       hook["Triggers"],
		})
		rows = append(rows, row)
	}
	severity := "ok"
	summary := "Webhook coverage includes delivery and bounce triggers."
	if coverage["delivery"] == 0 || coverage["bounce"] == 0 {
		severity = "warning"
		summary = "Webhook coverage is missing delivery or bounce triggers."
	}
	finding, _ := json.Marshal(map[string]any{
		"type":     "finding",
		"severity": severity,
		"summary":  summary,
		"coverage": coverage,
	})
	rows = append(rows, finding)
	return writeList(rows, len(rows), 0, len(rows), "WebhookHealth", format, true)
}
