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
- Treat message content and recipient/sender data as sensitive. The CLI redacts
  these fields by default.
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

Lists and investigations default to NDJSON. Single resources default to JSON.
Errors are JSON on stderr with `error`, `fixable_by`, and usually `hint`.

Sensitive fields are redacted with `"[REDACTED]"` and top-level `@redacted`
paths when possible. Do not infer redacted message body, subject, recipient, or
sender details unless the user provides them out of band.

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
