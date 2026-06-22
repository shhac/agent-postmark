package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/output"
	libcli "github.com/shhac/lib-agent-cli/cli"
	agentmcp "github.com/shhac/lib-agent-mcp"
)

// GlobalFlags carries the persistent flags shared by every command. The shared
// --format/--timeout/--debug live in the embedded libcli.Globals; the rest are
// Postmark-specific (profile/credential/server/stream selection).
type GlobalFlags struct {
	libcli.Globals // Format, TimeoutMS, Debug

	Profile       string
	Host          string
	AccountToken  string
	ServerToken   string
	Server        string
	ServerID      int
	MessageStream string
	MaxRetries    int
	Full          bool
}

func newRootCmd(version string) *cobra.Command {
	globals := &GlobalFlags{}

	var root *cobra.Command
	root = libcli.NewRoot(libcli.Options{
		Use:            "agent-postmark",
		Short:          "Postmark delivery triage CLI for AI agents",
		Version:        version,
		Globals:        &globals.Globals,
		Redacts:        true,
		DefaultFormat:  output.FormatNDJSON,
		ConfigDefaults: func() { applyConfiguredDefaults(root, globals) },
		UnknownHint:    "run 'agent-postmark usage' to see the available domains",
	})

	pf := root.PersistentFlags()
	pf.StringVarP(&globals.Profile, "profile", "p", "", "Profile alias (or AGENT_POSTMARK_PROFILE)")
	pf.StringVar(&globals.Host, "host", "", "Postmark API host override")
	pf.StringVar(&globals.AccountToken, "account-token", "", "Account token override; never printed or persisted")
	pf.StringVar(&globals.ServerToken, "server-token", "", "Server token override; never printed or persisted")
	pf.StringVar(&globals.Server, "server", "", "Server alias override within the selected profile")
	pf.IntVar(&globals.ServerID, "server-id", 0, "Numeric Postmark server ID override")
	pf.StringVar(&globals.MessageStream, "stream", "", "Message stream override, such as outbound or broadcasts")
	pf.IntVar(&globals.MaxRetries, "max-retries", 2, "Maximum automatic retries for transient responses")
	pf.BoolVar(&globals.Full, "full", false, "Return fuller API payloads where supported")

	registerUsage(root)
	registerConfig(root)
	registerAuth(root, globals)
	registerResources(root, globals)
	registerInvestigate(root, globals)
	registerAPI(root, globals)

	installGroupUnknownHandlers(root)

	// Expose the whole command tree as an MCP server (added last, so it reflects
	// the complete tree). --color/--expose are output-shaping, irrelevant to a
	// tool call, so hide them from the generated schemas.
	root.AddCommand(agentmcp.Command(root, agentmcp.WithHiddenFlags("color", "expose")))

	return root
}

// installGroupUnknownHandlers walks the tree and gives every group node (one
// with subcommands but no action of its own) the same structured
// unknown-subcommand behaviour the root already has: an unknown subcommand
// returns a fixable_by:agent error listing the group's valid subcommands rather
// than cobra's usage text, and no args falls back to help. Done as a post-pass
// so each group is already attached and CommandPath() resolves the full path
// for the hint (e.g. "agent-postmark profiles servers").
func installGroupUnknownHandlers(root *cobra.Command) {
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, sub := range cmd.Commands() {
			if len(sub.Commands()) > 0 && sub.RunE == nil && sub.Run == nil {
				libcli.HandleUnknownCommand(sub, "run '"+sub.CommandPath()+" --help' to see the available subcommands")
			}
			walk(sub)
		}
	}
	walk(root)
}

func applyConfiguredDefaults(cmd *cobra.Command, globals *GlobalFlags) {
	cfg := config.Read()
	flags := cmd.Root().PersistentFlags()
	if cfg.Defaults.TimeoutMS != nil && !flags.Changed("timeout") {
		globals.TimeoutMS = *cfg.Defaults.TimeoutMS
	}
	if cfg.Defaults.MaxRetries != nil && !flags.Changed("max-retries") {
		globals.MaxRetries = *cfg.Defaults.MaxRetries
	}
	SetExpose(globals.Expose)
}

// NewRootCmd builds the root command after applying the auth→profiles alias
// rewrite, so main can hand it straight to libcli.Run.
func NewRootCmd(version string) *cobra.Command {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		os.Args[1] = "profiles"
	}
	return newRootCmd(version)
}

// Execute builds the root, runs it, and renders any bubbled error as the
// family's structured JSON on stderr exactly once, returning it for the caller
// to set an exit code. main uses libcli.Run instead (render + exit); Execute is
// the package-level seam that tests drive to cover the top-level error sink.
func Execute(version string) error {
	err := NewRootCmd(version).Execute()
	if err != nil {
		output.WriteError(output.Stderr(), err)
	}
	return err
}
