# agent-postmark architecture

This note records the current package layout so future agents can extend the CLI
without re-learning the boundaries from scratch.

## Command layer

`cmd/agent-postmark` only calls `cli.Execute`. All CLI behavior lives in
`internal/cli`.

Command registration is intentionally split by user-facing domain:

- `root.go`: global flags, configured defaults, and hidden `auth` alias routing.
- `auth.go`, `auth_add.go`, `auth_update.go`, `auth_errors.go`: profile
  management and secret-safe setup. Tokens are collected via `--form` or direct
  local-only flags, then stored through `internal/credential`.
- `resources.go`, `resources_messages.go`, `resources_recipients.go`,
  `resources_helpers.go`: resource commands, shared read/mutation command
  factories, list output adapters, and guarded `--yes` mutation handling.
- `investigate.go`: higher-level investigation command wiring.
- `investigate_evidence.go`: evidence extraction, compaction, lookups, and
  typed value helpers used by investigations.
- `investigate_findings.go`: finding and `next_command` construction. This is
  the main LLM-facing triage policy surface and should stay covered by focused
  unit tests.
- `context.go`: profile/env/flag resolution, API client construction, and
  output helpers.
- `evidence.go`, `redact.go`, `compact.go`: NDJSON evidence records, sensitive
  field redaction, and compact list projections.

The command layer keeps Postmark secrets out of output. Mutations return
structured errors unless `--yes` is explicit.

## API client

`internal/api` is a small HTTP client with dependency injection:

- `Doer` for tests and mock server e2e.
- `Sleep` for retry tests without real delays.
- token-kind routing for account-token and server-token headers.
- structured error mapping with `fixable_by` and hints.

The client does not know about profiles, config files, redaction, or command
defaults; those stay in `internal/cli`.

## Configuration and credentials

`internal/config` stores only non-secret profile metadata such as host, default
server ID, and default message stream.

`internal/credential` stores account/server tokens under profile-specific
Keychain names on macOS, with a local index fallback on other platforms. CLI
commands should report token presence as booleans or storage names only, never
token values.

## Mock server

`cmd/mockpostmark` serves `internal/mockpostmark`. The mock is fixture-driven,
not a general Postmark clone.

Routes are declared in one route table in `server.go`; this keeps `--routes`,
dispatch, and fixture handlers aligned. Add new e2e fixtures by adding:

1. A route entry.
2. A fixture function or filtered list function.
3. A CLI e2e assertion for the behavior being protected.

## Test shape

Current coverage intentionally combines:

- API unit tests for headers and error mapping.
- CLI e2e tests against `mockpostmark` for command contracts, redaction,
  mutation guards, and investigation output.
- Focused investigation helper tests for finding severity and `next_command`
  policy.
- Optional subprocess e2e for the real binaries when localhost binding is
  available.

When adding an endpoint, prefer one mock-backed CLI e2e test plus focused unit
tests for any new triage policy. This keeps confidence high without forcing
every route through a brittle golden file.
