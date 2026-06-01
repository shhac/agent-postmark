package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
)

func registerAuthAdd(parent *cobra.Command) {
	var accountToken, host string
	var wantAccount, form bool

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a Postmark profile with an optional account token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if form {
				var err error
				if wantAccount && accountToken == "" {
					accountToken, err = promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark account token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --account-token-value in a human-controlled terminal.")
						return nil
					}
				}
			}
			if wantAccount && accountToken == "" {
				writeProfileAgentError("missing --account-token", "Agents should use 'agent-postmark profiles add <profile> --form --account-token' so the token never appears in chat.")
				return nil
			}
			stored := map[string]string{}
			if wantAccount {
				storage, err := credential.StoreAccount(alias, accountToken)
				if err != nil {
					writeProfileHumanError(err, "The account token was not written to disk. Fix Keychain access and retry with --form.")
					return nil
				}
				stored["account"] = storage
			}
			if err := config.StoreProfile(alias, config.Profile{Host: host}); err != nil {
				writeProfileHumanError(err, "Check config directory permissions and retry.")
				return nil
			}
			cfg := config.Read()
			profile := cfg.Profiles[alias]
			return writeItem(map[string]any{
				"status":         "added",
				"profile":        alias,
				"default":        cfg.DefaultProfile == alias,
				"storage":        stored,
				"host":           profile.Host,
				"default_server": profile.DefaultServer,
				"servers":        profile.Servers,
				"credentials":    credential.Summary(alias),
			}, "")
		},
	}
	cmd.Flags().StringVar(&accountToken, "account-token-value", "", "Postmark account token (use --form for LLM-guided setup)")
	cmd.Flags().BoolVar(&wantAccount, "account-token", false, "Store an account token for account-level APIs")
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for tokens via native OS dialogs")
	cmd.Flags().StringVar(&host, "host", config.DefaultHost, "Postmark API host")
	parent.AddCommand(cmd)
}
