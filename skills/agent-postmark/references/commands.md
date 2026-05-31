# agent-postmark command reference

## Profiles

```bash
agent-postmark profiles add prod --form --account-token --server-token --server 123 --stream outbound
agent-postmark profiles check prod
agent-postmark profiles update prod --server 456 --stream broadcasts --default
agent-postmark profiles list
agent-postmark auth list
```

`profiles` is primary. `auth` is a hidden compatibility alias.

## Account-token commands

```bash
agent-postmark servers list
agent-postmark servers get <server-id>
agent-postmark streams list --server <server-id>
agent-postmark streams get <stream-id>
agent-postmark domains list
agent-postmark domains get <domain-id>
agent-postmark domains verify-dkim <domain-id> --yes
agent-postmark domains verify-spf <domain-id> --yes
agent-postmark signatures list
agent-postmark signatures get <signature-id>
```

## Server-token commands

```bash
agent-postmark messages search --to user@example.com --stream outbound
agent-postmark messages inbound-search --from reply@example.com
agent-postmark messages get <message-id>
agent-postmark messages inbound-get <message-id>
agent-postmark messages opens --count 20
agent-postmark messages clicks --count 20
agent-postmark bounces list --email user@example.com
agent-postmark bounces get <bounce-id>
agent-postmark suppressions dump --stream outbound
agent-postmark suppressions check user@example.com
agent-postmark webhooks list
agent-postmark webhooks health
agent-postmark stats delivery
```

Guarded mutations:

```bash
agent-postmark suppressions create user@example.com --yes
agent-postmark suppressions delete user@example.com --yes
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

