# agent-postmark investigations

Use investigations when the user describes a delivery/support problem rather
than asking for a specific Postmark object.

## Commands

| Command | Use for | Key records |
| --- | --- | --- |
| `investigate delivery --email <email>` | "Did this email send?", "Why did they not receive it?", wrong stream suspicion | `messages_search`, `bounces_search`, findings, next commands |
| `investigate bounce <bounce-id>` | Explain a bounce, inactive recipient, or activation risk | `bounce`, optional `message`, findings |
| `investigate domain-health <domain-id-or-name>` | DKIM/SPF/Return-Path and sender signature health | `domain`, `sender_signatures`, findings |
| `investigate stream-health --stream <stream>` | Stream-level bounces, suppressions, stats, webhook coverage | `delivery_stats`, `recent_bounces`, `webhooks`, `suppressions` |
| `investigate webhook-health` | Whether webhooks cover delivery/bounce/inbound signals | `webhooks`, findings |

## Record Semantics

- `entity`: compact evidence from Postmark, with secrets redacted.
- `finding`: CLI interpretation. Severities are `ok`, `info`, `warning`, and
  `critical`.
- `next_command`: suggested follow-up; it is not permission to mutate state.

## Interpretation Tips

- A `critical` delivery or bounce finding usually means delivery may be blocked
  by inactive/suppressed recipient state.
- A `warning` with no messages found often means the stream, date window, or
  recipient filter needs adjustment.
- Domain health warnings usually need DNS inspection outside the CLI before
  running guarded verification commands.
- Webhook health warnings tell you observability may be incomplete; they do not
  prove email delivery failed.
