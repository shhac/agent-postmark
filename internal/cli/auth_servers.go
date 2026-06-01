package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
)

func registerAuthServers(parent *cobra.Command) {
	cmd := &cobra.Command{Use: "servers", Short: "Manage server-token contexts within a profile"}
	registerAuthServerAdd(cmd)
	registerAuthServerUpdate(cmd)
	registerAuthServerDefault(cmd)
	registerAuthServerList(cmd)
	registerAuthServerRemove(cmd)
	parent.AddCommand(cmd)
}

func registerAuthServerAdd(parent *cobra.Command) {
	var token, stream string
	var serverID int
	var form, setDefault bool
	cmd := &cobra.Command{
		Use:   "add <profile> <server>",
		Short: "Add a server token context to a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileAlias, serverAlias := args[0], args[1]
			if form && token == "" {
				filled, err := promptSecret(cmd.Context(), "agent-postmark: "+profileAlias+"/"+serverAlias, "Postmark server token", "")
				if err != nil {
					writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --server-token-value in a human-controlled terminal.")
					return nil
				}
				token = filled
			}
			if token == "" {
				writeProfileAgentError("missing --server-token-value", "Use 'agent-postmark profiles servers add <profile> <server> --form --server-token' so the token never appears in chat.")
				return nil
			}
			if _, err := credential.StoreServer(profileAlias, serverAlias, token); err != nil {
				writeProfileHumanError(err, "The server token was not written to Keychain. Fix Keychain access and retry with --form.")
				return nil
			}
			if err := config.StoreServer(profileAlias, serverAlias, config.ServerProfile{ServerID: serverID, MessageStream: stream}); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			if setDefault {
				if err := config.SetDefaultServer(profileAlias, serverAlias); err != nil {
					writeProfileCommandError(err)
					return nil
				}
			}
			return writeServerProfileResult("server_added", profileAlias, serverAlias)
		},
	}
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for the server token via native OS dialog")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this the default server for the profile")
	cmd.Flags().StringVar(&token, "server-token-value", "", "Postmark server token")
	cmd.Flags().Bool("server-token", false, "Store a server token for server-level APIs")
	cmd.Flags().IntVar(&serverID, "server-id", 0, "Numeric Postmark server ID")
	cmd.Flags().StringVar(&stream, "stream", "outbound", "Default message stream for this server")
	parent.AddCommand(cmd)
}

func registerAuthServerUpdate(parent *cobra.Command) {
	var token, stream string
	var serverID int
	var form, updateToken, clearServerID, clearStream, setDefault bool
	cmd := &cobra.Command{
		Use:   "update <profile> <server>",
		Short: "Update a server token context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileAlias, serverAlias := args[0], args[1]
			if form && updateToken {
				filled, err := promptSecret(cmd.Context(), "agent-postmark: "+profileAlias+"/"+serverAlias, "Postmark server token", "")
				if err != nil {
					writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --server-token-value in a human-controlled terminal.")
					return nil
				}
				token = filled
			}
			if updateToken {
				if token == "" {
					writeProfileAgentError("missing --server-token-value", "Use 'agent-postmark profiles servers update <profile> <server> --form --server-token' so the token never appears in chat.")
					return nil
				}
				if _, err := credential.StoreServer(profileAlias, serverAlias, token); err != nil {
					writeProfileHumanError(err, "The server token was not written to Keychain. Fix Keychain access and retry with --form.")
					return nil
				}
			}
			if err := config.UpdateServer(profileAlias, serverAlias, func(server config.ServerProfile) config.ServerProfile {
				if cmd.Flags().Changed("server-id") {
					server.ServerID = serverID
				}
				if cmd.Flags().Changed("stream") {
					server.MessageStream = stream
				}
				if clearServerID {
					server.ServerID = 0
				}
				if clearStream {
					server.MessageStream = ""
				}
				return server
			}); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			if setDefault {
				if err := config.SetDefaultServer(profileAlias, serverAlias); err != nil {
					writeProfileCommandError(err)
					return nil
				}
			}
			return writeServerProfileResult("server_updated", profileAlias, serverAlias)
		},
	}
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for replacement token via native OS dialog")
	cmd.Flags().BoolVar(&updateToken, "server-token", false, "Replace the server token")
	cmd.Flags().StringVar(&token, "server-token-value", "", "Replacement server token")
	cmd.Flags().IntVar(&serverID, "server-id", 0, "Numeric Postmark server ID")
	cmd.Flags().StringVar(&stream, "stream", "", "Default message stream for this server")
	cmd.Flags().BoolVar(&clearServerID, "clear-server-id", false, "Clear the numeric Postmark server ID")
	cmd.Flags().BoolVar(&clearStream, "clear-stream", false, "Clear the default message stream")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Make this the default server for the profile")
	parent.AddCommand(cmd)
}

func registerAuthServerDefault(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "default <profile> <server>",
		Short: "Set the default server context for a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetDefaultServer(args[0], args[1]); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			return writeServerProfileResult("server_default_set", args[0], args[1])
		},
	})
}

func registerAuthServerList(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "list <profile>",
		Short: "List server contexts for a profile without exposing tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Read()
			profile, ok := cfg.Profiles[args[0]]
			if !ok {
				writeProfileCommandError(config.ErrProfileNotConfigured(args[0]))
				return nil
			}
			rows := make([]json.RawMessage, 0, len(profile.Servers))
			summary := credential.Summary(args[0])
			serverTokens, _ := summary["servers"].(map[string]bool)
			for alias, server := range profile.Servers {
				row, _ := json.Marshal(map[string]any{
					"profile":        args[0],
					"server":         alias,
					"default":        alias == profile.DefaultServer,
					"server_id":      server.ServerID,
					"message_stream": server.MessageStream,
					"credential":     "keychain",
					"server_token":   serverTokens[alias],
				})
				rows = append(rows, row)
			}
			return writeList(rows, len(rows), 0, len(rows), "ProfileServers", "", false)
		},
	})
}

func registerAuthServerRemove(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "remove <profile> <server>",
		Short: "Remove a server context and its token",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := credential.RemoveServer(args[0], args[1]); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			if err := config.RemoveServer(args[0], args[1]); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			return writeItem(map[string]any{"status": "server_removed", "profile": args[0], "server": args[1]}, "")
		},
	})
}

func writeServerProfileResult(status, profileAlias, serverAlias string) error {
	cfg := config.Read()
	profile := cfg.Profiles[profileAlias]
	server := profile.Servers[serverAlias]
	return writeItem(map[string]any{
		"status":          status,
		"profile":         profileAlias,
		"server":          serverAlias,
		"default":         profile.DefaultServer == serverAlias,
		"server_id":       server.ServerID,
		"message_stream":  server.MessageStream,
		"credential":      "keychain",
		"credential_kind": credential.Summary(profileAlias),
	}, "")
}
