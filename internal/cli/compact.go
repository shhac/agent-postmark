package cli

import "encoding/json"

func compactListItem(resource string, raw json.RawMessage, full bool) json.RawMessage {
	if full {
		return redactRaw(raw)
	}
	var item map[string]any
	if err := json.Unmarshal(redactRaw(raw), &item); err != nil {
		return redactRaw(raw)
	}

	var keys []string
	switch resource {
	case "Servers":
		keys = []string{"ID", "Name", "Color", "ServerLink"}
	case "MessageStreams":
		keys = []string{"ID", "Name", "ServerID", "MessageStreamType", "ArchivedAt"}
	case "Domains":
		keys = []string{"ID", "Name", "DKIMVerified", "SPFVerified", "ReturnPathDomainVerified"}
	case "SenderSignatures":
		keys = []string{"ID", "Name", "EmailAddress", "Confirmed", "DKIMVerified", "SPFVerified", "ReturnPathDomainVerified"}
	case "Webhooks":
		keys = []string{"ID", "Url", "MessageStream", "Triggers"}
	case "Messages":
		keys = []string{"MessageID", "Status", "MessageStream", "Tag", "ReceivedAt", "LastOpen", "LastClick"}
	case "InboundMessages":
		keys = []string{"MessageID", "Status", "FromName", "MailboxHash", "Date"}
	case "Bounces":
		keys = []string{"ID", "Type", "Name", "MessageID", "Inactive", "CanActivate", "MessageStream", "BouncedAt", "DumpAvailable"}
	case "Opens":
		keys = []string{"MessageID", "ReceivedAt", "FirstOpen", "LastOpen", "TotalOpens", "UniqueOpens", "OS", "Platform"}
	case "Clicks":
		keys = []string{"MessageID", "ReceivedAt", "FirstClick", "LastClick", "TotalClicks", "UniqueClicks", "OriginalLink", "OS", "Platform"}
	case "Suppressions":
		keys = []string{"EmailAddress", "SuppressionReason", "Origin", "CreatedAt", "Status", "Message"}
	case "Profiles":
		keys = []string{"profile", "default", "credential", "credentials", "host", "default_server", "servers"}
	case "ProfileServers":
		keys = []string{"profile", "server", "default", "server_id", "message_stream", "credential", "server_token_configured"}
	default:
		return redactRaw(raw)
	}

	compact := map[string]any{}
	for _, key := range keys {
		if value, ok := item[key]; ok {
			compact[key] = value
		}
	}
	if redacted, ok := item["@redacted"]; ok {
		compact["@redacted"] = redacted
	}
	out, err := json.Marshal(compact)
	if err != nil {
		return redactRaw(raw)
	}
	return out
}
