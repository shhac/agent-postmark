package cli

import (
	"github.com/shhac/agent-postmark/internal/dialog"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func writeProfileHumanError(err error, hint string) {
	output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByHuman).WithHint(hint))
}

// writeProfileDialogError surfaces a native-dialog failure. A user-cancelled
// dialog is recoverable by re-running (fixable_by retry); a headless or
// unsupported host needs a human to move to a graphical session, so it keeps
// humanHint.
func writeProfileDialogError(err error, humanHint string) {
	if dialog.Cancelled(err) {
		output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Re-run the command to retry the prompt."))
		return
	}
	writeProfileHumanError(err, humanHint)
}

func writeProfileAgentError(message string, hint string) {
	output.WriteError(output.Stderr(), agenterrors.New(message, agenterrors.FixableByAgent).WithHint(hint))
}

func writeProfileCommandError(err error) {
	writeProfileHumanError(err, "Run 'agent-postmark profiles list' to see configured profiles.")
}
