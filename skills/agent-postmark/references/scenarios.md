# agent-postmark scenarios

Use this file when the user asks a support-style question. For exact command
syntax, use `commands.md`; for evidence interpretation, use `investigations.md`
or `output.md`.

## Did this email send?

```bash
agent-postmark investigate delivery --email user@example.com
agent-postmark messages search --to user@example.com --stream outbound
agent-postmark messages get <message-id>
agent-postmark messages content <message-id>
```

Look for `Status`, `ReceivedAt`, message stream, subject, addressing, and any
bounce findings. Use `messages content` only when the user needs body, header,
or attachment details for a specific send.

## Did it bounce?

```bash
agent-postmark bounces list --email user@example.com
agent-postmark investigate bounce <bounce-id>
agent-postmark bounces dump <bounce-id>
agent-postmark suppressions check user@example.com
```

Critical signals: `Inactive:true`, hard bounce type, and suppression rows.

## Is this recipient suppressed or inactive?

```bash
agent-postmark suppressions check user@example.com
agent-postmark bounces list --email user@example.com --inactive true
```

Do not remove suppressions unless the user explicitly asks and understands the
delivery implications.

Do not activate a bounce unless the user explicitly asks and understands that it
may allow future delivery to that recipient.

## Is domain authentication broken?

```bash
agent-postmark investigate domain-health example.com
agent-postmark domains get <domain-id>
agent-postmark signatures list
```

If DNS has just been fixed, ask before running verification commands with
`--yes`.

## Are webhooks configured for delivery investigation?

```bash
agent-postmark investigate webhook-health
agent-postmark webhooks list
```

At minimum, delivery and bounce triggers should usually be present for delivery
support workflows.

## Is this in the wrong stream?

```bash
agent-postmark --server <server-alias> streams list
agent-postmark investigate stream-health --stream outbound
agent-postmark messages search --to user@example.com --stream broadcasts
```

Use `profiles servers update <profile> <server> --stream <stream>` only when the
user wants the stored server-context default changed. Use `--server <alias>` to
switch among stored server contexts for a single command.
