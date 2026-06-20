# agent-postmark

Postmark delivery triage CLI for AI agents. It is designed for read-heavy
investigation workflows where an LLM needs compact structured output, useful
error hints, and no access to Postmark account or server tokens.

## Features

- Keychain-first profiles: account and server tokens are never printed back.
- Postmark-aware scope model: profiles can hold an optional account token and
  multiple server-token contexts, each with its own default stream.
- `profiles` primary command with hidden `auth` alias for sibling CLI familiarity.
- LLM-shaped output: lists default to NDJSON, single resources default to JSON.
- Structured errors: stderr JSON includes `fixable_by: agent|human|retry`.
- Delivery metadata is visible by default: subject and addressing fields are
  available for triage, while list output omits bulky bodies, headers, and
  attachments.
- Redaction: tokens, secrets, URL credentials, and original raw email blobs are
  redacted by default.
- Mock server: `mockpostmark` provides deterministic e2e fixtures.
- Agent onboarding: ships with `skills/agent-postmark/SKILL.md`.

## Quick Start

```bash
make build
./agent-postmark profiles setup prod --form --account-token \
  --server app:123:outbound \
  --server billing:456:outbound
./agent-postmark profiles add prod --form --account-token
./agent-postmark profiles servers add prod app --form --server-token --server-id 123 --stream outbound --default
./agent-postmark profiles check prod
./agent-postmark servers list
./agent-postmark --server app streams list
./agent-postmark messages search --to user@example.com --count 20
./agent-postmark bounces list --email user@example.com
./agent-postmark suppressions check user@example.com
./agent-postmark webhooks health
./agent-postmark investigate delivery --email user@example.com
./agent-postmark investigate bounce 9001
./agent-postmark investigate domain-health example.com
./agent-postmark investigate stream-health --stream outbound
```

When an LLM is guiding setup, prefer `--form`. A native OS dialog asks the user
for account or server tokens, and the CLI returns only a redacted receipt.

`auth` is a hidden compatibility alias:

```bash
agent-postmark auth list
```

## Token Scopes

Postmark account tokens are used for account-level resources like servers,
domains, and sender signatures. Server tokens are used for one server's scoped
resources, including message streams, messages, bounces, delivery stats,
webhooks, and suppressions. They are independent: a profile can have only an
account token, only server tokens, or both.

Rotate tokens without changing profile metadata:

```bash
agent-postmark profiles update prod --form --account-token
agent-postmark profiles servers update prod app --form --server-token
```

Remove a server context and its stored token:

```bash
agent-postmark profiles servers remove prod old-server
```

## Operational Commands

```bash
agent-postmark messages inbound-search --from reply@example.com
agent-postmark messages content <message-id> [message-id...]
agent-postmark messages dump <message-id>
agent-postmark messages opens --count 20
agent-postmark messages opens --message-id <message-id>
agent-postmark messages clicks --count 20
agent-postmark messages clicks --message-id <message-id>
agent-postmark bounces dump <bounce-id>
agent-postmark suppressions list --stream outbound
agent-postmark suppressions check user@example.com
agent-postmark webhooks health
agent-postmark investigate webhook-health
```

Use `suppressions list` for paginated suppression reads. Postmark also has a
`/suppressions/dump` API endpoint, but the CLI intentionally does not expose it
as a command because it can return very large full exports.

State-changing commands require `--yes`:

```bash
agent-postmark domains verify-dkim 501 --yes
agent-postmark domains verify-spf 501 --yes
agent-postmark bounces activate 9001 --yes
agent-postmark suppressions create user@example.com --yes
agent-postmark suppressions delete user@example.com --yes
agent-postmark messages inbound-retry in-1 --yes
agent-postmark messages inbound-bypass in-1 --yes
```

## Development

```bash
make test
make vet
make build
make build-mock
make mock
make mock-dev ARGS="messages search --to user@example.com"
AGENT_POSTMARK_RUN_SUBPROCESS_E2E=1 go test ./internal/cli -run SubprocessE2E -count=1
```

## Mock Postmark

```bash
make build-mock
./mockpostmark --routes
./mockpostmark --addr 127.0.0.1:12122
AGENT_POSTMARK_BASE_URL=http://127.0.0.1:12122 \
  AGENT_POSTMARK_ACCOUNT_TOKEN=account_mock \
  AGENT_POSTMARK_SERVER_TOKEN=server_mock \
  ./agent-postmark servers list
```

## License

MIT
