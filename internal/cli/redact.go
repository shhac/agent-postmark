package cli

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	out "github.com/shhac/lib-agent-output"
)

var sensitiveKeys = map[string]bool{
	"Message":          false,
	"OriginalEmail":    true,
	"ReturnPathDomain": false,
}

var safeTokenMetadataKeys = map[string]bool{
	"account_token_configured": true,
	"server_token_configured":  true,
	"server_tokens_configured": true,
}

// redactExpose is the --expose allowlist, set once from the global flag.
var redactExpose []string

// SetExpose records the --expose allowlist used by redaction (the global
// --expose flag, wired from the root command's defaults hook).
func SetExpose(expose []string) { redactExpose = expose }

// redactRaw masks agent-postmark's sensitive fields in a raw JSON document.
//
// The shared out.Redact owns the field-masking MECHANISM (the walk, the
// [REDACTED] placeholder, the @redacted notes, and --expose handling);
// postmarkSecrets() is the local POLICY (which keys/values are secret).
//
// URL userinfo (https://user:pass@host/…) is a PARTIAL-value transform that the
// shared whole-value redactor deliberately does not own, so it stays here as a
// local pass layered after field masking — it only touches values a sensitive
// key has not already fully masked, preserving "key sensitivity wins".
func redactRaw(raw json.RawMessage) json.RawMessage {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return raw
	}
	redacted := out.Redact(decoded, postmarkSecrets(), redactExpose)
	redacted = maskURLUserinfo(redacted)
	result, err := marshalJSONNoEscape(redacted)
	if err != nil {
		return raw
	}
	return result
}

// postmarkSecrets is agent-postmark's redaction POLICY: mask token/secret-named
// fields (with the documented safe-shape exceptions for *_configured presence
// metadata) and the OriginalEmail field.
func postmarkSecrets() out.RedactRule {
	return func(field out.RedactField) bool {
		return shouldRedact(field.Key, field.Value)
	}
}

func marshalJSONNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// maskURLUserinfo walks the (already field-redacted) tree and rewrites any URL
// string carrying userinfo to https://[REDACTED]@host/…, recording each masked
// path in the @redacted note list. Values a sensitive key already masked are the
// literal placeholder, which is not a URL, so they are left untouched.
func maskURLUserinfo(value any) any {
	masked, notes := maskURLUserinfoWalk(value, "")
	if len(notes) == 0 {
		return masked
	}
	m, ok := masked.(map[string]any)
	if !ok {
		return masked
	}
	if existing, ok := m[out.MetaKeyRedacted].([]out.RedactionNote); ok {
		m[out.MetaKeyRedacted] = append(existing, notes...)
		return m
	}
	m[out.MetaKeyRedacted] = notes
	return m
}

func maskURLUserinfoWalk(value any, path string) (any, []out.RedactionNote) {
	switch val := value.(type) {
	case map[string]any:
		var notes []out.RedactionNote
		for key, child := range val {
			if key == out.MetaKeyRedacted {
				continue
			}
			childPath := joinRedactPath(path, key)
			if masked, ok := redactURLUserinfo(child); ok {
				val[key] = masked
				notes = append(notes, out.RedactionNote{Path: childPath, Reason: "url_userinfo"})
				continue
			}
			redacted, childNotes := maskURLUserinfoWalk(child, childPath)
			val[key] = redacted
			notes = append(notes, childNotes...)
		}
		return val, notes
	case []any:
		var notes []out.RedactionNote
		itemPath := path + "[]"
		for i, child := range val {
			redacted, childNotes := maskURLUserinfoWalk(child, itemPath)
			val[i] = redacted
			notes = append(notes, childNotes...)
		}
		return val, notes
	default:
		return value, nil
	}
}

func joinRedactPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
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
	masked := parsed.Scheme + "://[REDACTED]@" + parsed.Host
	if parsed.Opaque != "" {
		masked += parsed.Opaque
	}
	masked += parsed.EscapedPath()
	if parsed.RawQuery != "" {
		masked += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		masked += "#" + parsed.EscapedFragment()
	}
	return masked, true
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
