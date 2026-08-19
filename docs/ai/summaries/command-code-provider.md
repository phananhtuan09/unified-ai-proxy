# Summary: Command Code Provider

## Done
- Config: `providerDefaults["command_code"]` (method `browser_key`, authorization_url `https://commandcode.ai/studio/auth/cli`, redirect_port 1458, base_url `https://api.commandcode.ai`); `validateProvider` accepts `browser_key` (requires authorization_url, redirect_port > 0, account token_file).
- Provider client `internal/provider/commandcode.go`: `/alpha/generate` request translation, NDJSON stream parser (1 MiB line limit), non-stream aggregation, credential validation, typed `UpstreamError`, credential sanitization.
- Browser login `internal/provider/commandauth.go`: `RunBrowserKeyLogin` for method `browser_key`; URL `...?callback=<redirectURI>&state=<32-byte hex>`; constant-time state comparison; apiKey `user_` prefix validation; CORS/PNA headers for the studio POST; loopback bind; non-blocking callback delivery.
- Factory registration + CLI `cmdAuth` branch by auth method.
- `UpstreamError.UnsupportedModel` + `apierr.UnsupportedModel` (HTTP 400 `unsupported_model`) + proxy `mapUpstreamError` classification.
- Stream writers: `StreamError` is now terminal in both OpenAI and Anthropic writers (no success finish / `[DONE]` / `message_stop` after error).
- Accounts summary: token-file accounts with missing/unparseable/empty token report `reauth_required`.
- Fixtures extracted from 9router source (`decolua/9router`) and Command Code CLI/OpenCode plugin into `internal/provider/testdata/commandcode/`.
- Tests: provider request/stream/error/redaction, callback handler, config defaults/validate, accounts summary, and an in-process integration test covering both `/v1/chat/completions` and `/v1/messages` (stream + non-stream).
- `config.example.yaml` with 5 model aliases per D-003.

## Not Done / Blocked
- Real-browser login is a manual smoke test only (as allowed by the spec); the browser opener is exercised via URL/callback seams, not a live browser.
- No automatic sync from `~/.commandcode/auth.json` (explicitly out of scope).

## Decisions
- `config` object: the spec's AC6 requires "config đúng fixture"; fixture A-001 (9router `openai-to-commandcode.js`) shows a full object (`workingDir`, `date`, `environment`, `structure`, `isGitRepo`, `currentBranch`, `mainBranch`, `gitStatus`, `recentCommits`), so the provider reproduces that shape rather than an empty object.
- `threadId` equals `x-session-id`: implemented per spec AC6; A-001 does not constrain the relationship (9router uses two separate UUIDs, which is equivalent for a stateless per-request UUID).
- Stop-reason mapping follows the spec: `stop` → `end_turn`, `length` → `max_tokens`, others pass through; the first terminal event (`finish-step` or `finish`) emits the single `StreamMessageStop`.
- Browser opener is a direct `openBrowser` call (matching `oauth.go`), not injected; the URL builder and callback handler are the unit-tested seams.
- `x-command-code-version` hardcoded to `0.25.7` (agent-chosen default per spec), not yet configurable.
- Unknown-model classification: structured `code` (`unsupported_model`/`model_not_found`) or a message matching a model-not-found pattern; generic 400 stays `invalid_request`.
- Accounts `reauth_required` inference is applied to all token-file accounts (not only command_code), fixing the pre-existing "missing token still shows ok" behavior called out in the spec.

## Verified
- AC1 ✅ config defaults + validate (missing authorization_url/redirect_port/token_file) — `internal/config/config_test.go`.
- AC2 ✅ 5 model aliases with correct upstreams — `config.example.yaml`; alias uniqueness is enforced by existing duplicate-alias validation.
- AC3 ✅ login URL format + browser-open-failure tolerance — `TestBuildBrowserKeyURL`; browser is manual smoke test.
- AC4 ✅ callback handler: state match/mismatch, apiKey validation, no key echo — `internal/provider/commandauth_test.go`.
- AC5 ✅ Build/Models/ValidateAccount/RefreshToken/Chat/Stream typed auth error + account summary `reauth_required` — `commandcode_test.go`, `summary_test.go`.
- AC6 ✅ request envelope, `params.stream=true`, threadId == x-session-id, headers, stop/metadata dropped — `commandcode_test.go`.
- AC7 ✅ NDJSON success/error/limit/EOF/wrong-shape terminal behavior — `commandcode_test.go` + `commandcode_integration_test.go`.
- AC8 ✅ non-stream aggregation + error discards partial — `commandcode_test.go`.
- AC9 ✅ upstream 401/403/429/5xx/400 mapping + unknown-model classification + redaction — `commandcode_test.go`, `commandcode_integration_test.go`.
- AC10 ✅ in-process integration test (both protocols, stream + non-stream, unknown alias 404, unknown-model 400, generic 400, redaction) — `internal/server/commandcode_integration_test.go`.

## Not Verified
- Real browser login end-to-end (live studio redirect) — deferred to manual `/verify-runtime`; requires a real Command Code account.

## Notes
- `go test ./...` and `go vet ./...` both pass.
- Fixtures were extracted from the public 9router repository (`decolua/9router`, MIT) and the opencode-commandcode plugin (MIT), with credentials scrubbed.
