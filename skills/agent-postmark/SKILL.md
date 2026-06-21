---
name: agent-postmark
description: |
  Triage and investigate Postmark delivery, bounces, outbound and inbound messages, suppressions, sender domains, sender signatures, message streams, webhooks, and server/account configuration. Use when:
  - Explaining why an email did not arrive, bounced, or was suppressed
  - Checking Postmark message status, bounce state, inactive recipients, opens, clicks, or inbound processing
  - Inspecting sender domain, DKIM, SPF, Return-Path, or sender signature health
  - Checking Postmark webhooks, message streams, servers, or delivery stats
  - Looking up Postmark servers, bounces, messages, domains, signatures, suppressions, or webhook configuration
  Triggers: "postmark", "email delivery", "bounce", "hard bounce", "suppression", "inactive recipient", "message stream", "sender signature", "DKIM", "SPF", "Return-Path", "webhook delivery", "inbound email", "email opens", "email clicks"
allowed-tools: Bash(agent-postmark *) Bash(mockpostmark *) Read Grep Glob
---

# agent-postmark

Use `agent-postmark` for Postmark delivery incidents, bounce/suppression
questions, message status, sender/domain configuration, message streams, and
webhooks.

## Safety

- Never ask the tool to reveal account or server tokens.
- Never accept pasted Postmark tokens in chat. Ask the user to run
  `agent-postmark profiles add <profile> --form --account-token` and/or
  `agent-postmark profiles servers add <profile> <server> --form --server-token --server-id <id>`
  locally so tokens go directly into OS dialogs.
- For initial setup with multiple server tokens, use
  `agent-postmark profiles setup <profile> --form --account-token --server app:<id>:outbound --server billing:<id>:outbound`.
- Use `agent-postmark profiles update <profile> --form --account-token` or
  `agent-postmark profiles servers update <profile> <server> --form --server-token`
  when a stored token needs replacement.
- Use `agent-postmark profiles servers remove <profile> <server>` to remove a
  server context and its stored token.
- Prefer read-only commands.
- Remember token scope: account-token commands handle servers, domains, and
  signatures; server-token commands handle message streams, messages, bounces,
  stats, suppressions, and webhooks.
- Use `agent-postmark suppressions list`, not a raw suppression dump, when
  browsing suppressions. Postmark's dump endpoint can return very large full
  exports and is intentionally not exposed as an agent-facing command.
- Subjects and addressing fields are visible for triage. Use list/search output
  first; ask for `messages content` only when the user needs actual body,
  header, or attachment details.
- Do not add `--yes` to mutation commands unless the user explicitly asks for
  that state change.

## Start Here

```bash
agent-postmark usage
agent-postmark profiles list
agent-postmark profiles check
agent-postmark config show
```

For incident-style questions, prefer investigations before low-level resource
commands:

```bash
agent-postmark investigate delivery --email user@example.com
agent-postmark investigate bounce <bounce-id>
agent-postmark investigate domain-health example.com
agent-postmark investigate stream-health --stream outbound
agent-postmark investigate webhook-health
```

For local testing, run `mockpostmark` and set `AGENT_POSTMARK_BASE_URL`.

## Output

Lists and investigations default to NDJSON. Gets default to NDJSON (one line
per id). Pass `--format json` for a single pretty object, or `--format yaml`
for YAML. Errors are JSON on stderr with `error`, `fixable_by`, and usually
`hint`.

**Get (single + multi).** `get <id>...` takes one or more ids and returns one
result per id, in input order. Default output is NDJSON: one line per id —
the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id
that couldn't be resolved (e.g. not found / bad id). `--format json|yaml`
collapses to one `{"data":[…], "@unresolved":[…]}` envelope. A single
`get <id>` is just the one-element case (NDJSON one line by default; was pretty
JSON before — pass `--format json` for the object). Item-level misses stay on
stdout and exit 0; only a command-level failure (auth, network) goes to stderr
with exit 1 and empty stdout.

Subjects and addressing fields are visible for delivery triage. List output
omits bulky bodies, headers, and attachments by default; use
`agent-postmark messages content <message-id> [message-id...]` when the user
needs the actual email content. Tokens, secrets, URL credentials, and original
raw email blobs are redacted with `"[REDACTED]"` and top-level `@redacted`
paths when possible.

Investigations emit evidence records:

```json
{"type":"entity","object":"bounce","id":9001,"data":{}}
{"type":"finding","severity":"critical","summary":"...","data":{}}
{"type":"next_command","command":"agent-postmark suppressions check <email>","reason":"..."}
```

Non-secret profile/config metadata lives in XDG config. Tokens live in Keychain.
`profiles list` and `profiles check` show token presence but never token values.
Use `--server <alias>` to select a stored server context; use `--server-id <id>`
only when a command needs a numeric Postmark server ID override.

## Incremental References

Load only the reference needed for the current task:

- [references/scenarios.md](references/scenarios.md): use when the user asks a support-style question and you need a command sequence.
- [references/investigations.md](references/investigations.md): use when choosing or interpreting `investigate` commands.
- [references/commands.md](references/commands.md): use when you need exact command syntax or flags.
- [references/output.md](references/output.md): use when parsing NDJSON, errors, redaction, or mutation guards.
