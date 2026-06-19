package output

import "testing"

// Migrating ParseFormat onto lib-agent-output makes it intentionally more
// lenient than the old hand-rolled switch: it accepts the "ndjson" alias and is
// case-insensitive, while still rejecting unknown formats. This pins that
// contract so a future lib change can't silently narrow or widen it.
func TestParseFormatLenientAliases(t *testing.T) {
	cases := map[string]Format{
		"json":   FormatJSON,
		"JSON":   FormatJSON,
		"yaml":   FormatYAML,
		"YAML":   FormatYAML,
		"yml":    FormatYAML,
		"jsonl":  FormatNDJSON,
		"ndjson": FormatNDJSON,
		"NDJSON": FormatNDJSON,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil {
			t.Errorf("ParseFormat(%q) returned error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseFormatRejectsUnknown(t *testing.T) {
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatal("ParseFormat(\"xml\") = nil error, want rejection")
	}
}

func TestResolveFormatEmptyUsesDefault(t *testing.T) {
	got, err := ResolveFormat("", FormatNDJSON)
	if err != nil {
		t.Fatalf("ResolveFormat: %v", err)
	}
	if got != FormatNDJSON {
		t.Errorf("ResolveFormat(\"\", ndjson) = %q, want %q", got, FormatNDJSON)
	}
}
