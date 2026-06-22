package cli

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
	"github.com/shhac/agent-postmark/internal/dialog"
)

var promptSecret = dialog.PromptSecret

func registerAuth(root *cobra.Command, globals *GlobalFlags) {
	auth := &cobra.Command{
		Use:   "profiles",
		Short: "Manage Postmark credential profiles",
	}
	registerAuthAdd(auth)
	registerAuthSetup(auth)
	registerAuthUpdate(auth)
	registerAuthServers(auth, globals)
	registerAuthCheck(auth, globals)
	registerAuthDefault(auth)
	registerAuthList(auth, globals)
	registerAuthRemove(auth)
	root.AddCommand(auth)
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
					"server":         resolved.Server,
					"default_server": resolved.ServerID,
					"message_stream": resolved.MessageStream,
					"tokens": map[string]bool{
						"account": resolved.AccountToken,
						"server":  resolved.ServerToken,
					},
				}
				if resolved.AccountToken {
					raw, err := resolved.Client.Get(ctx, api.AccountToken, "/servers", paginationQuery(1, 0))
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
				writeProfileCommandError(err)
				return nil
			}
			return writeItem(map[string]any{"status": "default_set", "profile": args[0]}, "")
		},
	})
}

func registerAuthList(parent *cobra.Command, globals *GlobalFlags) {
	parent.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured profiles without exposing tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Read()
			rows := profileListRows(cfg)
			return writeList(rows, len(rows), 0, len(rows), "Profiles", globals.Format, false)
		},
	})
}

func profileListRows(cfg *config.Config) []json.RawMessage {
	rows := make([]json.RawMessage, 0, len(cfg.Profiles))
	for _, alias := range sortedProfileAliases(cfg.Profiles) {
		rows = append(rows, profileListRow(alias, cfg.Profiles[alias], alias == cfg.DefaultProfile))
	}
	return rows
}

func profileListRow(alias string, profile config.Profile, isDefault bool) json.RawMessage {
	fields := map[string]any{
		"profile":        alias,
		"default":        isDefault,
		"credentials":    credential.Summary(alias),
		"host":           profile.Host,
		"default_server": profile.DefaultServer,
		"servers":        profile.Servers,
	}
	if storage := credential.ProfileStorage(alias); storage != "" {
		fields["storage"] = storage
	}
	row, _ := json.Marshal(fields)
	return row
}

func sortedProfileAliases(profiles map[string]config.Profile) []string {
	aliases := make([]string, 0, len(profiles))
	for alias := range profiles {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func sortedServerAliases(servers map[string]config.ServerProfile) []string {
	aliases := make([]string, 0, len(servers))
	for alias := range servers {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func registerAuthRemove(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a profile and its Keychain tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := credential.Remove(args[0]); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			if err := config.RemoveProfile(args[0]); err != nil {
				writeProfileCommandError(err)
				return nil
			}
			return writeItem(map[string]any{"status": "removed", "profile": args[0]}, "")
		},
	})
}
