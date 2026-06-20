package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
)

func registerAPI(root *cobra.Command, globals *GlobalFlags) {
	cmd := &cobra.Command{Use: "api", Short: "Constrained raw Postmark API escape hatch"}
	var tokenKind string
	var query []string
	raw := &cobra.Command{
		Use:   "get <api-path>",
		Short: "GET a raw Postmark API path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if !strings.HasPrefix(path, "/") {
				return agenterrors.New("raw API paths must start with /", agenterrors.FixableByAgent).
					WithHint("Use a Postmark API path such as '/bounces' or '/domains'.")
			}
			kind := api.ServerToken
			if tokenKind == "account" {
				kind = api.AccountToken
			} else if tokenKind != "server" {
				return agenterrors.New("unknown --token, expected server or account", agenterrors.FixableByAgent).
					WithHint("Use '--token server' for messages/bounces/webhooks/stats or '--token account' for servers/domains/signatures.")
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, kind, path, queryPairs(query))
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	raw.Flags().StringVar(&tokenKind, "token", "server", "Token kind: server or account")
	raw.Flags().StringArrayVar(&query, "query", nil, "Query parameter as key=value; repeatable")
	cmd.AddCommand(raw)
	root.AddCommand(cmd)
}
