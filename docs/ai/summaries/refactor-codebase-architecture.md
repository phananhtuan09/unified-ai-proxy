## Done
- Added `internal/app` as the shared config, accounts, proxy and server composition owner.
- Updated CLI and TUI runtime loading to use the shared composition owner.
- Made proxy routing use a shallow request copy, preserving caller-owned model, provider and stream fields on success and error paths.
- Narrowed `provider.Provider` to catalog and chat-routing methods.
- Removed API-key provider dummy refresh methods and added an explicit OAuth refresh capability function for CLI import handling.
- Split shared provider HTTP metadata/helpers into `internal/provider/transport.go` and Codex-only OAuth token lifecycle into `internal/provider/oauth_token.go`.
- Split Command Code request models, NDJSON stream parsing, and error/redaction classification into capability-owned files.
- Split config schema, load/save I/O, defaults, and validation into capability-owned files.
- Moved shared OpenAI content decoding and finish-reason mapping into `openai_helpers.go` and added parser/normalization matrix tests.
- Added proxy error-mapping matrix coverage for invalid request, rate limit, timeout, unsupported model, plan restriction, unknown model, no account, and auth reauthentication.
- Added proxy request immutability tests and an executable architecture import/package-name check.
- Added `scripts/check.sh` and documented it as the final local gate.

## Not Done / Blocked
- OpenAI Responses parser/handler code moved to `openai_responses.go`, but Responses renderers remain in `openai.go`; TUI hotspot is still not fully split.
- Full characterization coverage for every protocol and terminal stream case was not added because existing integration coverage was retained and the broad split was deferred rather than risking behavior changes.
- TUI terminal smoke verification was not run in this non-interactive session.

## Decisions
- Kept OAuth refresh as an explicit provider capability function rather than widening the chat `Provider` interface or adding a universal auth interface.
- Kept the request copy shallow as approved by the spec; routing does not mutate slices or maps.
- Kept existing provider/config/server package paths and runtime contracts unchanged.

## Verified
- AC2: `./scripts/check.sh` passed `go test ./...`, `go vet ./...` and `go build ./...`.
- AC3: CLI and TUI use `internal/app.Build`.
- AC4: TUI swaps its graph only after `app.Build` succeeds.
- AC5: App build error test passes without starting a listener.
- AC6: Chat request immutability tests pass.
- AC7: Stream request immutability tests pass.
- AC8: Proxy tests verify the provider receives the upstream model/provider while the caller retains its alias.
- AC11: Chat provider interface no longer includes auth lifecycle methods.
- AC12: Shared transport no longer owns tokenstore/OAuth refresh; Codex composes the OAuth capability separately.
- AC14: Command Code adapter, request, stream, and error/redaction capabilities are now separate files.
- AC15: Command Code provider tests pass after the capability split.
- AC19: Config schema, I/O, defaults, and validation are now separate files in the config package; config tests pass.
- AC9: Proxy error-mapping matrix tests pass for the added deterministic branches.
- OpenAI Responses parser/handler move compiles and server tests pass; renderer split remains pending.
- AC23-AC25: Architecture package tests pass against the current production import graph.
- AC26: `scripts/check.sh` runs the required quality commands and is documented in `CLAUDE.md`.

## Not Verified
- AC1, AC10, AC13, AC16-AC18, AC20-AC22 and AC27-AC28 remain partially or fully unverified because the Responses renderer/TUI split, full protocol matrix and interactive smoke check were not completed.
