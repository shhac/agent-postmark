package cli

import (
	"encoding/json"
	"sort"
	"strings"
)

var sensitiveKeys = map[string]bool{
	"HtmlBody":         true,
	"TextBody":         true,
	"Body":             true,
	"Content":          true,
	"Headers":          true,
	"Attachments":      true,
	"Metadata":         true,
	"Message":          false,
	"Subject":          true,
	"Email":            true,
	"From":             true,
	"To":               true,
	"Cc":               true,
	"Bcc":              true,
	"ReplyTo":          true,
	"OriginalEmail":    true,
	"Recipient":        true,
	"Recipients":       true,
	"EmailAddress":     true,
	"ReturnPathDomain": false,
}

var safeTokenMetadataKeys = map[string]bool{
	"account_token_configured": true,
	"server_token_configured":  true,
	"server_tokens_configured": true,
}

func redactRaw(raw json.RawMessage) json.RawMessage {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return raw
	}
	redacted, paths := redactValue(decoded, "")
	if len(paths) > 0 {
		sort.Strings(paths)
		if m, ok := redacted.(map[string]any); ok {
			m["@redacted"] = paths
		}
	}
	out, err := json.Marshal(redacted)
	if err != nil {
		return raw
	}
	return out
}

func redactValue(v any, path string) (any, []string) {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		var paths []string
		for k, child := range val {
			nextPath := k
			if path != "" {
				nextPath = path + "." + k
			}
			if shouldRedact(k, child) {
				out[k] = "[REDACTED]"
				paths = append(paths, nextPath)
				continue
			}
			redacted, childPaths := redactValue(child, nextPath)
			out[k] = redacted
			paths = append(paths, childPaths...)
		}
		return out, paths
	case []any:
		out := make([]any, len(val))
		var paths []string
		for i, child := range val {
			redacted, childPaths := redactValue(child, path)
			out[i] = redacted
			paths = append(paths, childPaths...)
		}
		return out, paths
	default:
		return v, nil
	}
}

func shouldRedact(key string, value any) bool {
	if safeTokenMetadataKeys[key] {
		return false
	}
	if sensitiveKeys[key] {
		return redactableValue(value)
	}
	lower := strings.ToLower(key)
	if strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
		return redactableValue(value)
	}
	return false
}

func redactableValue(value any) bool {
	switch value.(type) {
	case string, []any, map[string]any:
		return true
	default:
		return false
	}
}
