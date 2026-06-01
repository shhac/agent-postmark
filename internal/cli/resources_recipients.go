package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
)

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
	cmd.AddCommand(serverGetCommand("dump <bounce-id>", "Get redacted bounce dump", globals, "/bounces/%s/dump"))
	cmd.AddCommand(serverPutCommand("activate <bounce-id>", "Reactivate a bounced recipient", globals, "/bounces/%s/activate", "Activating a bounce can allow future delivery to this recipient."))
	root.AddCommand(cmd)
}

func registerSuppressions(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "suppressions", Short: "Check and manage suppressions"}
	var email, reason, origin, stream string
	var count, offset int
	list := &cobra.Command{
		Use:   "list",
		Short: "List suppressions for a message stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				if stream == "" {
					stream = resolved.MessageStream
				}
				q := url.Values{"count": {strconv.Itoa(count)}, "offset": {strconv.Itoa(offset)}}
				addIfSet(q, "EmailAddress", email)
				addIfSet(q, "SuppressionReason", reason)
				addIfSet(q, "Origin", origin)
				raw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/list", stream), q)
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, "Suppressions", offset, count, globals.Format, globals.Full)
			})
		},
	}
	list.Flags().StringVar(&email, "email", "", "Email address filter")
	list.Flags().StringVar(&reason, "reason", "", "Suppression reason filter")
	list.Flags().StringVar(&origin, "origin", "", "Suppression origin filter")
	list.Flags().StringVar(&stream, "stream", "", "Message stream ID")
	addCountOffsetFlags(list, &count, &offset)
	cmd.AddCommand(list)

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
				raw, err := resolved.Client.Get(ctx, api.ServerToken, fmt.Sprintf("/message-streams/%s/suppressions/list", stream), q)
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

func suppressionMutationCommand(use, short string, globals *GlobalFlags, deleteSuppression bool, warning string) *cobra.Command {
	var yes bool
	var stream string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmMutation(yes, warning) {
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
