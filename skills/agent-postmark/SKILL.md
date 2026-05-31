---
name: agent-postmark
description: |
  Triage and investigate Postmark delivery, bounces, outbound and inbound messages, sender domains, sender signatures, message streams, webhooks, and server/account configuration. Use when:
  - Explaining why an email did not arrive or bounced
  - Checking Postmark message status, bounce state, or inactive recipients
  - Inspecting sender domain, DKIM, SPF, Return-Path, or sender signature health
  - Checking Postmark webhooks or message streams
  - Looking up Postmark servers, bounces, messages, domains, signatures, or delivery stats
  Triggers: "postmark", "email delivery", "bounce", "hard bounce", "message stream", "sender signature", "DKIM", "SPF", "Return-Path", "webhook delivery", "inactive recipient"
allowed-tools: Bash(agent-postmark *) Bash(mockpostmark *) Read Grep Glob
---

# agent-postmark

Use `agent-postmark` when investigating Postmark delivery problems, bounces,
message status, sender/domain configuration, message streams, or webhooks.

## Safety

- Never ask the tool to reveal account or server tokens.
- Never accept pasted Postmark tokens in chat. Ask the user to run
  `agent-postmark profiles add <profile> --form --account-token --server-token`
  locally so tokens go directly into OS dialogs.
- Use `agent-postmark profiles update <profile> --form --account-token` or
  `--server-token` when a stored token needs replacement.
- Prefer read-only commands.
- Remember the scope split: account-token commands handle servers, domains, and
  signatures; server-token commands handle messages, bounces, stats, and
  webhooks.
- Treat message content and recipient data as sensitive. The CLI redacts these
  fields by default.

## Start Here

```bash
agent-postmark usage
agent-postmark profiles list
agent-postmark profiles check
agent-postmark config show
agent-postmark servers list
```

Prefer `investigate` commands when the user asks an incident-style question:

```bash
agent-postmark investigate delivery --email user@example.com
agent-postmark investigate bounce <bounce-id>
agent-postmark investigate domain-health example.com
agent-postmark investigate stream-health --stream outbound
agent-postmark investigate webhook-health
agent-postmark messages search --to user@example.com --count 20
agent-postmark bounces list --email user@example.com --count 20
agent-postmark suppressions check user@example.com
agent-postmark messages get <message-id>
agent-postmark bounces get <bounce-id>
```

For configuration checks:

```bash
agent-postmark servers list
agent-postmark streams list --server <server-id>
agent-postmark domains list
agent-postmark domains get <domain-id>
agent-postmark signatures list
agent-postmark webhooks list
agent-postmark webhooks health
agent-postmark stats delivery
```

Operational server-token commands:

```bash
agent-postmark messages inbound-search --from reply@example.com
agent-postmark messages opens --count 20
agent-postmark messages clicks --count 20
agent-postmark suppressions dump --stream outbound
```

Mutation commands require explicit human confirmation with `--yes`. Do not run
them unless the user asks for the state change:

```bash
agent-postmark domains verify-dkim <domain-id> --yes
agent-postmark suppressions create user@example.com --yes
agent-postmark suppressions delete user@example.com --yes
agent-postmark messages inbound-retry <message-id> --yes
```

For local testing, run `mockpostmark` and set `AGENT_POSTMARK_BASE_URL`.

## Output

Lists and investigations default to NDJSON. Single resources default to JSON.
Errors include `fixable_by` and usually a `hint`.

Sensitive fields are redacted with `"[REDACTED]"` and top-level `@redacted`
paths when possible. Do not infer redacted message body or recipient details
unless the user provides them out of band.

Investigation output uses evidence records:

```json
{"type":"entity","object":"messages_search","data":{}}
{"type":"finding","severity":"info","summary":"..."}
```

Non-secret profile/config metadata lives in XDG config. Tokens live in Keychain.
`profiles list` and `profiles check` show token presence but never token values.

## Incremental References

Load these only when you need more detail:

- [references/commands.md](references/commands.md): command map and common flags.
- [references/scenarios.md](references/scenarios.md): common support questions and command sequences.
- [references/output.md](references/output.md): NDJSON, evidence records, redaction, errors, and mutation guards.
