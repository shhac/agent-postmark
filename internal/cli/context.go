package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-postmark/internal/api"
	"github.com/shhac/agent-postmark/internal/config"
	"github.com/shhac/agent-postmark/internal/credential"
	agenterrors "github.com/shhac/agent-postmark/internal/errors"
	"github.com/shhac/agent-postmark/internal/output"
)

type resolvedContext struct {
	Client        *api.Client
	Profile       string
	Host          string
	ServerID      int
	MessageStream string
	AccountToken  bool
	ServerToken   bool
}

var newAPIClient = api.New

func withClient(cmdCtx context.Context, flags *GlobalFlags, fn func(context.Context, *resolvedContext) error) error {
	resolved, err := resolve(flags)
	if err != nil {
		output.WriteError(output.Stderr(), err)
		return nil
	}
	ctx := cmdCtx
	if flags.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(flags.Timeout)*time.Millisecond)
		defer cancel()
	}
	if err := fn(ctx, resolved); err != nil {
		output.WriteError(output.Stderr(), err)
		return nil
	}
	return nil
}

func resolve(flags *GlobalFlags) (*resolvedContext, error) {
	cfg := config.Read()
	profileName := firstNonEmpty(flags.Profile, os.Getenv("AGENT_POSTMARK_PROFILE"), cfg.DefaultProfile)
	profile := config.Profile{}
	if profileName != "" {
		if found, ok := cfg.Profiles[profileName]; ok {
			profile = found
		}
	}

	host := firstNonEmpty(flags.Host, os.Getenv("AGENT_POSTMARK_BASE_URL"), profile.Host, os.Getenv("AGENT_POSTMARK_HOST"), config.DefaultHost)
	serverID := firstNonZero(flags.ServerID, profile.DefaultServer, envInt("AGENT_POSTMARK_SERVER_ID"), envInt("POSTMARK_SERVER_ID"))
	stream := firstNonEmpty(flags.MessageStream, profile.MessageStream, os.Getenv("AGENT_POSTMARK_MESSAGE_STREAM"), "outbound")

	accountToken := firstNonEmpty(flags.AccountToken, os.Getenv("AGENT_POSTMARK_ACCOUNT_TOKEN"), os.Getenv("POSTMARK_ACCOUNT_TOKEN"))
	serverToken := firstNonEmpty(flags.ServerToken, os.Getenv("AGENT_POSTMARK_SERVER_TOKEN"), os.Getenv("POSTMARK_SERVER_TOKEN"))
	accountStored := accountToken != ""
	serverStored := serverToken != ""
	if profileName != "" {
		if accountToken == "" {
			if token, err := credential.Get(profileName, credential.AccountToken); err == nil {
				accountToken = token
				accountStored = true
			}
		}
		if serverToken == "" {
			if token, err := credential.Get(profileName, credential.ServerToken); err == nil {
				serverToken = token
				serverStored = true
			}
		}
	}
	if accountToken == "" && serverToken == "" {
		return nil, agenterrors.New("missing Postmark credentials", agenterrors.FixableByHuman).
			WithHint("Run 'agent-postmark profiles add <profile> --form --account-token --server-token' or set direct env vars for local testing.")
	}
	client := newAPIClient(host, accountToken, serverToken)
	client.MaxRetries = flags.MaxRetries
	client.Debug = flags.Debug
	client.DebugOut = output.Stderr()
	return &resolvedContext{
		Client:        client,
		Profile:       profileName,
		Host:          host,
		ServerID:      serverID,
		MessageStream: stream,
		AccountToken:  accountStored,
		ServerToken:   serverStored,
	}, nil
}

func requireServer(ctx *resolvedContext) error {
	if ctx.ServerID == 0 {
		return agenterrors.New("missing server id", agenterrors.FixableByAgent).
			WithHint("Run 'agent-postmark servers list' or set one with 'agent-postmark profiles update <profile> --server <id>'.")
	}
	return nil
}

func writeItem(data any, flagFormat string) error {
	format, err := output.ResolveFormat(flagFormat, output.FormatJSON)
	if err != nil {
		output.WriteError(output.Stderr(), err)
		return nil
	}
	output.Print(data, format, true)
	return nil
}

func writeRaw(raw json.RawMessage, flagFormat string) error {
	format, err := output.ResolveFormat(flagFormat, output.FormatJSON)
	if err != nil {
		output.WriteError(output.Stderr(), err)
		return nil
	}
	output.WriteRawJSON(redactRaw(raw), format, true)
	return nil
}

func writeList(items []json.RawMessage, total, offset, count int, resource, flagFormat string, full bool) error {
	format, err := output.ResolveFormat(flagFormat, output.FormatNDJSON)
	if err != nil {
		output.WriteError(output.Stderr(), err)
		return nil
	}
	if format != output.FormatNDJSON {
		decoded := make([]any, 0, len(items))
		for _, raw := range items {
			var item any
			if err := json.Unmarshal(compactListItem(resource, raw, full), &item); err == nil {
				decoded = append(decoded, item)
			}
		}
		output.Print(map[string]any{"results": decoded, "total": total}, format, true)
		return nil
	}
	writer := output.NewNDJSONWriter(output.Stdout())
	for _, raw := range items {
		var item any
		if err := json.Unmarshal(compactListItem(resource, raw, full), &item); err != nil {
			return err
		}
		if err := writer.WriteItem(item); err != nil {
			return err
		}
	}
	if total > offset+count {
		return writer.WriteMetaLine("@pagination", output.Pagination{HasMore: true, TotalItems: total, NextOffset: offset + count})
	}
	return nil
}

func queryPairs(values []string) url.Values {
	q := url.Values{}
	for _, pair := range values {
		key, value, ok := stringsCut(pair, "=")
		if ok && key != "" {
			q.Add(key, value)
		}
	}
	return q
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func envInt(name string) int {
	value, _ := strconv.Atoi(os.Getenv(name))
	return value
}

func addCountOffsetFlags(cmd *cobra.Command, count *int, offset *int) {
	cmd.Flags().IntVar(count, "count", 50, "Number of records to request")
	cmd.Flags().IntVar(offset, "offset", 0, "Number of records to skip")
}

func stringsCut(s, sep string) (string, string, bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
