# agent-postmark

Postmark delivery triage CLI for AI agents.

## Project Rules

- Keep `profiles` as the primary command for local credential/profile setup.
  `auth` is a hidden compatibility alias only.
- Preserve Postmark's account-token/server-token distinction:
  - account token: servers, domains, sender signatures, account/server setup
  - server token: message streams, messages, bounces, stats, webhooks, suppressions
- Never print stored tokens. Secrets live in Keychain; config files contain only
  non-secret metadata.
- Prefer read-only commands. Any command that can change Postmark state must
  require `--yes` and return a human-fixable JSON error without it.
- Lists default to NDJSON; single resources default to JSON.
- Redact message bodies, headers, attachments, recipient data, sender addresses,
  tokens, and secrets by default.
- Keep HTTP logic dependency-injected so tests do not need real network access.
- Use `mockpostmark` fixtures for CLI contract tests.

## Verification

```bash
GOCACHE=/private/tmp/agent-postmark-go-build go test ./... -count=1
GOCACHE=/private/tmp/agent-postmark-go-build go vet ./...
AGENT_POSTMARK_RUN_SUBPROCESS_E2E=1 GOCACHE=/private/tmp/agent-postmark-go-build go test ./internal/cli -run SubprocessE2E -count=1
```
