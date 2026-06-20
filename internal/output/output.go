// Package output re-exports the shared output contract from lib-agent-output,
// keeping the internal/output import path while the wire mechanism (format
// parsing, JSON/YAML encoding, error rendering) lives in one place. What stays
// local is agent-postmark policy: the writer indirection used by tests and the
// Postmark-shaped pagination trailer. YAML support (and its yaml.v3 dependency)
// comes from the shared lib-agent-cli/yaml encoder, blank-imported below.
// (Migration shim.)
package output

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	_ "github.com/shhac/lib-agent-cli/yaml" // registers the shared YAML encoder for out.FormatYAML
	out "github.com/shhac/lib-agent-output"
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
		data = out.PruneNils(data)
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

// NDJSONWriter writes one JSON record per line with HTML escaping disabled. The
// writer and its item/meta methods come from the shared contract; only the
// Postmark-shaped Pagination trailer below stays local.
type NDJSONWriter = out.NDJSONWriter

var NewNDJSONWriter = out.NewNDJSONWriter

// Pagination is Postmark-shaped (a total-count + next-offset trailer), so it
// stays local rather than using out.Pagination.
type Pagination struct {
	HasMore    bool `json:"has_more"`
	TotalItems int  `json:"total_items,omitempty"`
	NextOffset int  `json:"next_offset,omitempty"`
}
