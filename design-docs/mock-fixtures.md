# mockpostmark fixtures

`mockpostmark` is the deterministic local Postmark-shaped server used for e2e
development. It should stay small, explicit, and tuned to CLI contracts rather
than trying to emulate every Postmark behavior.

## Run

```bash
make build-mock
./mockpostmark --routes
./mockpostmark --addr 127.0.0.1:12122
AGENT_POSTMARK_BASE_URL=http://127.0.0.1:12122 \
  AGENT_POSTMARK_ACCOUNT_TOKEN=account_mock \
  AGENT_POSTMARK_SERVER_TOKEN=server_mock \
  ./agent-postmark servers list
```

Full subprocess e2e is opt-in because some sandboxes cannot bind localhost:

```bash
AGENT_POSTMARK_RUN_SUBPROCESS_E2E=1 go test ./internal/cli -run SubprocessE2E -count=1
```

## Current fixtures

- Server `101`, named `Production`.
- Message streams `outbound` and `broadcasts`.
- Domain `501`, `example.com`, DKIM/SPF/Return-Path verified.
- Sender signature `601`, `support@example.com`, confirmed.
- Webhook `701`, delivery and bounce enabled.
- Outbound message `msg-1` to `user@example.com`.
- Inbound message `in-1`.
- Bounce `9001`, hard bounce for `user@example.com`, inactive and activatable.
- Open and click records for `msg-1`.
- Suppression for `user@example.com` on the `outbound` stream.

## Required behaviors

- Missing or invalid token returns Postmark-shaped `401` with `ErrorCode: 10`.
- Unknown route returns Postmark-shaped `404` with `ErrorCode: 12`.
- List routes use `TotalCount` plus the documented list field.
- Fixture message/bounce payloads include sensitive fields so e2e tests can
  prove redaction is working.
- Guarded mutation fixtures return deterministic receipts after the CLI has
  enforced `--yes`.
