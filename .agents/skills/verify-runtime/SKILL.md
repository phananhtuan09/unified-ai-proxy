---
name: verify-runtime
description: Run bounded runtime verification for Unified AI Proxy through its local HTTP API and CLI, with optional real-browser smoke tests only for browser-based authentication flows.
---

# Verify Runtime

Verify observable runtime behavior against the approved spec checklist.

## Project defaults

Read this skill's `tools.yaml` and `project.env` first.
They define the HTTP/CLI driver, config path, startup command, origin, healthcheck, and artifact directory.
Parse `project.env` as dotenv data; never shell-source it.
Resolve credentials only from named process environment variables and never print or persist them.

## Project runtime model

- Start with `go run . start --config config.yaml` when configured and wait for `GET /health`.
- Derive the origin from `server.host` and `server.port`; never assume port `3000`.
- Verify OpenAI `POST /v1/chat/completions` and Anthropic `POST /v1/messages` through the local authenticated API when required.
- Verify SSE by consuming `text/event-stream` frames and terminal behavior, not provider internals.
- Use CLI commands for auth/config/lifecycle cases.
- A real browser is optional and limited to Codex OAuth or Command Code `browser_key` smoke tests explicitly requiring it.
- Prefer existing Go `httptest` integration paths for deterministic evidence; do not call live upstreams unless the approved testcase explicitly requires a safe manual smoke test.

## Input and output

- Required: `docs/ai/specs/{feature-name}.md`, `docs/ai/checklists/{feature-name}.md`, and existing `docs/ai/verifications/{feature-name}.md`.
- Optional: explicit HTTP/CLI target, config, fixture, credential, or browser overrides.
- Update the verification artifact and checklist; store sanitized logs/responses/screenshots under configured artifacts.

## Workflow

1. Read all inputs, `tools.yaml`, `project.env`, and relevant architecture sections.
2. Confirm the selected driver and derive the configured origin.
3. Check `/health`; if unavailable, run only the declared startup command and wait up to its timeout.
4. Stop as `Blocked` if config, server, auth, fixture, or driver is unavailable.
5. Select only cases requiring runtime, integration, environment, streaming, CLI, or manual smoke evidence.
6. Execute the exact protocol/CLI action and expected result, including auth, request shape, stream flag, error path, provider/account setup, and cleanup.
7. Capture method, path, status, sanitized headers, response envelope, SSE event types, CLI output, config identity, and artifact pointers.
8. For browser auth smoke tests, use a clean context and record only safe observations; never capture tokens or authenticated content.
9. Update detailed evidence, statuses, counts, percentages, and fully covered ACs before returning.

## Runtime evidence rules

- `🟢` requires the exact testcase to pass through the configured HTTP/CLI driver or required browser flow.
- `🟡` means partial coverage, narrower fixture/provider/account scope, manual judgment, or environment limitation.
- `🔴` means failed, not run, blocked, contradictory, or unclear by spec.
- Do not mark a whole protocol/provider matrix green from one model, account, endpoint, or happy path.
- Confirm protocol envelopes, status codes, local auth, stream lifecycle, terminal errors, and failover only when required by the testcase.
- Never write API keys, OAuth tokens, authorization headers, passwords, decrypted backups, or raw secret-bearing responses to evidence.
- Preserve testcase IDs, wording, expected results, order, Vietnamese headings, and existing implementation evidence.

## Artifact and boundaries

Append/update: `Runtime Target`, `Runtime Testcase Evidence`, `HTTP/API Checks`, `CLI Checks`, `Browser Smoke Checks`, `Manual-Only Testcases`, `Runtime Failures / Blocks`, `Spec Gaps / Drift`, `Final Checklist Update`, and `Runtime Status`.
Use `Not automatically verifiable` only inside detailed evidence.
Do not modify code, tests, specs, or testcase definitions; do not repair failures or recreate the artifact.
Keep execution bounded to the approved checklist and safe setup/cleanup.

When run by `/orchestrator`, append exactly one final line after both artifacts are updated:

- Pass/Partial: `<!-- orchestrator: outcome=continue provides=runtime_verified,checklist_path checklist_path=docs/ai/checklists/{feature-name}.md -->`
- Fail: `<!-- orchestrator: outcome=stop-fail -->`
- Blocked: `<!-- orchestrator: outcome=stop-blocked -->`
