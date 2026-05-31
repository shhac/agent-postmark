package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
)

func registerAuthAdd(parent *cobra.Command) {
	var accountToken, serverToken, host, stream string
	var serverID int
	var wantAccount, wantServer, form bool

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Postmark profile with Keychain-stored tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if !wantAccount && !wantServer {
				wantAccount = true
				wantServer = true
			}
			if form {
				var err error
				if wantAccount && accountToken == "" {
					accountToken, err = promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark account token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --account-token-value in a human-controlled terminal.")
						return nil
					}
				}
				if wantServer && serverToken == "" {
					serverToken, err = promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark server token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --server-token-value in a human-controlled terminal.")
						return nil
					}
				}
			}
			if wantAccount && accountToken == "" {
				writeProfileAgentError("missing --account-token", "Agents should use 'agent-postmark profiles add <profile> --form --account-token' so the token never appears in chat.")
				return nil
			}
			if wantServer && serverToken == "" {
				writeProfileAgentError("missing --server-token", "Agents should use 'agent-postmark profiles add <profile> --form --server-token' so the token never appears in chat.")
				return nil
			}
			stored := map[string]string{}
			if wantAccount {
				storage, err := credential.Store(alias, credential.AccountToken, accountToken)
				if err != nil {
					writeProfileHumanError(err, "The account token was not written to disk. Fix Keychain access and retry with --form.")
					return nil
				}
				stored["account_token"] = storage
			}
			if wantServer {
				storage, err := credential.Store(alias, credential.ServerToken, serverToken)
				if err != nil {
					writeProfileHumanError(err, "The server token was not written to disk. Fix Keychain access and retry with --form.")
					return nil
				}
				stored["server_token"] = storage
			}
			if err := config.StoreProfile(alias, config.Profile{Host: host, DefaultServer: serverID, MessageStream: stream}); err != nil {
				writeProfileHumanError(err, "Check config directory permissions and retry.")
				return nil
			}
			cfg := config.Read()
			profile := cfg.Profiles[alias]
			return writeItem(map[string]any{
				"status":          "added",
				"profile":         alias,
				"default":         cfg.DefaultProfile == alias,
				"storage":         stored,
				"host":            profile.Host,
				"default_server":  profile.DefaultServer,
				"message_stream":  profile.MessageStream,
				"credential_kind": credential.Summary(alias),
			}, "")
		},
	}
	cmd.Flags().StringVar(&accountToken, "account-token-value", "", "Postmark account token (use --form for LLM-guided setup)")
	cmd.Flags().StringVar(&serverToken, "server-token-value", "", "Postmark server token (use --form for LLM-guided setup)")
	cmd.Flags().BoolVar(&wantAccount, "account-token", false, "Store an account token for account-level APIs")
	cmd.Flags().BoolVar(&wantServer, "server-token", false, "Store a server token for server-level APIs")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for tokens via native OS dialogs")
	cmd.Flags().StringVar(&host, "host", config.DefaultHost, "Postmark API host")
	cmd.Flags().IntVar(&serverID, "server", 0, "Default server ID")
	cmd.Flags().StringVar(&stream, "stream", "outbound", "Default message stream")
	parent.AddCommand(cmd)
}
