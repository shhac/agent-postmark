package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/output"
)

type GlobalFlags struct {
	Profile       string
	Host          string
	AccountToken  string
	ServerToken   string
	Server        string
	ServerID      int
	MessageStream string
	Format        string
	Timeout       int
	MaxRetries    int
	Debug         bool
	Full          bool
}

func newRootCmd(version string) *cobra.Command {
	globals := &GlobalFlags{}
	root := &cobra.Command{
		Use:           "agent-postmark",
		Short:         "Postmark delivery triage CLI for AI agents",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			applyConfiguredDefaults(cmd, globals)
		},
	}

	root.PersistentFlags().StringVarP(&globals.Profile, "profile", "p", "", "Profile alias (or AGENT_POSTMARK_PROFILE)")
	root.PersistentFlags().StringVar(&globals.Host, "host", "", "Postmark API host override")
	root.PersistentFlags().StringVar(&globals.AccountToken, "account-token", "", "Account token override; never printed or persisted")
	root.PersistentFlags().StringVar(&globals.ServerToken, "server-token", "", "Server token override; never printed or persisted")
	root.PersistentFlags().StringVar(&globals.Server, "server", "", "Server alias override within the selected profile")
	root.PersistentFlags().IntVar(&globals.ServerID, "server-id", 0, "Numeric Postmark server ID override")
	root.PersistentFlags().StringVar(&globals.MessageStream, "stream", "", "Message stream override, such as outbound or broadcasts")
	root.PersistentFlags().StringVarP(&globals.Format, "format", "f", "", "Output format: json, yaml, jsonl")
	root.PersistentFlags().IntVarP(&globals.Timeout, "timeout", "t", 0, "Request timeout in milliseconds")
	root.PersistentFlags().IntVar(&globals.MaxRetries, "max-retries", 2, "Maximum automatic retries for transient responses")
	root.PersistentFlags().BoolVarP(&globals.Debug, "debug", "d", false, "Log redacted HTTP debug records to stderr")
	root.PersistentFlags().BoolVar(&globals.Full, "full", false, "Return fuller API payloads where supported")

	registerUsage(root)
	registerConfig(root)
	registerAuth(root, globals)
	registerResources(root, globals)
	registerInvestigate(root, globals)
	registerAPI(root, globals)

	return root
}

func applyConfiguredDefaults(cmd *cobra.Command, globals *GlobalFlags) {
	cfg := config.Read()
	flags := cmd.Root().PersistentFlags()
	if cfg.Defaults.TimeoutMS != nil && !flags.Changed("timeout") {
		globals.Timeout = *cfg.Defaults.TimeoutMS
	}
	if cfg.Defaults.MaxRetries != nil && !flags.Changed("max-retries") {
		globals.MaxRetries = *cfg.Defaults.MaxRetries
	}
}

func Execute(version string) error {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		os.Args[1] = "profiles"
	}
	err := newRootCmd(version).Execute()
	if err != nil {
		output.WriteError(output.Stderr(), err)
	}
	return err
}
