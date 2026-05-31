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
| `streams list --server` | `X-Postmark-Account-Token` | Listing streams for a server is account/server management. |
| `messages` | `X-Postmark-Server-Token` | Message activity belongs to one server token. |
| `bounces` | `X-Postmark-Server-Token` | Bounce activity belongs to one server token. |
| `stats delivery` | `X-Postmark-Server-Token` | Delivery stats are server-level. |
| `webhooks` | `X-Postmark-Server-Token` | Webhook configuration is exposed on server-token API in v1. |

## v1 command map

| CLI command | Method/path | Output |
| --- | --- | --- |
| `servers list` | `GET /servers?count=&offset=` | NDJSON rows from `Servers`; pagination meta from `TotalCount`. |
| `servers get <id>` | `GET /servers/{id}` | Redacted JSON. |
| `streams list --server <id>` | `GET /servers/{id}/message-streams` | NDJSON rows from `MessageStreams`. |
| `streams get <id>` | `GET /message-streams/{id}` | Redacted JSON. |
| `domains list` | `GET /domains?count=&offset=` | NDJSON rows from `Domains`. |
| `domains get <id>` | `GET /domains/{id}` | Redacted JSON. |
| `domains verify-dkim <id>` | `POST /domains/{id}/verifyDkim` | JSON result. |
| `domains verify-spf <id>` | `POST /domains/{id}/verifySPF` | JSON result. |
| `signatures list` | `GET /senders?count=&offset=` | NDJSON rows from `SenderSignatures`. |
| `signatures get <id>` | `GET /senders/{id}` | Redacted JSON. |
| `webhooks list` | `GET /webhooks` | NDJSON rows from `Webhooks`. |
| `webhooks get <id>` | `GET /webhooks/{id}` | Redacted JSON. |
| `messages search` | `GET /messages/outbound` | NDJSON rows from `Messages`. |
| `messages inbound-search` | `GET /messages/inbound` | NDJSON rows from `InboundMessages`. |
| `messages opens` | `GET /messages/outbound/opens` | NDJSON rows from `Opens`. |
| `messages clicks` | `GET /messages/outbound/clicks` | NDJSON rows from `Clicks`. |
| `messages get <id>` | `GET /messages/outbound/{id}/details` | Redacted JSON. |
| `messages inbound-get <id>` | `GET /messages/inbound/{id}/details` | Redacted JSON. |
| `messages inbound-retry <id> --yes` | `PUT /messages/inbound/{id}/retry` | Guarded mutation. |
| `messages inbound-bypass <id> --yes` | `PUT /messages/inbound/{id}/bypass` | Guarded mutation. |
| `bounces list` | `GET /bounces` | NDJSON rows from `Bounces`. |
| `bounces get <id>` | `GET /bounces/{id}` | Redacted JSON. |
| `suppressions dump` | `GET /message-streams/{stream}/suppressions/dump` | NDJSON rows from `Suppressions`. |
| `suppressions check <email>` | `GET /message-streams/{stream}/suppressions/dump?EmailAddress=...` | NDJSON rows from `Suppressions`. |
| `suppressions create <email> --yes` | `POST /message-streams/{stream}/suppressions` | Guarded mutation. |
| `suppressions delete <email> --yes` | `POST /message-streams/{stream}/suppressions/delete` | Guarded mutation. |
| `stats delivery` | `GET /deliverystats` | JSON. |
| `api get <path> --token server|account` | raw `GET` only | Redacted JSON. |

## Query parameter conventions

Postmark list endpoints generally use `count` and `offset`, often with a
`TotalCount` envelope. The CLI keeps those names so LLMs can map directly to the
docs. Message and bounce dates are currently passed through as Postmark expects:
`fromdate` and `todate`, with Postmark's documented timestamp shape.

## Redaction defaults

The CLI redacts fields likely to contain personal data or message content:

- recipient and sender fields such as `To`, `Cc`, `Bcc`, `From`, `Email`,
  `EmailAddress`, and `Recipients`
- message content such as `HtmlBody`, `TextBody`, `Body`, `Content`, `Subject`,
  `Headers`, `Attachments`, and `Metadata`
- anything with `token` or `secret` in the field name

The top-level resource includes `@redacted` when possible, so an LLM knows why a
field is unavailable rather than hallucinating from absent data.
