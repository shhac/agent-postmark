# Postmark API mapping

Sources:

- `https://postmarkapp.com/developer/api/overview`
- `https://postmarkapp.com/developer/api/servers-api`
- `https://postmarkapp.com/developer/api/messages-api`
- `https://postmarkapp.com/developer/api/bounce-api`
- `https://postmarkapp.com/developer/api/message-streams-api`
- `https://postmarkapp.com/developer/api/domains-api`
- `https://postmarkapp.com/developer/api/webhooks-api`

## Token headers

| CLI area | Postmark header | Why |
| --- | --- | --- |
| `servers` | `X-Postmark-Account-Token` | Server inventory is account-level. |
| `domains` | `X-Postmark-Account-Token` | Sender domains are account-level. |
| `signatures` | `X-Postmark-Account-Token` | Sender signatures are account-level. |
| `streams` | `X-Postmark-Server-Token` | Message streams are scoped to one server token. |
| `messages` | `X-Postmark-Server-Token` | Message activity belongs to one server token. |
| `bounces` | `X-Postmark-Server-Token` | Bounce activity belongs to one server token. |
| `stats delivery` | `X-Postmark-Server-Token` | Delivery stats are server-level. |
| `webhooks` | `X-Postmark-Server-Token` | Webhook configuration is exposed on server-token API in v1. |

## v1 command map

| CLI command | Method/path | Output |
| --- | --- | --- |
| `servers list` | `GET /servers?count=&offset=` | NDJSON rows from `Servers`; pagination meta from `TotalCount`. |
| `servers get <id> [id...]` | `GET /servers/{id}` per id | NDJSON by default; one record or `{"@unresolved":{…}}` per id in order. Pass `--format json` for object/envelope. |
| `streams list` | `GET /message-streams` | Uses the selected profile server token; NDJSON rows from `MessageStreams`. |
| `streams get <id> [id...]` | `GET /message-streams/{id}` per id | NDJSON by default; multi-get contract. |
| `domains list` | `GET /domains?count=&offset=` | NDJSON rows from `Domains`. |
| `domains get <id> [id...]` | `GET /domains/{id}` per id | NDJSON by default; multi-get contract. |
| `domains verify-dkim <id>` | `POST /domains/{id}/verifyDkim` | JSON result. |
| `domains verify-spf <id>` | `POST /domains/{id}/verifySPF` | JSON result. |
| `signatures list` | `GET /senders?count=&offset=` | NDJSON rows from `SenderSignatures`. |
| `signatures get <id> [id...]` | `GET /senders/{id}` per id | NDJSON by default; multi-get contract. |
| `webhooks list` | `GET /webhooks` | NDJSON rows from `Webhooks`. |
| `webhooks get <id> [id...]` | `GET /webhooks/{id}` per id | NDJSON by default; multi-get contract. |
| `messages search` | `GET /messages/outbound` | NDJSON rows from `Messages`. |
| `messages inbound-search` | `GET /messages/inbound` | NDJSON rows from `InboundMessages`. |
| `messages opens` | `GET /messages/outbound/opens` | NDJSON rows from `Opens`. |
| `messages opens --message-id <id>` | `GET /messages/outbound/opens/{id}` | NDJSON rows from `Opens`. |
| `messages clicks` | `GET /messages/outbound/clicks` | NDJSON rows from `Clicks`. |
| `messages clicks --message-id <id>` | `GET /messages/outbound/clicks/{id}` | NDJSON rows from `Clicks`. |
| `messages get <id> [id...]` | `GET /messages/outbound/{id}/details` per id | NDJSON by default; multi-get contract. Secrets redacted. |
| `messages content <id> [id...]` | `GET /messages/outbound/{id}/details` | One ID defaults to JSON; multiple IDs default to NDJSON. Bodies, headers, attachments, subject, and addressing are visible; secrets are still redacted. |
| `messages dump <id>` | `GET /messages/outbound/{id}/dump` | JSON (single; not multi-get). Secrets redacted. |
| `messages inbound-get <id> [id...]` | `GET /messages/inbound/{id}/details` per id | NDJSON by default; multi-get contract. Secrets redacted. |
| `messages inbound-retry <id> --yes` | `PUT /messages/inbound/{id}/retry` | Guarded mutation. |
| `messages inbound-bypass <id> --yes` | `PUT /messages/inbound/{id}/bypass` | Guarded mutation. |
| `bounces list` | `GET /bounces` | NDJSON rows from `Bounces`. |
| `bounces get <id> [id...]` | `GET /bounces/{id}` per id | NDJSON by default; multi-get contract. Secrets redacted. |
| `bounces dump <id>` | `GET /bounces/{id}/dump` | JSON (single; not multi-get). Secrets redacted. |
| `bounces activate <id> --yes` | `PUT /bounces/{id}/activate` | Guarded mutation. |
| `suppressions list` | `GET /message-streams/{stream}/suppressions/list` | NDJSON rows from `Suppressions`; Postmark returns `TotalCount` and honors `count`/`offset`. |
| `suppressions check <email>` | `GET /message-streams/{stream}/suppressions/list?EmailAddress=...` | NDJSON rows from `Suppressions`. |
| `suppressions create <email> --yes` | `POST /message-streams/{stream}/suppressions` | Guarded mutation. |
| `suppressions delete <email> --yes` | `POST /message-streams/{stream}/suppressions/delete` | Guarded mutation. |
| `stats delivery` | `GET /deliverystats` | JSON. |
| `investigate delivery` | `GET /messages/outbound`, `GET /bounces` | Evidence NDJSON. |
| `investigate bounce <id>` | `GET /bounces/{id}`, optional `GET /messages/outbound/{id}/details` | Evidence NDJSON. |
| `investigate domain-health <domain>` | `GET /domains` or `GET /domains/{id}`, `GET /senders` | Evidence NDJSON. |
| `investigate stream-health` | `GET /deliverystats`, `GET /bounces`, `GET /webhooks`, `GET /message-streams/{stream}/suppressions/list` | Evidence NDJSON. |
| `investigate webhook-health` | `GET /webhooks` | Evidence NDJSON. |
| `api get <path> --token server|account` | raw `GET` only | Redacted JSON. |

Suppression endpoint probe notes:

- Official docs currently document `GET /message-streams/{stream}/suppressions/dump` for reads, but live API also supports `GET /message-streams/{stream}/suppressions/list`.
- `GET /message-streams/{stream}/suppressions/list` returned `TotalCount`, honored `count`/`offset`, and supported `EmailAddress`, `SuppressionReason`, and `Origin` filters in live probes.
- `GET /message-streams/{stream}/suppressions` returned `Method Not Allowed`; that path remains mutation-only for `POST` create.
- `GET /message-streams/{stream}/suppressions/dump` is a real full-export endpoint, but agent-facing commands intentionally avoid it because it can return very large responses.
- `page=6` alone returned a missing `offset` API error; UI page navigation maps to CLI `--offset`, not an API `page` parameter.

## Query parameter conventions

Postmark list endpoints generally use `count` and `offset`, often with a
`TotalCount` envelope. The CLI keeps those names so LLMs can map directly to the
docs. Message and bounce dates are currently passed through as Postmark expects:
`fromdate` and `todate`, with Postmark's documented timestamp shape.

## Redaction and compacting defaults

The CLI separates compacting from redaction:

- list commands default to compact rows and omit bulky fields such as bodies,
  headers, and attachments unless `--full` is requested
- delivery triage fields such as `Subject`, `From`, `To`, `Cc`, `Bcc`,
  `ReplyTo`, `Email`, and `EmailAddress` are visible by default
- entity get commands (`servers/streams/domains/signatures/webhooks/bounces/messages get` and `messages inbound-get`) accept 1..N ids; default NDJSON one line per id or `{"@unresolved":{…}}` for a miss; item misses → stdout exit 0; command failures → stderr exit 1; `--format json|yaml` collapses to `{"data":[…],"@unresolved":[…]}` envelope
- single-resource commands (dump, api get) stay single-id JSON
- `messages content <id> [id...]` is the explicit command for retrieving full
  outbound email content from one or more message IDs
- fields named like tokens or secrets, URL credentials, and Postmark
  `OriginalEmail` raw email blobs are redacted by default

The top-level resource includes `@redacted` when possible, so an LLM knows why a
field is unavailable rather than hallucinating from absent data.
