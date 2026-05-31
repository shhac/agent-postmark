package cli

import (
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func writeProfileHumanError(err error, hint string) {
	output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).WithHint(hint))
}

func writeProfileAgentError(message string, hint string) {
	output.WriteError(output.Stderr(), agenterrors.New(message, agenterrors.FixableByAgent).WithHint(hint))
}

func writeProfileCommandError(err error) {
	writeProfileHumanError(err, "Run 'agent-postmark profiles list' to see configured profiles.")
}
