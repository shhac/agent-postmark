package cli

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
	"github.com/shhac/agent-postmark/internal/dialog"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

var promptSecret = dialog.PromptSecret

func registerAuth(root *cobra.Command, globals *GlobalFlags) {
	auth := &cobra.Command{Use: "profiles", Short: "Manage Postmark credential profiles"}
	registerAuthAdd(auth)
	registerAuthUpdate(auth)
	registerAuthCheck(auth, globals)
	registerAuthDefault(auth)
	registerAuthList(auth)
	registerAuthRemove(auth)
	root.AddCommand(auth)
}

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
						output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
						return nil
					}
				}
				if wantServer && serverToken == "" {
					serverToken, err = promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark server token", "")
					if err != nil {
						output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
						return nil
					}
				}
			}
			if wantAccount && accountToken == "" {
				output.WriteError(output.Stderr(), agenterrors.New("missing --account-token", agenterrors.FixableByAgent).
					WithHint("Agents should use 'agent-postmark profiles add <profile> --form --account-token' so the token never appears in chat."))
				return nil
			}
			if wantServer && serverToken == "" {
				output.WriteError(output.Stderr(), agenterrors.New("missing --server-token", agenterrors.FixableByAgent).
					WithHint("Agents should use 'agent-postmark profiles add <profile> --form --server-token' so the token never appears in chat."))
				return nil
			}
			stored := map[string]string{}
			if wantAccount {
				storage, err := credential.Store(alias, credential.AccountToken, accountToken)
				if err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
						WithHint("The account token was not written to disk. Fix Keychain access and retry with --form."))
					return nil
				}
				stored["account_token"] = storage
			}
			if wantServer {
				storage, err := credential.Store(alias, credential.ServerToken, serverToken)
				if err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
						WithHint("The server token was not written to disk. Fix Keychain access and retry with --form."))
					return nil
				}
				stored["server_token"] = storage
			}
			if err := config.StoreProfile(alias, config.Profile{Host: host, DefaultServer: serverID, MessageStream: stream}); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
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
						output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
						return nil
					}
					accountToken = filled
				}
				if updateServer {
					filled, err := promptSecret(cmd.Context(), "agent-postmark: "+alias, "Postmark server token", "")
					if err != nil {
						output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
						return nil
					}
					serverToken = filled
				}
			}
			if updateAccount {
				if accountToken == "" {
					output.WriteError(output.Stderr(), agenterrors.New("missing --account-token-value", agenterrors.FixableByAgent))
					return nil
				}
				if _, err := credential.Store(alias, credential.AccountToken, accountToken); err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
					return nil
				}
			}
			if updateServer {
				if serverToken == "" {
					output.WriteError(output.Stderr(), agenterrors.New("missing --server-token-value", agenterrors.FixableByAgent))
					return nil
				}
				if _, err := credential.Store(alias, credential.ServerToken, serverToken); err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
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
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).
					WithHint("Run 'agent-postmark profiles list' to see configured profiles."))
				return nil
			}
			if setDefault {
				if err := config.SetDefault(alias); err != nil {
					output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
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

func registerAuthCheck(parent *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{
		Use:   "check [profile]",
		Short: "Verify stored credentials against Postmark",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := *globals
			if len(args) > 0 {
				flags.Profile = args[0]
			}
			return withClient(cmd.Context(), &flags, func(ctx context.Context, resolved *resolvedContext) error {
				result := map[string]any{
					"status":         "ok",
					"profile":        resolved.Profile,
					"host":           resolved.Host,
					"default_server": resolved.ServerID,
					"message_stream": resolved.MessageStream,
					"tokens": map[string]bool{
						"account": resolved.AccountToken,
						"server":  resolved.ServerToken,
					},
				}
				if resolved.AccountToken {
					raw, err := resolved.Client.Get(ctx, api.AccountToken, "/servers", nil)
					if err != nil {
						return err
					}
					var payload map[string]any
					_ = json.Unmarshal(raw, &payload)
					result["account_check"] = map[string]any{"route": "/servers", "ok": true, "server_count": payload["TotalCount"]}
				}
				if resolved.ServerToken {
					raw, err := resolved.Client.Get(ctx, api.ServerToken, "/deliverystats", nil)
					if err != nil {
						return err
					}
					var payload map[string]any
					_ = json.Unmarshal(raw, &payload)
					result["server_check"] = map[string]any{"route": "/deliverystats", "ok": true, "inactive_mails": payload["InactiveMails"]}
				}
				return writeItem(result, flags.Format)
			})
		},
	}
	parent.AddCommand(cmd)
}

func registerAuthDefault(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "default <profile>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetDefault(args[0]); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
				return nil
			}
			return writeItem(map[string]any{"status": "default_set", "profile": args[0]}, "")
		},
	})
}

func registerAuthList(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured profiles without exposing tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Read()
			rows := make([]json.RawMessage, 0, len(cfg.Profiles))
			for alias, profile := range cfg.Profiles {
				row, _ := json.Marshal(map[string]any{
					"profile":         alias,
					"default":         alias == cfg.DefaultProfile,
					"credential":      "keychain",
					"credential_kind": credential.Summary(alias),
					"host":            profile.Host,
					"default_server":  profile.DefaultServer,
					"message_stream":  profile.MessageStream,
				})
				rows = append(rows, row)
			}
			return writeList(rows, len(rows), 0, len(rows), "Profiles", "", false)
		},
	})
}

func registerAuthRemove(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a profile and its Keychain tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := credential.Remove(args[0]); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
				return nil
			}
			if err := config.RemoveProfile(args[0]); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman))
				return nil
			}
			return writeItem(map[string]any{"status": "removed", "profile": args[0]}, "")
		},
	})
}
