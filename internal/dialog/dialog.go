// Package dialog delegates the native secret-entry boilerplate to
// lib-agent-cli/dialog; this thin wrapper keeps agent-postmark's existing
// PromptSecret signature (with an initial value to edit). (Migration shim.)
package dialog

import (
	"context"

	clidialog "github.com/shhac/lib-agent-cli/dialog"
)

// PromptSecret opens a masked native prompt seeded with initial, so a token
// never transits argv. It returns a structured error on a headless host.
func PromptSecret(ctx context.Context, title, label, initial string) (string, error) {
	results, err := clidialog.Prompt(ctx, clidialog.Spec{
		Title: title,
		Items: []clidialog.Field{{ID: "secret", Label: label, InputType: clidialog.Password, Initial: initial}},
	})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return results[0].Value, nil
}

// Cancelled reports whether err is a dialog the user dismissed, which the
// caller can recover from by re-running (fixable_by retry) rather than
// treating as an environment problem (fixable_by human).
func Cancelled(err error) bool {
	cat, _ := clidialog.ClassifyError(err)
	return cat == clidialog.CategoryRetry
}
