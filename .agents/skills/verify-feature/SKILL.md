---
name: verify-feature
description: Verify a Go implementation against an approved Unified AI Proxy spec and checklist using focused tests, HTTP handler integration tests, code inspection, and project checks without redefining expected behavior.
---

# Verify Feature

Verify implementation-level evidence for every testcase in an approved feature checklist.

## Project context

- This is a Go 1.22 local proxy.
- Public runtime surfaces are `/health`, `/v1/chat/completions`, `/v1/messages`, and CLI/TUI commands.
- Prefer tests beside the owning package and `httptest`-based server integration tests.
- Use `go test`, `go vet`, and `./scripts/check.sh` according to scope; never call real upstreams in the default test suite.
- Read `docs/dev/architecture.md` before interpreting package ownership.

## Input and output

- Required spec: `docs/ai/specs/{feature-name}.md`.
- Required checklist: `docs/ai/checklists/{feature-name}.md`.
- Optional summary: `docs/ai/summaries/{feature-name}.md`.
- Update `docs/ai/verifications/{feature-name}.md` and permitted evidence fields in the checklist.

## Source of truth

- The approved spec defines expected behavior.
- Preserve testcase IDs, wording, order, expected results, and mappings.
- Code, tests, command output, and runtime observations are evidence of current behavior only.
- Never change expected results to match implementation; record spec gaps or drift instead.

## Workflow

1. Read the spec, checklist, summary, existing verification artifact, and relevant architecture sections.
2. Map each testcase to its owning package, handler, command, test, or configuration surface.
3. Choose the smallest direct evidence: focused unit test, `httptest` integration test, CLI check, config validation, or code inspection.
4. Run relevant checks without adding production mocks or calling live providers.
5. Record exact commands, assertions, output, failures, limitations, and secrets-safe evidence.
6. Assign statuses, recalculate counts/percentages, and update the checklist before returning.

## Evidence rules

- `🟢` requires the exact testcase to be directly executed and passed with its inputs, transition, expected output, and environment.
- `🟡` means evidence is indirect, narrower, environment-limited, or needs runtime/human follow-up.
- `🔴` means failed, not run, blocked, contradictory, or unclear by spec.
- Code inspection, compilation, vet, lint, or build alone can never produce green.
- A focused Go test can produce green only when it directly covers the deterministic testcase.
- A happy path does not prove negative, boundary, auth, retry/failover, streaming, persistence, or error cases.
- Never change `[ ]` to `[x]` or erase existing `[x]`; green may remove an unchecked marker, yellow/red retain it.
- Keep checklist notes short and Vietnamese; put detailed evidence in the verification artifact.

## Artifact and boundaries

Keep/create: `Sources`, `Implementation Surfaces`, `Evidence Strategy`, `Executed Checks`, `Testcase Evidence`, `Failed`, `Coverage Gaps`, `Needs Runtime Verification`, `Spec Gaps / Drift`, `Checklist Update`, and `Final Status`.
Do not delete valid runtime evidence appended by `verify-runtime`.
Do not modify production code, tests, specs, or testcase definitions.
Never write API keys, OAuth tokens, authorization headers, passwords, decrypted backups, or secret response bodies to artifacts.

When run by `/orchestrator`, append exactly one final line after both artifacts are updated:

- Pass/Partial: `<!-- orchestrator: outcome=continue provides=verification_path verification_path=docs/ai/verifications/{feature-name}.md -->`
- Fail: `<!-- orchestrator: outcome=stop-fail -->`
- Blocked: `<!-- orchestrator: outcome=stop-blocked -->`
