package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	"github.com/shhac/agent-postmark/internal/output"
)

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
				addIfSet(q, "tag", tag)
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
	inboundSearch.Flags().StringVar(&tag, "tag", "", "Tag filter")
	inboundSearch.Flags().StringVar(&status, "status", "", "Inbound status filter")
	inboundSearch.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	inboundSearch.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	addCountOffsetFlags(inboundSearch, &inboundCount, &inboundOffset)
	cmd.AddCommand(inboundSearch)

	cmd.AddCommand(messageActivityCommand("opens", "List outbound message opens", globals, "/messages/outbound/opens", "Opens"))
	cmd.AddCommand(messageActivityCommand("clicks", "List outbound message clicks", globals, "/messages/outbound/clicks", "Clicks"))

	cmd.AddCommand(serverGetCommand("get <message-id>", "Get outbound message details", globals, "/messages/outbound/%s/details"))
	cmd.AddCommand(messageContentCommand(globals))
	cmd.AddCommand(singleGetCommand("dump <message-id>", "Get outbound message dump", globals, api.ServerToken, "/messages/outbound/%s/dump"))
	cmd.AddCommand(serverGetCommand("inbound-get <message-id>", "Get inbound message details", globals, "/messages/inbound/%s/details"))
	cmd.AddCommand(serverPutCommand("inbound-retry <message-id>", "Retry inbound message processing", globals, "/messages/inbound/%s/retry", "Retrying inbound processing can trigger downstream processing again."))
	cmd.AddCommand(serverPutCommand("inbound-bypass <message-id>", "Bypass inbound message rules", globals, "/messages/inbound/%s/bypass", "Bypassing inbound rules can deliver a message that rules previously blocked."))
	root.AddCommand(cmd)
}

func messageActivityCommand(use, short string, globals *GlobalFlags, basePath, envelope string) *cobra.Command {
	var count, offset int
	var fromDate, toDate, tag, messageID string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := paginationQuery(count, offset)
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				addIfSet(q, "tag", tag)
				path := basePath
				if messageID != "" {
					path += "/" + messageID
				}
				raw, err := resolved.Client.Get(ctx, api.ServerToken, path, q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, envelope, offset, count, globals.Format, globals.Full)
			})
		},
	}
	cmd.Flags().StringVar(&fromDate, "fromdate", "", "Start datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	cmd.Flags().StringVar(&toDate, "todate", "", "End datetime, Postmark format YYYY-MM-DDTHH:MM:SS")
	cmd.Flags().StringVar(&tag, "tag", "", "Tag filter")
	cmd.Flags().StringVar(&messageID, "message-id", "", "Single message ID")
	addCountOffsetFlags(cmd, &count, &offset)
	return cmd
}

func messageContentCommand(globals *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "content <message-id> [message-id...]",
		Short: "Get email content for one or more outbound messages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				rows := make([]json.RawMessage, 0, len(args))
				for _, id := range args {
					raw, err := resolved.Client.Get(ctx, api.ServerToken, "/messages/outbound/"+id+"/details", url.Values{})
					if err != nil {
						return err
					}
					rows = append(rows, raw)
				}
				return writeMessageContents(rows, globals.Format)
			})
		},
	}
}

func writeMessageContents(rows []json.RawMessage, flagFormat string) error {
	defaultFormat := output.FormatJSON
	if len(rows) > 1 {
		defaultFormat = output.FormatNDJSON
	}
	format, err := output.ResolveFormat(flagFormat, defaultFormat)
	if err != nil {
		return err
	}
	if format == output.FormatNDJSON {
		writer := output.NewNDJSONWriter(output.Stdout())
		for _, raw := range rows {
			var item any
			if err := json.Unmarshal(redactRaw(raw), &item); err != nil {
				return err
			}
			if err := writer.WriteItem(item); err != nil {
				return err
			}
		}
		return nil
	}
	if len(rows) == 1 {
		output.WriteRawJSON(redactRaw(rows[0]), format, true)
		return nil
	}
	decoded := make([]any, 0, len(rows))
	for _, raw := range rows {
		var item any
		if err := json.Unmarshal(redactRaw(raw), &item); err != nil {
			return err
		}
		decoded = append(decoded, item)
	}
	output.Print(map[string]any{"results": decoded, "total": len(rows)}, format, true)
	return nil
}
