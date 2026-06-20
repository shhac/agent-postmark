package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	libcli "github.com/shhac/lib-agent-cli/cli"

	"github.com/shhac/agent-postmark/internal/api"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

func accountListCommand(use, short string, globals *GlobalFlags, path, envelope string) *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, api.AccountToken, path, paginationQuery(count, offset))
				if err != nil {
					return err
				}
				return writeEnvelopeListWithPage(raw, envelope, offset, count, globals.Format, globals.Full)
			})
		},
	}
	addCountOffsetFlags(cmd, &count, &offset)
	return cmd
}

func accountGetCommand(use, short string, globals *GlobalFlags, pathFormat string) *cobra.Command {
	return multiGetCommand(use, short, globals, api.AccountToken, pathFormat)
}

func serverGetCommand(use, short string, globals *GlobalFlags, pathFormat string) *cobra.Command {
	return multiGetCommand(use, short, globals, api.ServerToken, pathFormat)
}

// multiGetCommand replaces the old single-only getCommand: it accepts 1..N IDs
// and routes each through getEntities, yielding the family get contract (NDJSON
// by default; one record or {"@unresolved":…} per input ID in order).
func multiGetCommand(use, short string, globals *GlobalFlags, kind api.TokenKind, pathFormat string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short + " (one or more IDs)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getEntities(cmd.Context(), globals, kind, args, pathFormat)
		},
	}
}

// getEntities runs the family's multi-capable get for the postmark domain: it
// sets up one client, then resolves each id through the Postmark API and
// streams the result per the shared get contract (NDJSON by default — one
// record or {"@unresolved":…} per id in input order; item-level misses, such
// as 404s, stay on stdout; command-level failures bubble to the single sink).
func getEntities(cmdCtx context.Context, globals *GlobalFlags, kind api.TokenKind, args []string, pathFormat string) error {
	resolved, err := resolve(globals)
	if err != nil {
		return err
	}
	ctx := cmdCtx
	return libcli.EntityGet(os.Stdout, globals.Format, args, func(id string) (any, error) {
		raw, err := resolved.Client.Get(ctx, kind, fmt.Sprintf(pathFormat, id), url.Values{})
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	})
}

// singleGetCommand is the old single-ID-only variant; used for non-entity
// fetches (dump, inbound-retry, etc.) where multi-get semantics are not wanted.
func singleGetCommand(use, short string, globals *GlobalFlags, kind api.TokenKind, pathFormat string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Get(ctx, kind, fmt.Sprintf(pathFormat, args[0]), url.Values{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
}

func accountPostCommand(use, short string, globals *GlobalFlags, pathFormat string, warning string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmMutation(yes, warning) {
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Post(ctx, api.AccountToken, fmt.Sprintf(pathFormat, args[0]), url.Values{}, map[string]any{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm this state-changing Postmark request")
	return cmd
}

func serverPutCommand(use, short string, globals *GlobalFlags, pathFormat string, warning string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmMutation(yes, warning) {
				return nil
			}
			return withClient(cmd.Context(), globals, func(ctx context.Context, resolved *resolvedContext) error {
				raw, err := resolved.Client.Put(ctx, api.ServerToken, fmt.Sprintf(pathFormat, args[0]), url.Values{}, map[string]any{})
				if err != nil {
					return err
				}
				return writeRaw(raw, globals.Format)
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm this state-changing Postmark request")
	return cmd
}

func confirmMutation(yes bool, warning string) bool {
	if yes {
		return true
	}
	output.WriteError(output.Stderr(), agenterrors.New("mutation requires --yes", agenterrors.FixableByHuman).WithHint(warning))
	return false
}

func writeEnvelopeList(raw json.RawMessage, field, format string, full bool) error {
	return writeEnvelopeListWithPage(raw, field, 0, 0, format, full)
}

func writeEnvelopeListWithPage(raw json.RawMessage, field string, offset, count int, format string, full bool) error {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return writeRaw(raw, format)
	}
	var total int
	hasTotal := false
	if totalRaw, ok := payload["TotalCount"]; ok {
		hasTotal = true
		_ = json.Unmarshal(totalRaw, &total)
	}
	var items []json.RawMessage
	if listRaw, ok := payload[field]; ok {
		_ = json.Unmarshal(listRaw, &items)
	}
	if !hasTotal {
		total = len(items)
		items = localPage(items, offset, count)
	}
	if count == 0 {
		count = len(items)
	}
	if total == 0 {
		total = len(items)
	}
	return writeList(items, total, offset, count, field, format, full)
}

func localPage(items []json.RawMessage, offset, count int) []json.RawMessage {
	if offset < 0 {
		offset = 0
	}
	if offset > len(items) {
		offset = len(items)
	}
	if count <= 0 {
		return items[offset:]
	}
	end := offset + count
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func addIfSet(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func paginationQuery(count, offset int) url.Values {
	return url.Values{"count": {strconv.Itoa(count)}, "offset": {strconv.Itoa(offset)}}
}

func writeWebhookHealth(raw json.RawMessage, format string) error {
	webhooks := rawEnvelopeList(raw, "Webhooks")
	rows := make([]json.RawMessage, 0, len(webhooks)+1)
	for _, row := range webhooks {
		hook := rawObject(row)
		row, _ := json.Marshal(map[string]any{
			"type":           "entity",
			"object":         "webhook",
			"id":             hook["ID"],
			"message_stream": hook["MessageStream"],
			"triggers":       hook["Triggers"],
		})
		rows = append(rows, row)
	}
	finding := webhookCoverageFinding(webhookCoverage(webhooks))
	if data, ok := finding["data"].(map[string]any); ok {
		finding["coverage"] = data["coverage"]
		delete(finding, "data")
	}
	rawFinding, _ := json.Marshal(finding)
	rows = append(rows, rawFinding)
	return writeList(rows, len(rows), 0, len(rows), "WebhookHealth", format, true)
}
