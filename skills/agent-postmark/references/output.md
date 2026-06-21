# agent-postmark output

Use this file when parsing command output, errors, redaction, or mutation
guards. For command selection, use `scenarios.md` or `investigations.md`.

## Formats

- Lists default to NDJSON (`jsonl`).
- Gets (`get <id>...`) default to NDJSON — one line per id.
- Investigations always emit NDJSON evidence records.
- `--format json` gives a pretty object (or `{"data":[…],"@unresolved":[…]}`
  envelope for multi-get). `--format yaml` is available for humans on most
  non-investigation commands.

## Get Contract (single + multi)

`get <id>...` takes one or more ids and returns one result per id, in input
order. Default output is NDJSON: one line per id — the record, or
`{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't
be resolved (e.g. not found / bad id). `--format json|yaml` collapses to one
`{"data":[…], "@unresolved":[…]}` envelope. A single `get <id>` is just the
one-element case (NDJSON one line by default; was pretty JSON before — pass
`--format json` for the object). Item-level misses stay on stdout and exit 0;
only a command-level failure (auth, network) goes to stderr with exit 1 and
empty stdout.

## Evidence Records

Investigations emit:

```json
{"type":"entity","object":"bounce","id":9001,"data":{}}
{"type":"finding","severity":"critical","summary":"Recipient is inactive because of this bounce; future delivery may be suppressed.","data":{}}
{"type":"next_command","command":"agent-postmark suppressions check <email>","reason":"Check whether the bounced recipient is currently suppressed."}
```

Severities:

- `ok`: no obvious issue in the checked surface.
- `info`: relevant activity exists but is not necessarily a problem.
- `warning`: likely issue or incomplete setup.
- `critical`: delivery is probably blocked or suppressed.

## Redaction

The CLI separates compacting from redaction:

- list/search output omits bulky bodies, headers, and attachments by default
- subjects and addressing fields are visible for delivery triage
- single-resource output can include bodies, headers, attachments, and metadata
- `messages content <message-id> [message-id...]` explicitly retrieves outbound
  email content from one or more message IDs
- keys containing token or secret, URL credentials, and Postmark `OriginalEmail`
  raw email blobs are redacted

When possible, redacted resources include `@redacted` paths. Do not infer the
redacted content.

## Errors

Errors are JSON on stderr:

```json
{"error":"mutation requires --yes","fixable_by":"human","hint":"Creating a suppression blocks future delivery to this recipient."}
```

`fixable_by` values:

- `agent`: change arguments, IDs, streams, or command shape.
- `human`: auth, permissions, setup, or confirmed mutation required.
- `retry`: rate limits, network errors, or transient Postmark failures.

## Mutation Guards

Commands that can change Postmark state require `--yes`:

- domain DKIM/SPF verification
- bounce activation
- suppression create/delete
- inbound retry/bypass

The guard is intentionally `fixable_by:"human"` so an LLM does not silently add
`--yes` without the user's request.
