package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
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
				raw, err := resolved.Client.Get(ctx, api.ServerToken, "/message-streams", url.Values{})
				if err != nil {
					return err
				}
				return writeEnvelopeList(raw, "MessageStreams", globals.Format, globals.Full)
			})
		},
	})
	cmd.AddCommand(serverGetCommand("get <stream-id>", "Get a message stream", globals, "/message-streams/%s"))
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
