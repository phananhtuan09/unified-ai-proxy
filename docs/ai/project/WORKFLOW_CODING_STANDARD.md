---
phase: project
title: Workflow Coding Standard
description: Standard coding workflow for spec creation, approval, execution, and verification in this repository
---

# Workflow Coding Standard

## Purpose
This document defines the standard coding workflow that AI agents should follow when handling work in this repository.

It covers:
- task routing
- spec creation and approval
- required delivery phases
- human-controlled step execution
- expected artifacts
- optional human-triggered support steps
- when to use the standard flow versus shorter execution paths

The workflow is designed to prevent four common failures:
- the agent makes unconfirmed assumptions
- the spec does not match the real codebase
- large features enter implementation without being sliced first
- the agent claims feature completion without implementation and runtime evidence

## Core Rule

`design-spec` owns the high-level human review and approval loop.
The approved decision manifest records that authority without becoming an implementation specification.
After `review-spec` passes, the detailed Markdown spec is the durable source of truth for execution and verification.
The standard automated workflow must not require the human to review the full detailed spec unless they explicitly choose to do so.
The workflow must not rewrite the approved spec automatically after implementation.

`sync-spec` and `review-pr` remain available as human-triggered skills outside the automated workflow.
`manual-checklist` runs automatically after execution and becomes the primary human validation artifact.

## Routing

| Task type | Standard workflow |
|---|---|
| New feature | `/design-spec` → `/spec` → `review-spec` → `/execute-spec` → `/manual-checklist` → `/verify-feature` → `/verify-runtime` |
| Fix bug (user-visible or business-impacting, requiring durable design decisions) | `/design-spec` → `/spec` → `review-spec` → `/execute-spec` → `/manual-checklist` → `/verify-feature` → `/verify-runtime` |
| Refactor | `/execute-task "Refactor: ..."` |
| Small update (1-2 files) | `/execute-task "..."` |

Rule:
- the human decides which step to run
- the agent does not choose a mode
- `/design-spec` owns material human decisions for the standard feature workflow
- `/spec` owns the detailed codebase inspection and technical design needed to produce an executable spec

### Orchestrator-Driven Execution

This repository also allows a limited orchestrator-driven execution mode.

Rules:
- this mode exists only when the human explicitly invokes `/orchestrator` with a workflow config
- the default workflow remains human-step-driven; orchestrator mode is an explicit carve-out, not the new default
- by invoking `/orchestrator`, the human is choosing a predeclared sequence of steps and consented auto-run boundaries ahead of time
- orchestrator may auto-run only steps marked `auto: true` in the selected workflow config
- orchestrator must stop at `human_gate: true`, any configured `stop_on_outcome`, missing required contracts, or unknown skill outcome
- an interactive skill may keep a foreground human review loop open and must not emit `continue` before that internal approval contract is satisfied
- an interactive artifact created before approval does not satisfy the step's declared `provides`
- orchestrator may enforce a config-declared repo lock; if another run holds that lock for the next step, the current run must stop and report the owner instead of advancing
- orchestrator must not insert or run steps that are absent from the selected config
- orchestrator must treat workflow contracts and recorded evidence as the source of truth; it must not infer missing artifact paths or outcomes heuristically
- a step may declare `cwd_from` to run against an isolated implementation workspace recorded by an earlier step; orchestrator must fail closed when that directory cannot be resolved
- a bounded implementation step may return `stop-budget`; this pauses on the same step and allows an explicit continuation with a new incremental budget instead of converting useful partial progress into a terminal failure
- long-running implementation should use bounded epochs plus a cumulative feature budget; productive epochs may continue automatically while total limits remain
- `stop-total-budget` and `stop-no-progress` must pause on the same implementation step so a human can change compatible limits or runtime policy without losing the preserved workspace; changing approved scope, agent, or worker prompt requires a new run
- resumable step inputs must be persisted and reused; the human may override the current step with explicit `--input <key>=<value>` values when continuing

## Standard Flow

```text
Human intent
  ↓
/design-spec
  ↓
HUMAN DESIGN APPROVAL
  ↓
/spec
  ↓
review-spec (AI gate)
  ↓
/execute-spec
  ↓
/manual-checklist
  ↓
/verify-feature
  ↓
/verify-runtime
```

## Design And Spec Guardrails

`/design-spec` must expose only material high-level decisions to the human.
It must not ask the human to choose file paths, function boundaries, schema field names, test commands, or implementation order unless those details carry a business or operational tradeoff.
If core behavior is unclear, the skill must ask focused questions instead of guessing.
If the request is too broad, infeasible, or conflicts with codebase or business rules, the skill must stop instead of forcing approval.

`/spec` must consume the approved decisions and inspect the codebase deeply enough to create an implementation-ready contract.
It may add grounded technical detail but must not change approved behavior.
It must stop when implementation discovery conflicts with the approved decisions.

## Delivery Phases

### `/design-spec`

Purpose:

- create a concise interactive HTML review for material high-level decisions
- collect explicit human approval through the bundled local runner
- persist approval provenance before detailed spec creation

Rules:

- write `docs/ai/designs/{feature}.html`
- write `docs/ai/design-decisions/{feature}.json` only after explicit approval
- keep the HTML local-only and self-contained
- use stable `D-xxx` identifiers for required decisions
- show the recommendation and concrete tradeoffs before alternatives
- distinguish confirmed facts, assumptions, risks, and human choices
- start the local runner and stop while the human reviews asynchronously
- do not emit `continue` until the HTML and validated decision manifest both exist
- stop without approval when no valid decision manifest exists

The HTML is the human review surface.
The decision manifest is approval provenance for spec creation and review.
Neither artifact is the implementation source of truth.

### `/spec`
Purpose:
- create the detailed AI-facing source of truth for the approved slice

Rules:
- require and validate `design_decisions_path` under `feature-standard`
- allow standalone use when material decisions have been confirmed directly in chat
- inspect relevant codebase context deeply enough to map concrete implementation surfaces
- preserve every approved decision without changing its meaning
- label agent-chosen technical details separately from human decisions
- do not write one spec for a whole epic; spec only the next executable slice
- keep the spec aligned with the codebase context discovered during local inspection
- include concrete file paths, symbols, state and data changes, interfaces, validation, failure behavior, security, compatibility, migration, implementation sequence, and verification mapping when relevant
- do not invent low-level detail when codebase evidence is insufficient
- classify the feature as `Lite`, `Standard`, or `Extended`
- write the final tier explicitly in the spec
- write generated spec prose, headings, table headers, and recurring labels in Vietnamese
- keep only code symbols, file paths, API names, schema names, JSON keys, decision IDs, AC IDs, literal enum values, and command names in English
- use tier to control analysis depth, not document length or acceptance-criteria count
- prefer information density and traceability over repetition
- write to `docs/ai/specs/{feature}.md`
- split only by independently valuable outcome or dependency boundary
- do not leave blocking product questions in a spec that is ready for execution

Expected output:
- `## Cấp Độ`
- `## Hợp Đồng Thực Thi`
- `## Vấn Đề`
- `## Phạm Vi`
- `## Ngoài Phạm Vi`
- `## Quyết Định Thiết Kế Đã Duyệt`
- `## Kiểm Tra Giả Định`
- `## Bằng Chứng Hệ Thống Hiện Tại`
- `## Yêu Cầu Hành Vi`
- `## Thay Đổi Trạng Thái / Dữ Liệu / Giao Diện`
- `## Thiết Kế Kỹ Thuật Chi Tiết`
- `## Bản Đồ Thay Đổi Theo File`
- `## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên`
- `## Cân Nhắc Bảo Mật / Phân Quyền`
- `## Tương Thích / Di Chuyển Dữ Liệu`
- `## Trình Tự Triển Khai`
- `## Tiêu Chí Chấp Nhận`
- `## Ma Trận Xác Minh`
- `## Câu Hỏi Mở`
- `## Nhật Ký Quyết Định`

Assumption rules:
- `## Kiểm Tra Giả Định` should separate:
  - confirmed
  - inferred but safe
  - needs confirmation
  - agent-chosen technical details
- blocking product uncertainty requires `ask-human` instead of `write-spec`
- agent-chosen technical details must not alter approved user-visible behavior

### `review-spec`

Purpose:

- act as the automatic AI quality gate before execution
- verify approval provenance and decision traceability
- verify codebase accuracy, implementation completeness, and verification readiness

Rules:

- validate the design decision manifest and referenced HTML checksum
- ensure every approved `D-xxx` decision is represented without semantic drift
- fail invented product behavior or falsely attributed human decisions
- verify important codebase evidence and planned implementation surfaces
- check state, data, interfaces, validation, failure behavior, security, compatibility, migration, implementation sequence, acceptance criteria, and verification mapping when relevant
- do not enforce line-count or acceptance-criteria-count limits
- do not modify the spec or approval artifacts
- emit `spec_reviewed` only for `pass` or non-blocking `warn`

The standard workflow does not require a second human gate after this review.
The human may still inspect the detailed spec explicitly when the feature or organization requires technical approval.

### `/execute-spec`
Purpose:
- implement directly from the spec
- allow iterative code changes without promoting temporary planning artifacts into durable workflow state

Rules:
- read the source spec before making changes
- treat the spec as the source of truth, not chat history or temporary execution notes
- make code changes in focused iterations
- do not introduce new product behavior, validation rules, persistence rules, fallback behavior, or visible UX behavior that is not already in the spec or explicitly approved
- if a required decision is missing, stop and ask instead of guessing
- if execution reveals that the spec is too broad or not feasible as written, stop and return the issue to the human before expanding scope
- user feedback may trigger additional edit turns
- execution may use temporary internal task breakdown, but it should not create durable plan artifacts by default
- do not invent thresholds, scoring weights, ranking formulas, fairness rules, or tie-breakers in code unless they already exist in the spec or are explicitly recorded as agent-chosen slice constraints
- if execution requires choosing such a rule to complete the slice safely, stop and ask the human instead of leaving it implicit in code
- if the spec says the logic should be transparent, visible, simple, or non-hidden, do not implement it as an opaque weighted scoring formula or hidden heuristic
- in those cases, prefer ordered readable rules such as explicit priority checks, named tie-breakers, or directly rendered reasons that the user can inspect from the UI

Execution expectations:
- make minimal, scoped changes
- follow project code conventions and structure rules
- validate changed behavior where practical
- record important implementation decisions in the execution summary without modifying the approved spec
- write a concise execution summary to `docs/ai/summaries/{feature}.md`

Required summary sections:
- `## Done`
- `## Not Done / Blocked`
- `## Decisions`
- `## Verified`
- `## Not Verified`

Summary rules:
- `## Verified` may include only checks that were actually executed during implementation
- `## Not Verified` must contain pending checks, manual-only checks, or checks intentionally deferred to verification
- do not mark an acceptance criterion as verified in both sections
- if later verification changes the confidence level, the verification artifact is the source of truth
- the summary is an execution handoff artifact, not the final verification artifact

### `/sync-spec` (Human-Triggered)
Purpose:
- reconcile the durable spec with the implemented codebase state
- keep the spec useful for future updates after iterative implementation turns
- run only when the human explicitly requests reconciliation

Rules:
- read the current spec and inspect the relevant implemented code paths
- update technical sections of the spec to match the current implementation
- record important architecture, pattern, and constraint decisions discovered during implementation
- do not silently rewrite business intent just because the code drifted
- if business rules, acceptance criteria, scope, or visible behavior need to change, propose a spec delta and require human confirmation before applying it
- write the updated spec back to the same path in `docs/ai/specs/`
- never auto-chain this skill from `/execute-spec`, orchestrator, or verification

Technical sections the invoked skill may update without extra confirmation:
- `## Bằng Chứng Hệ Thống Hiện Tại`
- `## Thay Đổi Trạng Thái / Dữ Liệu / Giao Diện`
- `## Thiết Kế Kỹ Thuật Chi Tiết`
- `## Bản Đồ Thay Đổi Theo File`
- `## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên`
- `## Cân Nhắc Bảo Mật / Phân Quyền`
- `## Tương Thích / Di Chuyển Dữ Liệu`
- `## Trình Tự Triển Khai`
- `## Ma Trận Xác Minh`
- `## Nhật Ký Quyết Định`
- implementation constraints and technical clarifications

Human confirmation required for:
- `## Hợp Đồng Thực Thi`
- `## Vấn Đề`
- `## Phạm Vi`
- `## Tiêu Chí Chấp Nhận`
- `## Quyết Định Thiết Kế Đã Duyệt`
- `## Yêu Cầu Hành Vi`
- `## Ngoài Phạm Vi`
- transparent-vs-hidden recommendation logic or any user-visible decision rule that changed in meaning

Sync rule for transparent logic:
- if the spec says recommendation logic should be visible, transparent, or non-hidden, and the implementation instead uses hidden weighted scoring or opaque heuristics, treat that as business-level drift, not a technical-only sync

### `/verify-feature`
Purpose:
- verify implementation evidence for the spec-derived checklist testcases
- write detailed evidence to the verification artifact and concise status to the checklist

Rules:
- read the approved spec, execution summary when present, and `docs/ai/checklists/{feature}.md`
- treat the approved spec as the only source of truth for expected behavior
- preserve checklist testcase definitions, expected results, IDs, order, and spec mappings
- map checklist testcases to implementation surfaces before judging coverage
- for each testcase, record the smallest suitable evidence strategy:
  - observable user behavior normally goes to `/verify-runtime` for bounded runtime/E2E checks
  - UX quality or business judgment normally goes to human verification surfaced by `/manual-checklist`
  - implementation checks such as lint, typecheck, build, and focused inspection apply when relevant
  - focused unit or integration tests are used only when they have clear regression value: non-trivial validation or business rules, permission/authorization, persistence or state transitions, integration boundaries, or a regression bug
- do not add fixed `unit-test` or `integration-test` workflow phases, and do not require test infrastructure merely to satisfy the workflow
- run only the relevant implementation checks for the changed feature
- separate executed checks, testcase evidence, failures, coverage gaps, and runtime follow-ups
- do not modify code, write new tests, or sync the spec during verification
- flag unresolved questions, blocked checks, or partial coverage explicitly
- write detailed evidence to `docs/ai/verifications/{feature}.md`
- update only checklist icons, short evidence notes, human task markers, summary percentages, evidence path, and drift findings
- update the checklist before every outcome, including fail or blocked
- do not include runtime-only evidence in this phase; leave runtime behavior to `/verify-runtime`

Typical checks when relevant:
- lint
- typecheck
- build
- existing unit or integration tests
- migration check or dry-run
- docker build or packaging validation

Evidence allocation rules:
- the default evidence for an observable feature flow is `/verify-runtime`, using bounded MCP/browser E2E checks when a runtime target is available
- the checklist surfaces UX quality, business judgment, and other human-only checks without duplicating detailed evidence
- `/execute-spec` may add or update a focused automated test only for the risk-sensitive cases above; it must not create a test suite or test infrastructure solely for compliance
- `/verify-feature` verifies existing or newly added focused tests but does not write tests itself
- when risk-sensitive behavior has neither focused automated evidence nor a credible runtime/manual path, record it as a coverage gap; the absence of unit/integration tests alone is not a failure for UI-first, simple, or test-hostile projects

Expected output sections:
- `## Sources`
- `## Implementation Surfaces`
- `## Evidence Strategy`
- `## Executed Checks`
- `## Testcase Evidence`
- `## Failed`
- `## Coverage Gaps`
- `## Needs Runtime Verification`
- `## Spec Gaps / Drift`
- `## Checklist Update`
- `## Final Status`

Section intent:
- `## Sources`: approved spec, relevant code paths, existing verification artifact if present
- `## Implementation Surfaces`: files, components, routes, scripts, or other concrete surfaces mapped to ACs
- `## Evidence Strategy`: the smallest evidence type selected for each AC or behavior area, with a reason
- `## Executed Checks`: only checks actually run in this phase, with command or inspection method
- `## Testcase Evidence`: detailed evidence and result for each testcase evaluated in this phase
- `## Failed`: ACs or checks that failed, with concrete reason
- `## Coverage Gaps`: ACs or behaviors not proven yet
- `## Needs Runtime Verification`: observable behaviors that still need browser/manual/runtime proof
- `## Spec Gaps / Drift`: unclear requirements or implementation behavior that conflicts with the approved spec
- `## Checklist Update`: green, yellow, and red testcase counts after implementation verification
- `## Final Status`: one of `Pass`, `Partial`, `Fail`, `Blocked`

Status rules:
- `Pass`: all implementation-level checks needed for this phase passed and no material coverage gaps remain
- `Partial`: some checks passed but meaningful coverage gaps remain, especially risk-sensitive behavior without a credible evidence path
- `Fail`: at least one required implementation-level check failed
- `Blocked`: the phase could not complete because required inputs, environment, or artifacts were unavailable

Deferred runtime or manual proof alone does not make `/verify-feature` `Partial` when the evidence strategy explicitly assigns it to `/verify-runtime` or `/manual-checklist`.

Overwrite rules:
- `/verify-feature` may update the implementation-level sections above
- `/verify-feature` must not delete valid runtime sections that were previously appended by `/verify-runtime`
- when older content conflicts with current evidence, replace only the conflicting section and note the reason in `## Executed Checks` or `## Failed`
- `/verify-feature` must not redefine checklist testcases from code or summary content

### `/verify-runtime`
Purpose:
- verify runtime behavior for the spec-derived checklist testcases after implementation verification is complete
- leave the updated checklist as the primary human-facing workflow artifact

Rules:
- read the approved spec, checklist, and current verification artifact before runtime checks
- if the checklist or verification file does not exist, stop as blocked
- if the verification file already exists from another session, append or update runtime sections in place instead of recreating the file from scratch
- classify testcases as automatically verifiable, manual-only, or blocked before execution
- verify only observable runtime behavior; do not infer hidden system behavior without evidence
- record detailed evidence and a status for each testcase checked at runtime
- do not modify code, sync the spec, or repair failures during runtime verification
- append or update runtime verification sections in `docs/ai/verifications/{feature}.md`
- update checklist icons, short evidence notes, human task markers, summary percentages, evidence path, and drift findings
- update the checklist before every outcome, including fail or blocked
- do not rewrite implementation-level sections produced by `/verify-feature` except to add a narrow cross-reference when runtime evidence changes the overall conclusion

Per-testcase runtime results:
- `Pass`
- `Fail`
- `Partial`
- `Blocked`
- `Not automatically verifiable`

Final runtime status:
- `Pass`
- `Fail`
- `Partial`
- `Blocked`

Status rule:
- `Not automatically verifiable` is valid only at the acceptance-criterion level, not as the final runtime status

Expected output sections:
- `## Runtime Target`
- `## Runtime Testcase Evidence`
- `## Automated Runtime Checks`
- `## Manual Follow-ups`
- `## Spec Gaps / Drift`
- `## Final Checklist Update`
- `## Runtime Status`

Runtime append rules:
- `/verify-runtime` owns only the runtime sections above
- keep existing implementation-level sections intact
- if runtime evidence contradicts an earlier implementation-only pass, preserve the earlier section and record the contradiction explicitly in runtime sections
- preserve testcase definitions and downgrade checklist status when runtime evidence contradicts earlier evidence

Evidence rules:
- do not claim `no overflow`, `responsive`, `works on mobile`, or similar layout outcomes from CSS declarations alone when runtime measurement is available
- prefer concrete evidence such as viewport dimensions, DOM counts, visible text, screenshots, console output, `scrollWidth/clientWidth`, or browser-evaluated state
- when a claim cannot be proven in the current environment, mark it `Partial`, `Blocked`, or `Not automatically verifiable` instead of upgrading it to `Pass`
- never derive a green icon from agent confidence, code inspection, lint, typecheck, build, or a narrower testcase

## Human-Controlled Execution

The workflow is not mode-based.

The human decides which step to invoke next:
- `/execute-task` for small, local work
- `/design-spec` when a feature needs material high-level human decisions
- `/spec` after design approval or when standalone decisions are already explicit
- `/execute-spec` after `review-spec` has accepted the detailed spec
- `/sync-spec` only when the human intentionally wants to reconcile the approved spec with implementation
- `/verify-feature` when checking implementation readiness against the approved spec
- `/verify-runtime` when checking runtime behavior against the approved spec
- `/manual-checklist` to generate or regenerate spec-derived testcases; orchestrator runs it automatically after execution
- `/review-pr` when the human wants an independent, evidence-bound review before creating a PR

Agent behavior rules:
- do not silently choose a larger or smaller workflow mode
- if the current step is too early or too late, say so explicitly and recommend the next step
- if a request is too broad or feasibility is uncertain, `/design-spec` or `/spec` must stop at the earliest step that discovers the issue
- if the human explicitly invokes `/orchestrator`, follow the workflow config until the next stop condition instead of reverting to manual step-by-step mode mid-run

### `/manual-checklist`
Purpose:
- create the human validation testcases immediately after execution
- derive testcase definitions and expected results only from the approved spec
- give verification steps a stable human-facing artifact to update
- let the human focus on remaining yellow and red testcases instead of reading implementation code

When to run:
- automatically after `/execute-spec` in `feature-standard`
- automatically after `/execute-gnhf` in `feature-implement-gnhf`
- directly when the human wants to regenerate the checklist from an approved spec

Input and sequencing:
- consume only the approved spec when defining testcases
- the orchestrator requires `summary_path` only to prove execution completed before checklist generation
- do not read code, summary, verification, or runtime behavior to define expected results

Testcase rules:
- each item is one independently executable testcase, not one acceptance criterion
- one AC may map to multiple happy-path, negative, boundary, persistence, fallback, responsive, or failure testcases
- every testcase must map to one or more AC identifiers
- preserve stable IDs such as `TC-001`
- if the spec is unclear, create a red spec-gap entry instead of inventing behavior

Evidence icon rules:
- `🟢`: the exact testcase passed with direct evidence for its action, expected result, preconditions, and required environment
- `🟡`: evidence exists but is indirect, incomplete, environment-limited, or still requires human judgment
- `🔴`: not run, failed, blocked, contradictory, or based on an unclear spec rule
- icons represent evidence classification, never agent confidence
- lint, typecheck, build, code inspection, apparent correctness, or intent alone can never produce green
- observable UI behavior requires direct runtime or E2E evidence for green
- contradictory later evidence must downgrade green

Checklist update ownership:
- `/manual-checklist` owns testcase definitions, mappings, expected results, and initial red status
- `/verify-feature` and `/verify-runtime` may update only icons, short evidence notes, human task markers, summary percentages, evidence path, and drift findings
- the complete checklist and later verification updates to it must remain in Vietnamese
- verification must update the checklist before returning pass, partial, fail, or blocked
- green testcase items may omit an unchecked human checkbox
- yellow and red testcase items retain a human task checkbox
- verifier steps never auto-check or erase an existing human checkmark
- detailed commands, logs, screenshots, assertions, and reasoning belong in `docs/ai/verifications/{feature}.md`

Expected output sections:
- `## Tóm tắt xác minh` with green, yellow, and red testcase counts and percentages
- `## Chú giải bằng chứng`
- `## Các ca kiểm thử` as one-line Markdown task items with icon, testcase ID, spec mapping, expected result, and short AI verification note
- `## Lỗ hổng đặc tả / Sai lệch`
- `## Xác nhận của người kiểm tra`
- `## Nguồn`

Percentage rules:
- calculate percentages from testcase count, not AC count
- always show exact fractions and nearest-whole percentages
- count an AC as fully covered only when every testcase mapped to it is green

### `/review-pr` (Human-Triggered)
Purpose:
- independently review the PR diff and existing evidence after manual checks
- separate direct evidence that must be fixed from inferred risks, human decisions, and manual-only verification
- prepare a concise PR-ready review artifact; do not create or approve a PR

When to run:
- only after `/manual-checklist` is complete and human-only checks have been completed or explicitly accepted as deferred
- use an explicit base ref to define the PR diff

Rules:
- read the spec, summary, verification artifact, checklist, and base diff directly
- write `docs/ai/reviews/{feature}.md` in Vietnamese
- classify every finding as `Verified`, `Observed, limited scope`, `Inferred risk`, `Human decision required`, or `Manual verification required`
- use `Must fix` only for direct evidence or a direct violation of an approved spec/safety rule; do not turn assumptions into bugs
- report `Needs Fix`, `Needs Human Decision`, `Ready for Human PR Approval`, or `Blocked`
- do not modify feature code, tests, specs, summaries, verification artifacts, or the checklist
- if a fix is required, return to implementation and repeat the affected verification and human-requested review steps before another PR review

## Workflow Artifacts

| Artifact path | Produced by |
|---|---|
| `docs/ai/designs/{feature}.html` | `/design-spec` as the interactive human review surface |
| `docs/ai/design-decisions/{feature}.json` | `/design-spec` after explicit approval, as provenance for spec creation and review |
| `docs/ai/specs/{feature}.md` | `/spec`; optionally updated by human-triggered `/sync-spec` |
| `docs/ai/summaries/{feature}.md` | `/execute-spec` as an execution handoff summary, not final proof |
| `docs/ai/verifications/{feature}.md` | `/verify-feature` and `/verify-runtime` as the detailed evidence log |
| `docs/ai/checklists/{feature}.md` | `/manual-checklist` from the approved spec, then updated by both verification steps |
| `docs/ai/reviews/{feature}.md` | `/review-pr` as an independent PR readiness review |

## Usage Notes
- Choose the lightest workflow that still gives enough control.
- Use `/execute-task` instead of creating a design artifact for small bounded changes.
- Keep high-level human judgment in `/design-spec` and detailed codebase planning in `/spec`.
- Let the human control step entry; let the agent enforce the gate conditions inside the chosen step.
- If the task changes shape during execution, re-scope the workflow instead of forcing the original path.
- Keep durable workflow knowledge in the spec; keep temporary execution reasoning out of persistent workflow artifacts unless it remains useful after implementation.
- Keep the decision manifest limited to approval provenance; do not turn it into a second implementation spec.
- `/manual-checklist` exists to reduce human code-reading and context-switch; orchestrator creates it after execution and returns its final updated form after verification.
- `/review-pr` is the final human-triggered quality gate before PR creation; it must distinguish direct evidence from agent assumptions and human-only verification.
