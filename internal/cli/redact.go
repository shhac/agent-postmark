package cli

import (
	"encoding/json"
	"net/url"
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
		paths = uniqueStrings(paths)
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
			if redacted, ok := redactURLUserinfo(child); ok {
				out[k] = redacted
				paths = append(paths, nextPath)
				continue
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

func redactURLUserinfo(value any) (string, bool) {
	raw, ok := value.(string)
	if !ok || raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	out := parsed.Scheme + "://[REDACTED]@" + parsed.Host
	if parsed.Opaque != "" {
		out += parsed.Opaque
	}
	out += parsed.EscapedPath()
	if parsed.RawQuery != "" {
		out += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		out += "#" + parsed.EscapedFragment()
	}
	return out, true
}

func shouldRedact(key string, value any) bool {
	if safeTokenMetadataKeys[key] && safeTokenMetadataValue(key, value) {
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

func safeTokenMetadataValue(key string, value any) bool {
	switch key {
	case "account_token_configured", "server_token_configured":
		_, ok := value.(bool)
		return ok
	case "server_tokens_configured":
		servers, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for _, configured := range servers {
			if _, ok := configured.(bool); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func redactableValue(value any) bool {
	switch value.(type) {
	case string, []any, map[string]any:
		return true
	default:
		return false
	}
}

func uniqueStrings(sorted []string) []string {
	if len(sorted) < 2 {
		return sorted
	}
	out := sorted[:1]
	for _, value := range sorted[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
