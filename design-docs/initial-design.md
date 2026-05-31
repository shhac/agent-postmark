# agent-postmark initial design

## Goal

Build a Postmark CLI for AI agents: read-heavy email delivery investigation,
compact structured output, secret-safe profile setup, and enough account/server
discovery to let an LLM answer "why did this email not arrive?" without seeing
API tokens or dumping message bodies into context.

The CLI should feel like the sibling projects:

- `agent-stripe`: Keychain-only secrets, form-based setup, profiles, structured
  errors, NDJSON lists, mock server e2e tests, and bundled LLM skill.
- `agent-dd`: operational triage posture, compact default output, domain command
  groups, useful hints, and `fixable_by: agent|human|retry`.
- `agent-posthog`: profiles that bundle credential plus scope metadata, where
  the metadata helps future commands choose the right organization/project/env.

## Postmark model

Postmark has two API token scopes:

- Account token: sent as `X-Postmark-Account-Token`; used for account-level
  resources such as servers, domains, sender signatures, and some stream
  management.
- Server token: sent as `X-Postmark-Server-Token`; used for one server's
  delivery activity such as messages, bounces, delivery stats, webhooks, and
  sending.

That means the local object managed by the CLI is not just "auth". It is a
profile:

```text
profile
  host: https://api.postmarkapp.com or mock/self-hosted-compatible URL
  account token: stored in Keychain, optional but needed for account resources
  server token: stored in Keychain, optional but needed for delivery resources
  default server id: non-secret account/server navigation metadata
  default message stream: outbound by default
```

The primary command is therefore `profiles`, with `auth` as a hidden
compatibility alias for sibling CLI familiarity:

```bash
agent-postmark profiles add prod --form --account-token --server-token --server 123 --stream outbound
agent-postmark profiles check prod
agent-postmark profiles update prod --server 456 --stream broadcasts --default
agent-postmark profiles list
```

## Command shape

```text
agent-postmark
├── profiles          add, update, remove, list, default, check
├── servers           list, get
├── streams           list, get
├── domains           list, get, verify-dkim, verify-spf
├── signatures        list, get
├── webhooks          list, get
├── messages          search, inbound-search, opens, clicks, get, dump, inbound-get, inbound-retry, inbound-bypass
├── bounces           list, get, dump, activate
├── suppressions      dump, check, create, delete
├── stats             delivery
├── investigate       delivery, bounce, domain-health, stream-health, webhook-health
├── api               get
├── config            show, path, set, unset
└── usage
```

Global flags:

```text
-p, --profile <alias>       profile alias; default from config or AGENT_POSTMARK_PROFILE
    --host <url>            API host override; AGENT_POSTMARK_BASE_URL wins for tests
    --account-token <tok>   direct account token escape hatch; never persisted or printed
    --server-token <tok>    direct server token escape hatch; never persisted or printed
    --server <id>           default server ID override
    --stream <id>           message stream override, default outbound
-f, --format <fmt>          json, yaml, jsonl/ndjson
-t, --timeout <ms>          request timeout
    --max-retries <n>       bounded retries for 429/5xx
-d, --debug                 JSON debug records to stderr, no secrets
    --full                  reserved for fuller payloads where compacting is added
```

Environment variables:

```text
AGENT_POSTMARK_PROFILE
AGENT_POSTMARK_BASE_URL
AGENT_POSTMARK_HOST
AGENT_POSTMARK_ACCOUNT_TOKEN
AGENT_POSTMARK_SERVER_TOKEN
AGENT_POSTMARK_SERVER_ID
AGENT_POSTMARK_MESSAGE_STREAM
POSTMARK_ACCOUNT_TOKEN
POSTMARK_SERVER_TOKEN
POSTMARK_SERVER_ID
```

Resolution order is explicit flags, `AGENT_POSTMARK_*` env, profile metadata,
Postmark-native env aliases, then built-in defaults.

## Safety contract

- The LLM should never see account or server tokens.
- `profiles add --form` and `profiles update --form` prompt in a native OS
  dialog and store tokens in Keychain.
- `--account-token` and `--server-token` are direct-use escape hatches for local
  tests and automation; they are never persisted or printed.
- Profile config and `credentials.json` contain only non-secret metadata.
- Message bodies, recipients, sender addresses, headers, attachments, metadata,
  tokens, and secrets are redacted by default.
- Mutating commands require explicit `--yes` and return a human-fixable JSON
  error without it. This includes domain verification, suppression
  create/delete, bounce activation, and inbound retry/bypass. Webhook edits,
  sending, and template changes remain outside v1.

## Output contract

Default formats:

- list/search/investigation streams: NDJSON (`jsonl`)
- single resources: JSON
- `--format yaml` allowed for human inspection
- structured errors on stderr

NDJSON list rows are data objects followed by optional meta rows:

```jsonl
{"ID":101,"Name":"Production","Color":"Blue"}
{"@pagination":{"has_more":true,"total_items":83,"next_offset":50}}
```

Errors:

```json
{"error":"Authentication failed: Bad or missing API token (ErrorCode 10)","fixable_by":"human","hint":"This endpoint uses the Postmark server token header. Check the profile with 'agent-postmark profiles check <profile>'."}
```

Classifications:

- `agent`: bad arguments, unknown format, missing discoverable server IDs, wrong
  message IDs, unsupported raw path
- `human`: auth, permissions, missing token scopes, Keychain/dialog problems
- `retry`: rate limits, transient 5xx, network errors

## Initial investigations

Investigation output uses three record types:

```jsonl
{"type":"entity","object":"bounce","id":9001,"data":{}}
{"type":"finding","severity":"critical","summary":"Recipient is inactive because of this bounce; future delivery may be suppressed.","data":{}}
{"type":"next_command","command":"agent-postmark suppressions check <email>","reason":"Check whether the bounced recipient is currently suppressed."}
```

Severities are `ok`, `info`, `warning`, and `critical`. `next_command` records
are suggestions for the LLM to consider, not implicit permission to mutate state.

`investigate delivery --email <address>` is the first high-level workflow. It
queries outbound messages and bounces for the active message stream, emits
compact redacted evidence records, and classifies likely outcomes such as no
matching activity, sent activity, bounce activity, or inactive recipient state.

Implemented investigation commands:

- `investigate bounce <bounce-id>`: explain inactive/can-activate state and tie
  the bounce to the message.
- `investigate domain-health <domain>`: inspect domain and sender signature
  DKIM/SPF/Return-Path state.
- `investigate stream-health`: combine stats, recent bounces, and webhook state
  for one stream.
- `investigate webhook-health`: list webhooks and identify missing delivery,
  bounce, inbound, or spam complaint triggers. `webhooks health` remains a
  shorter resource-oriented equivalent.

## Testing strategy

- Keep the HTTP client dependency-injected with a `Doer`, sleep function, base
  URL, and token kind so retries/error mapping are unit-testable.
- Use `mockpostmark` for deterministic e2e fixtures, including good auth,
  missing auth, invalid auth, pagination, redaction, and common delivery/bounce
  records.
- Prefer e2e smoke tests against the mock for CLI command contracts.

See `design-docs/architecture.md` for the current package layout and extension
points.
