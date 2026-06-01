package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
)

type serverSetupSpec struct {
	Alias   string
	ID      int
	Stream  string
	Token   string
	Default bool
}

func registerAuthSetup(parent *cobra.Command) {
	var accountToken, host, defaultServer string
	var serverSpecs, serverTokenValues []string
	var wantAccount, form bool

	cmd := &cobra.Command{
		Use:   "setup <profile>",
		Short: "Add a profile plus multiple server-token contexts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileAlias := args[0]
			specs, err := parseServerSetupSpecs(serverSpecs, serverTokenValues, defaultServer)
			if err != nil {
				writeProfileAgentError("invalid server setup", err.Error())
				return nil
			}
			if accountToken != "" {
				wantAccount = true
			}
			if form && wantAccount && accountToken == "" {
				filled, err := promptSecret(cmd.Context(), "agent-postmark: "+profileAlias, "Postmark account token", "")
				if err != nil {
					writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --account-token-value in a human-controlled terminal.")
					return nil
				}
				accountToken = filled
			}
			if wantAccount && accountToken == "" {
				writeProfileAgentError("missing --account-token-value", "Use 'agent-postmark profiles setup <profile> --form --account-token' so the account token never appears in chat.")
				return nil
			}
			for i := range specs {
				if form && specs[i].Token == "" {
					filled, err := promptSecret(cmd.Context(), "agent-postmark: "+profileAlias+"/"+specs[i].Alias, "Postmark server token", "")
					if err != nil {
						writeProfileHumanError(err, "Run this command in a local graphical session, or omit --form and provide --server-token-value <server>=<token> in a human-controlled terminal.")
						return nil
					}
					specs[i].Token = filled
				}
				if specs[i].Token == "" {
					writeProfileAgentError("missing server token", "Use --form or provide --server-token-value "+specs[i].Alias+"=<token> from a human-controlled terminal.")
					return nil
				}
			}
			if err := config.StoreProfile(profileAlias, config.Profile{Host: host}); err != nil {
				writeProfileHumanError(err, "Check config directory permissions and retry.")
				return nil
			}
			stored := map[string]any{}
			if wantAccount {
				storage, err := credential.StoreAccount(profileAlias, accountToken)
				if err != nil {
					writeProfileHumanError(err, "The account token was not written to disk. Fix Keychain access and retry with --form.")
					return nil
				}
				stored["account"] = storage
			}
			for _, spec := range specs {
				storage, err := credential.StoreServer(profileAlias, spec.Alias, spec.Token)
				if err != nil {
					writeProfileHumanError(err, "The server token for "+spec.Alias+" was not written to Keychain. Fix Keychain access and retry with --form.")
					return nil
				}
				stored[spec.Alias] = storage
				if err := config.StoreServer(profileAlias, spec.Alias, config.ServerProfile{ServerID: spec.ID, MessageStream: spec.Stream}); err != nil {
					writeProfileCommandError(err)
					return nil
				}
				if spec.Default {
					if err := config.SetDefaultServer(profileAlias, spec.Alias); err != nil {
						writeProfileCommandError(err)
						return nil
					}
				}
			}
			cfg := config.Read()
			profile := cfg.Profiles[profileAlias]
			return writeItem(map[string]any{
				"status":         "setup",
				"profile":        profileAlias,
				"default":        cfg.DefaultProfile == profileAlias,
				"host":           profile.Host,
				"default_server": profile.DefaultServer,
				"servers":        profile.Servers,
				"storage":        stored,
				"credentials":    credential.Summary(profileAlias),
			}, "")
		},
	}
	cmd.Flags().BoolVar(&form, "form", false, "Prompt for tokens via native OS dialogs")
	cmd.Flags().BoolVar(&wantAccount, "account-token", false, "Store an account token for account-level APIs")
	cmd.Flags().StringVar(&accountToken, "account-token-value", "", "Postmark account token")
	cmd.Flags().StringArrayVar(&serverSpecs, "server", nil, "Server context as alias:id[:stream], repeatable")
	cmd.Flags().StringArrayVar(&serverTokenValues, "server-token-value", nil, "Server token as alias=token, repeatable")
	cmd.Flags().StringVar(&defaultServer, "default-server", "", "Server alias to make default; defaults to the first --server")
	cmd.Flags().StringVar(&host, "host", config.DefaultHost, "Postmark API host")
	parent.AddCommand(cmd)
}

func parseServerSetupSpecs(rawSpecs, rawTokenValues []string, defaultAlias string) ([]serverSetupSpec, error) {
	tokens := map[string]string{}
	for _, raw := range rawTokenValues {
		alias, token, ok := strings.Cut(raw, "=")
		if !ok || alias == "" || token == "" {
			return nil, fmt.Errorf("server token values must use <server>=<token>")
		}
		tokens[alias] = token
	}
	specs := make([]serverSetupSpec, 0, len(rawSpecs))
	seen := map[string]bool{}
	for _, raw := range rawSpecs {
		parts := strings.Split(raw, ":")
		if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("server specs must use <alias>:<server-id>[:stream]")
		}
		if seen[parts[0]] {
			return nil, fmt.Errorf("server %q is listed more than once", parts[0])
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("server %q has invalid server id %q", parts[0], parts[1])
		}
		stream := "outbound"
		if len(parts) == 3 && parts[2] != "" {
			stream = parts[2]
		}
		specs = append(specs, serverSetupSpec{Alias: parts[0], ID: id, Stream: stream, Token: tokens[parts[0]]})
		seen[parts[0]] = true
	}
	if len(specs) == 0 {
		return nil, nil
	}
	if defaultAlias == "" {
		defaultAlias = specs[0].Alias
	}
	if !seen[defaultAlias] {
		return nil, fmt.Errorf("default server %q is not listed in --server", defaultAlias)
	}
	for i := range specs {
		specs[i].Default = specs[i].Alias == defaultAlias
	}
	return specs, nil
}
