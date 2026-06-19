// Package output re-exports the shared output contract from lib-agent-output,
// keeping the internal/output import path while the wire mechanism (format
// parsing, JSON/YAML encoding, error rendering) lives in one place. What stays
// local is agent-postmark policy: the writer indirection used by tests, the
// Postmark-shaped pagination trailer, and the YAML number-normalization that
// renders whole floats as ints. (Migration shim.)
package output

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"os"
	"sync"

	out "github.com/shhac/lib-agent-output"
	"gopkg.in/yaml.v3"
)

// Format and its values come from the shared contract; ParseFormat is therefore
// the family's lenient parser (accepts "ndjson" as well as "jsonl",
// case-insensitive).
type Format = out.Format

const (
	FormatJSON   = out.FormatJSON
	FormatYAML   = out.FormatYAML
	FormatNDJSON = out.FormatNDJSON
)

var (
	ParseFormat   = out.ParseFormat
	ResolveFormat = out.ResolveFormat
	WriteError    = out.WriteError
)

// init registers agent-postmark's YAML encoder with lib-agent-output, so YAML
// support (and its yaml.v3 dependency) stays in this CLI while the core library
// remains dependency-free. The encoder normalizes whole-number floats to ints
// so JSON-decoded numbers render as 5 rather than 5.0.
func init() {
	out.RegisterEncoder(out.FormatYAML, func(v any) ([]byte, error) {
		// v arrives already JSON-decoded and pruned by Print, so the encoder
		// only normalizes numbers (whole floats -> ints) and renders YAML.
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(normalizeYAMLNumbers(v)); err != nil {
			return nil, err
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	})
}

var (
	writersMu sync.Mutex
	stdout    io.Writer = os.Stdout
	stderr    io.Writer = os.Stderr
)

func Stdout() io.Writer {
	writersMu.Lock()
	defer writersMu.Unlock()
	return stdout
}

func Stderr() io.Writer {
	writersMu.Lock()
	defer writersMu.Unlock()
	return stderr
}

func SetWriters(o, e io.Writer) func() {
	writersMu.Lock()
	oldOut, oldErr := stdout, stderr
	if o != nil {
		stdout = o
	}
	if e != nil {
		stderr = e
	}
	writersMu.Unlock()
	return func() {
		writersMu.Lock()
		stdout, stderr = oldOut, oldErr
		writersMu.Unlock()
	}
}

// Print prunes (when requested) then encodes data in the given format via the
// shared encoder. JSON prunes the typed value directly (no round-trip), exactly
// as the pre-migration printJSON did. YAML first round-trips through JSON so its
// number-normalization and pruning operate on decoded maps, matching the old
// printYAML.
func Print(data any, format Format, prune bool) {
	if format == FormatYAML {
		b, err := json.Marshal(data)
		if err != nil {
			return
		}
		var decoded any
		if err := json.Unmarshal(b, &decoded); err != nil {
			return
		}
		data = decoded
	}
	if prune {
		data = pruneNulls(data)
	}
	_ = out.Print(Stdout(), data, format, nil)
}

func WriteRawJSON(raw json.RawMessage, format Format, prune bool) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return
	}
	Print(decoded, format, prune)
}

// NDJSONWriter writes one JSON record per line with HTML escaping disabled.
type NDJSONWriter struct {
	enc *json.Encoder
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONWriter{enc: enc}
}

func (n *NDJSONWriter) WriteItem(item any) error {
	return n.enc.Encode(item)
}

func (n *NDJSONWriter) WriteMetaLine(key string, value any) error {
	return n.enc.Encode(map[string]any{key: value})
}

// Pagination is Postmark-shaped (a total-count + next-offset trailer), so it
// stays local rather than using out.Pagination.
type Pagination struct {
	HasMore    bool `json:"has_more"`
	TotalItems int  `json:"total_items,omitempty"`
	NextOffset int  `json:"next_offset,omitempty"`
}

func normalizeYAMLNumbers(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			val[k] = normalizeYAMLNumbers(child)
		}
		return val
	case []any:
		for i, child := range val {
			val[i] = normalizeYAMLNumbers(child)
		}
		return val
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) || math.Trunc(val) != val {
			return val
		}
		return int64(val)
	default:
		return v
	}
}

func pruneNulls(v any) any {
	switch val := v.(type) {
	case map[string]any:
		o := make(map[string]any, len(val))
		for k, child := range val {
			if child == nil {
				continue
			}
			o[k] = pruneNulls(child)
		}
		return o
	case []any:
		o := make([]any, len(val))
		for i, child := range val {
			o[i] = pruneNulls(child)
		}
		return o
	default:
		return v
	}
}
