package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	Server        string
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
	serverAlias := firstNonEmpty(flags.Server, os.Getenv("AGENT_POSTMARK_SERVER"), profile.DefaultServer)
	if serverAlias == "" && len(profile.Servers) == 1 {
		for alias := range profile.Servers {
			serverAlias = alias
		}
	}
	serverProfile := profile.Servers[serverAlias]
	serverID := firstNonZero(flags.ServerID, serverProfile.ServerID, envInt("AGENT_POSTMARK_SERVER_ID"), envInt("POSTMARK_SERVER_ID"))
	stream := firstNonEmpty(flags.MessageStream, serverProfile.MessageStream, os.Getenv("AGENT_POSTMARK_MESSAGE_STREAM"), "outbound")

	accountToken := firstNonEmpty(flags.AccountToken, os.Getenv("AGENT_POSTMARK_ACCOUNT_TOKEN"), os.Getenv("POSTMARK_ACCOUNT_TOKEN"))
	serverToken := firstNonEmpty(flags.ServerToken, os.Getenv("AGENT_POSTMARK_SERVER_TOKEN"), os.Getenv("POSTMARK_SERVER_TOKEN"))
	accountStored := accountToken != ""
	serverStored := serverToken != ""
	if profileName != "" {
		if accountToken == "" {
			if token, err := credential.GetAccount(profileName); err == nil {
				accountToken = token
				accountStored = true
			}
		}
		if serverToken == "" && serverAlias != "" {
			if token, err := credential.GetServer(profileName, serverAlias); err == nil {
				serverToken = token
				serverStored = true
			}
		}
	}
	if accountToken == "" && serverToken == "" {
		return nil, agenterrors.New("missing Postmark credentials", agenterrors.FixableByHuman).
			WithHint("Run 'agent-postmark profiles add <profile> --form --account-token' and/or 'agent-postmark profiles servers add <profile> <server> --form --server-token --server-id <id>'.")
	}
	client := newAPIClient(host, accountToken, serverToken)
	client.MaxRetries = flags.MaxRetries
	client.Debug = flags.Debug
	client.DebugOut = output.Stderr()
	return &resolvedContext{
		Client:        client,
		Profile:       profileName,
		Host:          host,
		Server:        serverAlias,
		ServerID:      serverID,
		MessageStream: stream,
		AccountToken:  accountStored,
		ServerToken:   serverStored,
	}, nil
}

func requireServer(ctx *resolvedContext) error {
	if ctx.ServerID == 0 {
		return agenterrors.New("missing server id", agenterrors.FixableByAgent).
			WithHint("Run 'agent-postmark servers list' or set one with 'agent-postmark profiles servers update <profile> <server> --server-id <id>'.")
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
		key, value, ok := strings.Cut(pair, "=")
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
	runE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *count < 0 {
			output.WriteError(output.Stderr(), agenterrors.New("invalid --count", agenterrors.FixableByAgent).
				WithHint("Use --count 0 or a positive number. To mirror a UI page, use --offset (page-1)*count."))
			return nil
		}
		if *offset < 0 {
			output.WriteError(output.Stderr(), agenterrors.New("invalid --offset", agenterrors.FixableByAgent).
				WithHint("Use --offset 0 or a positive number. To mirror page 6 with --count 50, use --offset 250."))
			return nil
		}
		return runE(cmd, args)
	}
}
