package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

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
				q := url.Values{"count": {strconv.Itoa(count)}, "offset": {strconv.Itoa(offset)}}
				raw, err := resolved.Client.Get(ctx, api.AccountToken, path, q)
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
	return getCommand(use, short, globals, api.AccountToken, pathFormat)
}

func serverGetCommand(use, short string, globals *GlobalFlags, pathFormat string) *cobra.Command {
	return getCommand(use, short, globals, api.ServerToken, pathFormat)
}

func getCommand(use, short string, globals *GlobalFlags, kind api.TokenKind, pathFormat string) *cobra.Command {
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
	if totalRaw, ok := payload["TotalCount"]; ok {
		_ = json.Unmarshal(totalRaw, &total)
	}
	var items []json.RawMessage
	if listRaw, ok := payload[field]; ok {
		_ = json.Unmarshal(listRaw, &items)
	}
	if count == 0 {
		count = len(items)
	}
	if total == 0 {
		total = len(items)
	}
	return writeList(items, total, offset, count, field, format, full)
}

func addIfSet(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func writeWebhookHealth(raw json.RawMessage, format string) error {
	var payload struct {
		Webhooks []map[string]any `json:"Webhooks"`
	}
	_ = json.Unmarshal(raw, &payload)
	rows := make([]json.RawMessage, 0, len(payload.Webhooks)+1)
	coverage := map[string]int{"delivery": 0, "bounce": 0, "inbound": 0, "spam_complaint": 0}
	for _, hook := range payload.Webhooks {
		triggers, _ := hook["Triggers"].(map[string]any)
		for key, value := range triggers {
			enabled, _ := value.(bool)
			if !enabled {
				continue
			}
			switch key {
			case "Delivery":
				coverage["delivery"]++
			case "Bounce":
				coverage["bounce"]++
			case "Inbound":
				coverage["inbound"]++
			case "SpamComplaint":
				coverage["spam_complaint"]++
			}
		}
		row, _ := json.Marshal(map[string]any{
			"type":           "entity",
			"object":         "webhook",
			"id":             hook["ID"],
			"message_stream": hook["MessageStream"],
			"triggers":       hook["Triggers"],
		})
		rows = append(rows, row)
	}
	severity := "ok"
	summary := "Webhook coverage includes delivery and bounce triggers."
	if coverage["delivery"] == 0 || coverage["bounce"] == 0 {
		severity = "warning"
		summary = "Webhook coverage is missing delivery or bounce triggers."
	}
	finding, _ := json.Marshal(map[string]any{
		"type":     "finding",
		"severity": severity,
		"summary":  summary,
		"coverage": coverage,
	})
	rows = append(rows, finding)
	return writeList(rows, len(rows), 0, len(rows), "WebhookHealth", format, true)
}
