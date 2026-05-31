package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
)

func registerAuthUpdate(parent *cobra.Command) {
	var host, stream, accountToken, serverToken string
	var serverID int
	var clearServer, clearStream, setDefault, form, updateAccount, updateServer bool

	cmd := &cobra.Command{
		Use:   "update <profile>",
		Short: "Update profile tokens or non-secret metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if form {
				if updateAccount {
					filled, err := promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark account token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --account-token-value in a human-controlled terminal.")
						return nil
					}
					accountToken = filled
				}
				if updateServer {
					filled, err := promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark server token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --server-token-value in a human-controlled terminal.")
						return nil
					}
					serverToken = filled
				}
			}
			if updateAccount {
				if accountToken == "" {
					writeProfileAgentError("missing --account-token-value", "Use 'agent-postmark profiles update <profile> --form --account-token' so the token never appears in chat.")
					return nil
				}
				if _, err := credential.Store(alias, credential.AccountToken, accountToken); err != nil {
					writeProfileHumanError(err, "The account token was not written to Keychain. Fix Keychain access and retry with --form.")
					return nil
				}
			}
			if updateServer {
				if serverToken == "" {
					writeProfileAgentError("missing --server-token-value", "Use 'agent-postmark profiles update <profile> --form --server-token' so the token never appears in chat.")
					return nil
				}
				if _, err := credential.Store(alias, credential.ServerToken, serverToken); err != nil {
					writeProfileHumanError(err, "The server token was not written to Keychain. Fix Keychain access and retry with --form.")
					return nil
				}
			}
			if err := config.UpdateProfile(alias, func(profile config.Profile) config.Profile {
				if cmd.Flags().Changed("host") {
					profile.Host = host
				}
				if cmd.Flags().Changed("server") {
					profile.DefaultServer = serverID
				}
				if cmd.Flags().Changed("stream") {
					profile.MessageStream = stream
				}
				if clearServer {
					profile.DefaultServer = 0
				}
				if clearStream {
					profile.MessageStream = ""
				}
				return profile
			}); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			if setDefault {
				if err := config.SetDefault(alias); err != nil {
					writeProfileCommandError(err)
					return nil
				}
			}
			cfg := config.Read()
			profile := cfg.Profiles[alias]
			return writeItem(map[string]any{
				"status":          "updated",
				"profile":         alias,
				"default":         cfg.DefaultProfile == alias,
				"host":            profile.Host,
				"default_server":  profile.DefaultServer,
				"message_stream":  profile.MessageStream,
				"credential_kind": credential.Summary(alias),
			}, "")
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Default Postmark API host")
	cmd.Flags().IntVar(&serverID, "server", 0, "Default server ID")
	cmd.Flags().StringVar(&stream, "stream", "", "Default message stream")
	cmd.Flags().BoolVar(&clearServer, "clear-server", false, "Clear the default server ID")
	cmd.Flags().BoolVar(&clearStream, "clear-stream", false, "Clear the default message stream")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this profile the default")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for replacement tokens via native OS dialogs")
	cmd.Flags().BoolVar(&updateAccount, "account-token", false, "Replace the account token")
	cmd.Flags().BoolVar(&updateServer, "server-token", false, "Replace the server token")
	cmd.Flags().StringVar(&accountToken, "account-token-value", "", "Replacement account token")
	cmd.Flags().StringVar(&serverToken, "server-token-value", "", "Replacement server token")
	parent.AddCommand(cmd)
}
