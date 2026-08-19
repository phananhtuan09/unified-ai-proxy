# Verification: Command Code Provider

## Sources
- Approved spec: docs/ai/specs/command-code-provider.md
- Checklist: docs/ai/checklists/command-code-provider.md
- Summary: docs/ai/summaries/command-code-provider.md

## Implementation Surfaces
- TC-001..004 / AC1 → `internal/config/config.go` (`providerDefaults`, `validateProvider`) + `config_test.go`
- TC-005 / AC2 → `config.example.yaml`, `config.yaml`
- TC-006..013 / AC3, AC4 → `internal/provider/commandauth.go` + `commandauth_test.go`, `internal/cli/cli.go`
- TC-014..031 / AC5..AC9 → `internal/provider/commandcode.go` + `commandcode_test.go`, `internal/provider/factory.go`, `internal/provider/provider.go`, `internal/proxy/service.go`, `internal/apierr/apierr.go`, `internal/accounts/summary.go`
- TC-032..035 / AC10 → `internal/server/commandcode_integration_test.go`, `internal/server/openai.go`, `internal/server/anthropic.go`

## Evidence Strategy
- TC-001..031 → focused automated tests (unit/integration) → deterministic non-UI rules directly exercised.
- TC-006..008, TC-012, TC-013 → partial: seams unit-tested; real browser login is a manual smoke test per spec AC3.
- TC-032..035 → in-process integration test (httptest stub) for implementation; direct runtime E2E against the live server for the running behavior.

## Executed Checks
- `go test ./...` → Pass (all packages ok)
- `go vet ./...` → Pass (no output)
- `go test -count=1 -v ./internal/provider/... ./internal/config/... ./internal/accounts/... ./internal/server/...` → Pass (all focused tests listed below)
- `go run . accounts --config config.yaml` → Pass (`command_code main ok never`)
- Live runtime: `GET /health`, `GET /v1/models`, `POST /v1/chat/completions` (stream + non-stream), `POST /v1/messages` (stream + non-stream), unknown alias → see Runtime Testcase Evidence.

## Testcase Evidence (implementation phase)

| Testcase | Spec mapping | Detailed evidence | Result | Checklist status |
|---|---|---|---|---|
| TC-001 | AC1 | `TestValidateCommandCodeDefaults` — minimal config with only `enabled:true` + account passes Validate via `providerDefaults` | Pass | 🟢 |
| TC-002 | AC1 | `TestValidateCommandCodeMissingAuthorizationURL` — error names provider `command_code` | Pass | 🟢 |
| TC-003 | AC1 | `TestValidateCommandCodeMissingRedirectPort` — error when redirect_port ≤ 0 | Pass | 🟢 |
| TC-004 | AC1 | `TestValidateCommandCodeMissingTokenFile` — error when account has no token_file | Pass | 🟢 |
| TC-005 | AC2 | Direct read of `config.example.yaml` lines 12-31: exactly 5 aliases with upstreams `deepseek/deepseek-v4-pro`, `deepseek/deepseek-v4-flash`, `Qwen/Qwen3.6-Max-Preview`, `zai-org/GLM-5`, `MiniMaxAI/MiniMax-M2.7`; aliases unique | Pass | 🟢 |
| TC-006 | AC3 | `TestBuildBrowserKeyURL` — URL base `https://commandcode.ai/studio/auth/cli`, query `callback` + `state` present; `RunBrowserKeyLogin` uses `randomHex(32)` for state | Pass | 🟢 |
| TC-007 | AC3 | Code inspection `commandauth.go:81-83`: openBrowser error prints "Could not open browser automatically" and continues to wait. No injected opener test; browser is manual smoke per spec. | Partial | 🟡 |
| TC-008 | AC4 | `TestBrowserKeyCallbackValid` verifies key delivery + no key echo in HTML. Full `RunBrowserKeyLogin` (browser → Save → Load) not executed in CI; tokenstore `TestSaveRoundTrip` covers persistence separately. | Partial | 🟡 |
| TC-009 | AC4 | `TestBrowserKeyCallbackStateMismatch` — mismatch branch returns 400 + error, no key delivered; missing state resolves to empty → same mismatch branch | Pass | 🟢 |
| TC-010 | AC4 | `TestBrowserKeyCallbackStateMismatch` — wrong state → 400, no key delivered | Pass | 🟢 |
| TC-011 | AC4 | `TestBrowserKeyCallbackInvalidKey` (non-`user_` prefix) + `TestBrowserKeyCallbackMissingKey` — 400, no key delivered | Pass | 🟢 |
| TC-012 | AC4 | Code inspection `commandauth.go:188-193` `send`/`fail` use non-blocking send; no explicit repeated-callback test | Partial | 🟡 |
| TC-013 | AC4 | Code inspection `commandauth.go:104-110` timeout (5m) and ctx.Done return error; no unit test for the full wait flow | Partial | 🟡 |
| TC-014 | AC5 | `factory.go:17` registers `command_code`; `var _ Provider = (*CommandCode)(nil)`; live `GET /v1/models` returned the 5 aliases via `svc.Models()` | Pass | 🟢 |
| TC-015 | AC5 | `TestCommandCodeCredentialErrors/missing_file` — typed auth error (401 AuthFailed) | Pass | 🟢 |
| TC-016 | AC5 | `TestCommandCodeCredentialErrors/parse_error` | Pass | 🟢 |
| TC-017 | AC5 | `TestCommandCodeCredentialErrors/empty_access_token` + `wrong_prefix` | Pass | 🟢 |
| TC-018 | AC5 | `TestSummarizeReauthWhen*` + `TestSummarizeValidToken*`; live `accounts` CLI shows `command_code main ok never` | Pass | 🟢 |
| TC-019 | AC6 | `TestCommandCodeBuildRequest` asserts `params.stream=true` | Pass | 🟢 |
| TC-020 | AC6 | `TestCommandCodeBuildRequest` asserts threadId == session id and headers (`Authorization`, `x-session-id`, `x-command-code-version`, `x-cli-environment`, `Accept`) | Pass | 🟢 |
| TC-021 | AC6 | `TestCommandCodeBuildRequestSkipsSystemMessages` — system/developer merged into `params.system`, messages are text blocks | Pass | 🟢 |
| TC-022 | AC6 | `TestCommandCodeBuildRequestDropsStopAndMetadata` — stop/metadata dropped upstream; downstream accepts them | Pass | 🟢 |
| TC-023 | AC7 | `TestCommandCodeStreamSuccess` — Start → Delta(s) → Stop with usage | Pass | 🟢 |
| TC-024 | AC7 | `TestCommandCodeStreamErrorEvent` — error → single StreamError, no stop | Pass | 🟢 |
| TC-025 | AC7 | `TestCommandCodeStreamEOFBeforeTerminal`, `TestCommandCodeStreamLineTooLong`, `TestCommandCodeStreamKnownEventWrongShape` | Pass | 🟢 |
| TC-026 | AC7 | `TestCommandCodeStreamIgnoresGarbageAndUnknown` — empty/garbage/unknown skipped | Pass | 🟢 |
| TC-027 | AC8 | `TestCommandCodeChatCompletion` — aggregates content/stop/usage | Pass | 🟢 |
| TC-028 | AC8 | `TestCommandCodeChatCompletionErrorDiscardsPartial` | Pass | 🟢 |
| TC-029 | AC9 | `TestCommandCodeUpstreamErrorMapping` — 401/403 AuthFailed non-retryable, 429/5xx retryable | Pass | 🟢 |
| TC-030 | AC9 | `TestCommandCodeUpstreamErrorMapping` + `TestCCIntegrationUnknownModelUpstream` (unsupported_model) + `TestCCIntegrationGenericBadRequest` (invalid_request) | Pass | 🟢 |
| TC-031 | AC9 | `TestCommandCodeRedaction`, `TestCommandCodeStreamRedaction`, `TestCCIntegrationAuthErrorRedacted`, `TestCCIntegrationRedactionInStream` | Pass | 🟢 |
| TC-032 | AC10 | `TestCCIntegration*` all pass with httptest stub. See Runtime evidence — live upstream FAILS. | Partial | 🔴 (downgraded by runtime) |
| TC-033 | AC10 | `TestCCIntegrationUnknownModelAlias` (404) + live runtime 404 `model_not_found` | Pass | 🟢 |
| TC-034 | AC10 | `TestCCIntegrationUnknownModelUpstream` + `TestCCIntegrationGenericBadRequest` | Pass | 🟢 |
| TC-035 | AC10 | `TestCCIntegrationOpenAIStreamErrorTerminal` + `TestCCIntegrationAnthropicStreamErrorTerminal` | Pass | 🟢 |

## Runtime Target
- Tool / driver: curl against the live running server (API proxy; no browser UI).
- Base URL: http://127.0.0.1:18474 (server already running via `go run . start --config config.yaml`).
- Auth: local proxy API key `sk-local-proxy` via `Authorization: Bearer`.
- Account: `command_code/main` token file `~/.config/unified-ai-proxy/command_code.json` — valid `user_...` apiKey, `token_type: Bearer`, zero `expires_at`.

## Runtime Testcase Evidence

| Testcase | Spec mapping | E2E actions and assertions | Browser diagnostics | Artifact | Result | Checklist status |
|---|---|---|---|---|---|---|
| TC-014 / models | AC5 | `GET /v1/models` → 5 cc-* aliases, owned_by `command_code` | 200 | — | Pass | 🟢 |
| TC-033 | AC10 | `POST /v1/chat/completions` model `no-such-model` → 404 `model_not_found` | 404 | — | Pass | 🟢 |
| TC-032 chat non-stream | AC10 | `POST /v1/chat/completions` model `cc-deepseek-v4-flash` → 200, `choices[0].message.content="hello"`, finish_reason "stop", usage present | 200 | — | Pass | 🟢 |
| TC-032 chat stream | AC10 | `POST /v1/chat/completions` stream=true → SSE `chat.completion.chunk` deltas (`Hello`, `!`, …), finish_reason "stop", `[DONE]` | 200 | — | Pass | 🟢 |
| TC-032 messages non-stream | AC10 | `POST /v1/messages` correct content-block body → 200, `content[0].text`, `stop_reason="end_turn"`, usage | 200 | — | Pass | 🟢 |
| TC-032 messages stream | AC10 | `POST /v1/messages` stream=true → SSE `message_start` → `content_block_delta`* → `content_block_stop` → `message_delta` (`stop_reason="end_turn"`) → `message_stop` | 200 | — | Pass | 🟢 |
| TC-018 | AC5 | `go run . accounts --config config.yaml` → `command_code main ok never` | — | — | Pass | 🟢 |

## Automated Runtime Checks
- `curl GET /health` → `{"status":"ok","version":"0.1.0"}` → Pass.
- `curl GET /v1/models` → 5 aliases → Pass.
- `curl POST /v1/chat/completions` (non-stream) → 200 `"hello"` + usage → Pass.
- `curl POST /v1/chat/completions` (stream) → SSE content deltas + `[DONE]` → Pass.
- `curl POST /v1/messages` (non-stream) → 200 content + `stop_reason=end_turn` + usage → Pass.
- `curl POST /v1/messages` (stream) → full Anthropic SSE event sequence → Pass.
- `curl POST /v1/chat/completions` unknown alias → 404 `model_not_found` → Pass.
- `go run . accounts` → `command_code main ok never` → Pass.

## Manual-Only Testcases
- TC-007, TC-008, TC-012, TC-013: real browser login (studio/auth/cli redirect) requires a live Command Code account and cannot be automated in this phase; spec marks browser login as a manual smoke test.

## Runtime Failures / Blocks
- Không còn. Bug `Invalid UUID at "threadId"` đã được sửa và xác minh lại.

## Fix Applied
- Root cause: `internal/provider/commandcode.go:218` used `randomHex(16)`, producing a 32-hex-char string without dashes, which the live `/alpha/generate` endpoint rejects (`Invalid UUID at "threadId"`).
- Fix: added `randomUUID()` in `internal/provider/oauth.go` (RFC 4122 v4, canonical `8-4-4-4-12` shape) and switched `StreamChatCompletion` to use it for `threadId`/`x-session-id`.
- Fixture: updated `alpha_generate_request.json` `threadId` to a dashed UUID to stay faithful to the live upstream.
- Regression guard: added `TestCommandCodeSessionIDIsUUID` in `commandcode_test.go`.
- `go test ./...` and `go vet ./...` pass; server restarted and all live endpoints re-verified green.

## Spec Gaps / Drift
- Đã giải quyết: giả định A-001 (định dạng `threadId`) được hiệu chỉnh theo validation UUID thật của upstream (canonical dashed UUID). Không còn drift đang mở.

## Final Checklist Update
- Green: 31/35
- Yellow: 4/35
- Red: 0/35
- Fully covered ACs: 8/10 (AC3, AC4 chưa đủ vì browser login là smoke test thủ công)

## Final Status
Pass (verify-feature và verify-runtime đều đạt; bug threadId-UUID đã được sửa và xác minh lại qua runtime; 4 testcase còn lại thuộc phạm vi browser login thủ công).
