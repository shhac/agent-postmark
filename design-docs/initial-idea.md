# agent-postmark Initial Idea

## Premise

Postmark has a broad REST API for sending, delivery investigation, bounces, messages, sender signatures, domains, webhooks, suppressions, stats, inbound rules, and templates. Its existing CLI is useful, but appears focused on sending email, listing servers, and template pull/push workflows.

There is room for a read-first, agent-safe Postmark CLI focused on delivery triage and account configuration checks.

## Gap

The official Postmark CLI covers a narrow developer/CI workflow:

- Send raw or templated emails.
- List servers.
- Pull, push, and preview templates.

It does not appear to cover many operational workflows exposed by the API:

- Sender signatures.
- Domains, DKIM, SPF, and Return-Path verification.
- Webhook configuration checks.
- Message delivery investigation.
- Bounces and suppression status.
- Stats and deliverability trends.
- Inbound rules and inbound message investigation.

An AI agent investigating email delivery usually needs compact evidence, not a dashboard dump or a raw API client.

## Product Shape

`agent-postmark` should be a Postmark delivery and configuration triage CLI for AI agents.

Design principles:

- Read-first by default.
- Structured output to stdout, structured errors to stderr.
- Token-efficient list output, likely NDJSON by default.
- Secret-safe credential profiles stored outside LLM context.
- Redact message bodies, headers, recipient data, and API tokens by default.
- Explicit opt-in for mutations such as creating suppressions, retrying inbound messages, or editing webhooks.
- Classified errors with `fixable_by: agent|human|retry`.
- Investigation commands that emit evidence records plus concise findings.

## Candidate Scope

Initial command groups:

```bash
agent-postmark auth add prod --form
agent-postmark auth check prod
agent-postmark servers list
agent-postmark --server-id <server-id> streams list
agent-postmark signatures list
agent-postmark signatures get <signature-id>
agent-postmark domains list
agent-postmark domains get <domain-id>
agent-postmark domains verify-dkim <domain-id>
agent-postmark domains verify-spf <domain-id>
agent-postmark --server <server-alias> webhooks list
agent-postmark webhooks get <webhook-id>
agent-postmark messages search --to user@example.com --since 24h
agent-postmark messages get <message-id>
agent-postmark bounces list --email user@example.com --since 30d
agent-postmark suppressions check user@example.com
agent-postmark --server <server-alias> stats delivery
```

High-level investigations:

```bash
agent-postmark investigate delivery --email user@example.com --since 7d
agent-postmark investigate bounce <bounce-id>
agent-postmark investigate domain-health example.com
agent-postmark --server <server-alias> investigate webhook-health
agent-postmark investigate stream-health --stream outbound --since 7d
```

Potential mutation commands, behind explicit flags or confirmation gates:

```bash
agent-postmark suppressions create user@example.com --reason "customer requested block"
agent-postmark suppressions delete user@example.com
agent-postmark webhooks update <webhook-id> --url https://example.com/postmark
agent-postmark inbound retry <message-id>
```

## Differentiator

The tool is not a replacement for Postmark's existing CLI. It fills the operational and AI-agent gap:

- Delivery investigation over template deployment.
- Sender/domain/webhook configuration checks over email sending.
- Compact evidence records instead of raw message dumps.
- Redaction-first handling of message content and recipient data.
- Higher-level investigations for common support questions like "why did this email not arrive?"

## Open Questions

- Should v1 be strictly read-only?
- Should server-token and account-token profiles be modeled separately?
- Which message fields should be redacted by default?
- Should template workflows be omitted initially since the existing CLI already handles them?
- Should the CLI include safe SMTP/API send-test commands, or avoid sending entirely in v1?
