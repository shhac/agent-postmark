package cli

import "github.com/spf13/cobra"

func registerUsage(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "Show common agent workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeItem(map[string]any{
				"setup": []string{
					"agent-postmark profiles setup prod --form --account-token --server app:<id>:outbound --server billing:<id>:outbound",
					"agent-postmark profiles add prod --form --account-token",
					"agent-postmark profiles servers add prod app --form --server-token --server-id <id> --stream outbound --default",
					"agent-postmark profiles check prod",
					"agent-postmark profiles servers update prod app --server-id <id> --stream outbound --default",
				},
				"rotation": []string{
					"agent-postmark profiles update prod --form --account-token",
					"agent-postmark profiles servers update prod app --form --server-token",
					"agent-postmark profiles servers remove prod old-server",
				},
				"discover": []string{
					"agent-postmark servers list",
					"agent-postmark --server <alias> streams list",
					"agent-postmark domains list",
					"agent-postmark signatures list",
					"agent-postmark webhooks list",
				},
				"triage": []string{
					"agent-postmark messages search --to user@example.com --fromdate 2026-05-01T00:00:00",
					"agent-postmark bounces list --email user@example.com --count 20",
					"agent-postmark suppressions check user@example.com",
					"agent-postmark messages get <message-id>",
					"agent-postmark investigate delivery --email user@example.com",
				},
				"server_token_workflows": []string{
					"agent-postmark messages inbound-search --from reply@example.com",
					"agent-postmark messages dump <message-id>",
					"agent-postmark messages opens --count 20",
					"agent-postmark messages opens --message-id <message-id>",
					"agent-postmark messages clicks --count 20",
					"agent-postmark messages clicks --message-id <message-id>",
					"agent-postmark bounces dump <bounce-id>",
					"agent-postmark suppressions list --stream outbound",
					"agent-postmark webhooks health",
				},
				"mutations": []string{
					"agent-postmark domains verify-dkim <domain-id> --yes",
					"agent-postmark bounces activate <bounce-id> --yes",
					"agent-postmark suppressions create user@example.com --yes",
					"agent-postmark suppressions delete user@example.com --yes",
					"agent-postmark messages inbound-retry <message-id> --yes",
				},
				"safety": []string{
					"Secrets are stored in Keychain and never printed.",
					"Lists default to NDJSON; single resources default to JSON.",
					"Message bodies, recipients, headers, attachments, tokens, and secrets are redacted by default.",
				},
			}, "")
		},
	})
}
