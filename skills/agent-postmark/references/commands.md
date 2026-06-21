# agent-postmark command reference

Use this file when you need exact command names, flags, or token scope. For
support workflows, check `scenarios.md` or `investigations.md` first.

## Profiles

```bash
agent-postmark profiles setup prod --form --account-token --server app:123:outbound --server billing:456:outbound
agent-postmark profiles add prod --form --account-token
agent-postmark profiles servers add prod app --form --server-token --server-id 123 --stream outbound --default
agent-postmark profiles check prod
agent-postmark profiles servers update prod app --server-id 456 --stream broadcasts --default
agent-postmark profiles update prod --form --account-token
agent-postmark profiles servers update prod app --form --server-token
agent-postmark profiles servers list prod
agent-postmark profiles servers remove prod app
agent-postmark profiles list
agent-postmark auth list
```

`profiles` is primary. `auth` is a hidden compatibility alias.

## Account-token commands

```bash
agent-postmark servers list
agent-postmark servers get <server-id> [server-id...]
agent-postmark domains list
agent-postmark domains get <domain-id> [domain-id...]
agent-postmark domains verify-dkim <domain-id> --yes
agent-postmark domains verify-spf <domain-id> --yes
agent-postmark signatures list
agent-postmark signatures get <signature-id> [signature-id...]
```

## Server-token commands

```bash
agent-postmark --server <server-alias> streams list
agent-postmark --server <server-alias> streams get <stream-id> [stream-id...]
agent-postmark messages search --to user@example.com --stream outbound
agent-postmark messages inbound-search --from reply@example.com
agent-postmark messages get <message-id> [message-id...]
agent-postmark messages content <message-id> [message-id...]
agent-postmark messages dump <message-id>
agent-postmark messages inbound-get <message-id> [message-id...]
agent-postmark messages opens --count 20
agent-postmark messages opens --message-id <message-id>
agent-postmark messages clicks --count 20
agent-postmark messages clicks --message-id <message-id>
agent-postmark bounces list --email user@example.com
agent-postmark bounces get <bounce-id> [bounce-id...]
agent-postmark bounces dump <bounce-id>
agent-postmark suppressions list --stream outbound
agent-postmark suppressions check user@example.com
agent-postmark webhooks list
agent-postmark webhooks get <webhook-id> [webhook-id...]
agent-postmark webhooks health
agent-postmark stats delivery
```

Entity gets default to NDJSON (one line per id). A missing id yields an
`{"@unresolved":{…}}` line on stdout (exit 0); auth/network failures go to
stderr (exit 1). Pass `--format json` for a pretty object or envelope.

Use `suppressions list` for paginated reads. Do not use Postmark's raw
`/suppressions/dump` endpoint for routine agent workflows; it can return a full
stream export.

Guarded mutations:

```bash
agent-postmark suppressions create user@example.com --yes
agent-postmark suppressions delete user@example.com --yes
agent-postmark bounces activate <bounce-id> --yes
agent-postmark messages inbound-retry <message-id> --yes
agent-postmark messages inbound-bypass <message-id> --yes
```

## Investigations

```bash
agent-postmark investigate delivery --email user@example.com
agent-postmark investigate bounce <bounce-id>
agent-postmark investigate domain-health example.com
agent-postmark investigate stream-health --stream outbound
agent-postmark investigate webhook-health
```

## Raw API

```bash
agent-postmark api get /bounces --token server --query count=10
agent-postmark api get /domains --token account
```

Raw API is GET-only in v1.
