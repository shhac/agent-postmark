package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func registerConfig(root *cobra.Command) {
	cmd := &cobra.Command{Use: "config", Short: "Inspect and update non-secret config"}
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeItem(map[string]any{"path": config.ConfigPath()}, "")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show non-secret config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeItem(config.Read(), "")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set <timeout_ms|max_retries> <value>",
		Short: "Set a persisted default",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := strconv.Atoi(args[1])
			if err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent))
				return nil
			}
			if err := config.SetDefaultValue(args[0], value); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent))
				return nil
			}
			return writeItem(map[string]any{"status": "set", "key": args[0], "value": value}, "")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "unset <timeout_ms|max_retries>",
		Short: "Unset a persisted default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.UnsetDefaultValue(args[0]); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent))
				return nil
			}
			return writeItem(map[string]any{"status": "unset", "key": args[0]}, "")
		},
	})
	root.AddCommand(cmd)
}
