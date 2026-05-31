package cli

import (
	"context"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
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
	var trackedMessageID string
	opens := &cobra.Command{
		Use:   "opens",
		Short: "List outbound message opens",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				q := url.Values{"count": {strconv.Itoa(opensCount)}, "offset": {strconv.Itoa(opensOffset)}}
				addIfSet(q, "fromdate", fromDate)
				addIfSet(q, "todate", toDate)
				addIfSet(q, "tag", tag)
				path := "/messages/outbound/opens"
				if trackedMessageID != "" {
					path += "/" + trackedMessageID
				}
				raw, err := resolved.Client.Get(ctx, api.ServerToken, path, q)
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
	opens.Flags().StringVar(&trackedMessageID, "message-id", "", "Single message ID")
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
				path := "/messages/outbound/clicks"
				if trackedMessageID != "" {
					path += "/" + trackedMessageID
				}
				raw, err := resolved.Client.Get(ctx, api.ServerToken, path, q)
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
	clicks.Flags().StringVar(&trackedMessageID, "message-id", "", "Single message ID")
	addCountOffsetFlags(clicks, &clicksCount, &clicksOffset)
	cmd.AddCommand(clicks)

	cmd.AddCommand(serverGetCommand("get <message-id>", "Get outbound message details", globals, "/messages/outbound/%s/details"))
	cmd.AddCommand(serverGetCommand("dump <message-id>", "Get redacted outbound message dump", globals, "/messages/outbound/%s/dump"))
	cmd.AddCommand(serverGetCommand("inbound-get <message-id>", "Get inbound message details", globals, "/messages/inbound/%s/details"))
	cmd.AddCommand(serverPutCommand("inbound-retry <message-id>", "Retry inbound message processing", globals, "/messages/inbound/%s/retry", "Retrying inbound processing can trigger downstream processing again."))
	cmd.AddCommand(serverPutCommand("inbound-bypass <message-id>", "Bypass inbound message rules", globals, "/messages/inbound/%s/bypass", "Bypassing inbound rules can deliver a message that rules previously blocked."))
	root.AddCommand(cmd)
}
