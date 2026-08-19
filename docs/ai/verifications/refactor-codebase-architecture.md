## Sources
- Approved spec: docs/ai/specs/refactor-codebase-architecture.md
- Checklist: docs/ai/checklists/refactor-codebase-architecture.md
- Summary: docs/ai/summaries/refactor-codebase-architecture.md

## Implementation Surfaces
- TC-003 / AC3 -> internal/app/app.go, internal/cli/cli.go, internal/tui/runtime.go
- TC-005 / AC5 -> internal/app/app_test.go
- TC-006-TC-010 / AC6-AC10 -> internal/proxy/service.go, internal/proxy/service_test.go
- TC-011-TC-015 / AC11-AC15 -> internal/provider/provider.go and provider tests
- TC-023-TC-026 / AC23-AC26 -> internal/architecture/architecture_test.go, scripts/check.sh, CLAUDE.md

## Evidence Strategy
- Deterministic routing, composition, provider and architecture rules -> focused Go tests.
- Repository quality gate -> `./scripts/check.sh`.
- Documentation and placement -> implementation inspection plus architecture test.
- User-visible CLI/TUI behavior -> runtime/manual evidence; not claimed from compilation alone.

## Executed Checks
- `./scripts/check.sh` -> Pass: `go test ./...`, `go vet ./...`, and `go build ./...` all passed.
- `go test ./internal/proxy ./internal/app ./internal/architecture ./internal/provider ./internal/server ./internal/tui ./internal/config` -> Pass.
- `go test ./internal/server ./internal/proxy ./internal/config ./internal/tui` -> Pass, including new OpenAI parser and proxy error matrix tests.
- OpenAI Responses parser/handler is now in `internal/server/openai_responses.go`; renderer functions remain in `openai.go`, so AC16 remains partial.
- `go test ./internal/proxy ./internal/app ./internal/architecture ./internal/provider ./internal/server ./internal/tui` -> Pass.
- `go test ./...` -> Pass.
- `go vet ./...` -> Pass.
- `go build ./...` -> Pass.
- `git diff --check` -> Pass.
- `internal/proxy/service_test.go` -> Pass: direct Chat/Stream request snapshots and captured routed request assertions.
- `internal/app/app_test.go` -> Pass: missing config returns an error without constructing a server.
- `internal/architecture/architecture_test.go` -> Pass: current import graph and forbidden direct package names.
- `internal/provider` tests -> Pass, including Command Code parser/error/redaction and provider integration coverage present in the package.
- `go run . help` -> Pass: CLI usage and command list rendered.
- `go run . start --config config.yaml` with bounded startup -> Pass: server started at configured `127.0.0.1:18080`.
- `curl http://127.0.0.1:18080/health` -> Pass: HTTP 200 with `status=ok`.
- `curl http://127.0.0.1:18080/v1/models` without valid bearer -> Pass: HTTP 401 OpenAI-compatible unauthorized envelope.

## Testcase Evidence
| Testcase | Spec mapping | Detailed evidence | Result | Checklist status |
|---|---|---|---|---|
| TC-001 | AC1 | Existing server/provider tests passed, but the complete required characterization matrix was not independently enumerated before migration. | Partial | 🟡 |
| TC-002 | AC2 | Full test/vet/build gate passed; bounded CLI startup and health/auth checks passed, but CLI/TUI and every observable contract were not directly exercised. | Partial | 🟡 |
| TC-003 | AC3 | Code inspection confirms CLI `cmdStart` and TUI `Runtime.Load` call `app.Build`. | Pass | 🟢 |
| TC-004 | AC4 | `Runtime.Load` builds before lock/swap; no focused success/failure state-retention test was run. | Partial | 🟡 |
| TC-005 | AC5 | Missing-config build test passed; nil/non-nil logger behavior was not fully tested. | Partial | 🟡 |
| TC-006 | AC6 | `TestChatDoesNotMutateCallerRequest` passed for the captured request and caller snapshot. | Pass | 🟢 |
| TC-007 | AC7 | `TestStreamDoesNotMutateCallerRequest` passed and asserted internal `Stream=true`. | Pass | 🟢 |
| TC-008 | AC8 | Proxy tests captured upstream model/provider while the caller retained the alias. | Pass | 🟢 |
| TC-009 | AC9 | Full package tests passed, but the requested complete table of account/error branches was not independently covered. | Partial | 🟡 |
| TC-010 | AC10 | Stream immutability/setup error was tested; pre-channel failover and event contract were not fully tested. | Partial | 🟡 |
| TC-011 | AC11 | Package compiled with narrowed `Provider`; Gemini and Command Code dummy refresh methods were removed. | Pass | 🟢 |
| TC-012 | AC12 | Code inspection shows transport/base still contains shared HTTP and OAuth code; complete ownership split was not done. | Fail | 🔴 |
| TC-013 | AC13 | Existing provider tests passed for available Codex/Gemini/Command Code behavior. | Partial | 🟡 |
| TC-014 | AC14 | Command Code capability files required by the spec were not fully created; current implementation remains largely in `commandcode.go`. | Fail | 🔴 |
| TC-015 | AC15 | Existing Command Code tests passed, including parser and credential error coverage. | Pass | 🟢 |
| TC-016 | AC16 | OpenAI protocol split was not completed; `openai.go` still contains both protocol areas. | Fail | 🔴 |
| TC-017 | AC17 | Existing server tests passed, but the full requested compatibility matrix was not independently exercised. | Partial | 🟡 |
| TC-018 | AC18 | Existing tests passed; full invalid/auth/terminal matrix was not independently enumerated. | Partial | 🟡 |
| TC-019 | AC19 | Config behavior tests passed, but requested file split was not completed. | Partial | 🟡 |
| TC-020 | AC20 | Existing config tests passed; permission behavior was not separately executed in this verification. | Partial | 🟡 |
| TC-021 | AC21 | TUI package tests passed, but requested model/commands/forms/views split was not completed. | Partial | 🟡 |
| TC-022 | AC22 | TUI tests passed; interactive smoke check was not run in the non-interactive session. | Partial | 🟡 |
| TC-023 | AC23 | Architecture test's executable current-graph check passed; temporary forbidden-dependency failure scenario was not run. | Partial | 🟡 |
| TC-024 | AC24 | Forbidden package-name rule is implemented and current graph passes; failure fixture was not run. | Partial | 🟡 |
| TC-025 | AC25 | `go test ./internal/architecture` passed against the production import graph. | Pass | 🟢 |
| TC-026 | AC26 | `./scripts/check.sh` passed test, vet and build in fail-fast order; command is documented in `CLAUDE.md`. | Pass | 🟢 |
| TC-027 | AC27 | `docs/dev/architecture.md` documents `internal/app` and `scripts/check.sh`; provider/hotspot details still reflect incomplete migration. | Partial | 🟡 |
| TC-028 | AC28 | Quality gate and diff check passed, but known incomplete hotspot/provider ownership scope remains. | Fail | 🔴 |

## Failed
- TC-012: shared transport/OAuth ownership split is incomplete.
- TC-014: Command Code capability file split is incomplete.
- TC-016: OpenAI Chat Completions/Responses file split is incomplete.
- TC-028: full D-001 scope is not complete.

## Coverage Gaps
- Routing failover/error matrix is narrower than AC9-AC10.
- Logger parity, config permissions, and TUI load retention lack focused evidence.
- Temporary architecture violation fixtures were not executed.
- Full hotspot file split and characterization matrix remain incomplete per execution summary.

## Needs Runtime Verification
- TC-002, TC-004, TC-005, TC-021, TC-022 and all observable CLI/TUI flows need direct runtime/manual evidence.
- Configured runtime is `127.0.0.1:18080`; the prior `localhost:3000` target was stale.

## Spec Gaps / Drift
- Implementation summary explicitly records incomplete hotspot splits, while AC12, AC14, AC16, AC19 and AC21 require those splits. This is implementation drift, not a spec ambiguity.

## Checklist Update
- Green: 8/28
- Yellow: 16/28
- Red: 4/28

## Final Status
Partial

## Runtime Target
- Tool / driver: project-native HTTP/CLI driver from `.agents/skills/verify-runtime/tools.yaml`.
- Base URL / API URL / healthcheck: `http://127.0.0.1:18080` / `http://127.0.0.1:18080/v1` / `http://127.0.0.1:18080/health`.
- Startup: `go run . start --config config.yaml`, bounded local run; no live upstream call was made.

## Runtime Testcase Evidence
| Testcase | Spec mapping | E2E actions and assertions | Browser diagnostics | Artifact | Result | Checklist status |
|---|---|---|---|---|---|---|
| TC-002 | AC2 | Started with `go run . start --config config.yaml`; `GET /health` returned 200 and `GET /v1/models` without valid bearer returned 401 with sanitized unauthorized envelope. | Not applicable; project-native HTTP/CLI run. | None | Partial | 🟡 |
| TC-004, TC-005, TC-021, TC-022 | AC4, AC5, AC21, AC22 | No interactive TUI scenario executed; bounded CLI/runtime smoke does not cover TUI state and navigation. | Not applicable; no browser used. | None | Blocked/manual | 🔴 |

## Automated Runtime Checks
- `go run . help` -> pass.
- `go run . start --config config.yaml` -> server started on configured `127.0.0.1:18080`.
- `curl -fsS --max-time 5 http://127.0.0.1:18080/health` -> HTTP 200, `status=ok`.
- `curl -i -sS --max-time 5 http://127.0.0.1:18080/v1/models` -> HTTP 401; no secret-bearing value recorded.
- `curl -i -sS --max-time 5 -X POST http://127.0.0.1:18080/v1/chat/completions` without bearer -> HTTP 401 with OpenAI-compatible error envelope.
- `curl -i -sS --max-time 5 -X POST http://127.0.0.1:18080/v1/messages` without bearer -> HTTP 401 with Anthropic-compatible error envelope.

## Manual-Only Testcases
- TUI load/start-stop/navigation smoke check requires an interactive terminal.
- CLI listening/output lifecycle requires a real config and local runtime.

## Runtime Failures / Blocks
- TUI load/start-stop/navigation smoke remains manual-only and was not run in the non-interactive session.
- No browser-driver execution was needed; the approved runtime surface is a Go CLI/TUI and local HTTP API.

## Spec Gaps / Drift
- Existing verification evidence referenced `localhost:3000`, but `project.env` declares startup via `config.yaml`, whose server is `127.0.0.1:18080`; current runtime verification uses the config-derived address.

## Final Checklist Update
- Green: 8/28
- Yellow: 16/28
- Red: 4/28
- Fully covered ACs: 4/28

## Runtime Status
Partial
