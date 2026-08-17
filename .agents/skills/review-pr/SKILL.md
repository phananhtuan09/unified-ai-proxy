---
name: review-pr
description: Perform a focused, high-confidence review of a pull request, branch, commit range, or working-tree diff. Prioritize merge-blocking defects that AI-generated changes commonly introduce, with or without PR context, and avoid broad speculative review or generic best-practice advice.
---

# Review PR

Review a proposed code change without modifying it.

The goal is not to enumerate every possible risk.

The goal is to find a small number of concrete defects that are likely to matter before merge and that can be supported by reviewable evidence.

## Inputs

A review requires a determinable change set, such as:

- a PR URL or number
- a head and base ref
- a commit range
- the current branch relative to its merge base
- a staged, unstaged, or working-tree diff

Prefer the target supplied by the user.

If no target is supplied, infer the most reasonable local diff and state the inference.

Ask one focused question only when multiple materially different change sets are plausible.

Return `Blocked` only when no reliable change set can be obtained or inspected.

Use any context the user supplies, including a PR description, issue, requirements, design notes, risk area, or verification evidence.

Context may use any location or format and is never required to perform a diff review.

## Review Modes

Use `Context-Enriched Review` when explicit intent or requirements are available.

Use `Diff-Only Review` otherwise.

In `Diff-Only Review`, infer the apparent purpose only to orient the review.

Do not invent business requirements or claim that unstated behavior is wrong.

State that business-rule completeness could not be verified without additional context.

## Core Review Criteria

Review only these five areas by default.

### 1. Direct Intent Contradictions

Apply this criterion only when explicit PR context or requirements are available.

Look for implementation behavior that directly contradicts the stated behavior, acceptance criterion, compatibility constraint, or safety rule.

Do not report behavior that is merely unspecified.

### 2. Broken Changed Code Paths

Trace the paths changed by the diff and look for concrete failures such as:

- an incorrect condition, return value, state transition, or error outcome
- a reachable null, undefined, type, or exception failure
- an error being swallowed or incorrectly converted into success
- cleanup, transaction, or state restoration missing on a changed failure path
- a failing build, type check, focused test, or other executable validation caused by the change

Do not enumerate hypothetical edge cases unless the changed code makes the failure path concrete and reachable.

### 3. Incomplete Cross-File Changes

Search for references to changed symbols, contracts, schemas, configuration, and data shapes.

Look for incomplete updates such as:

- a changed signature with stale callers
- a backend contract that no longer matches its client or consumer
- a renamed or removed value still referenced elsewhere
- a schema or model change missing required serialization, migration, validation, or mapping updates
- a new dependency, environment variable, route, export, or configuration value that is not wired into the project

This criterion should rely on repository search and surrounding code rather than assumption.

### 4. Invalid Technical Assumptions

Check whether the change relies on an API, type, library behavior, return shape, lifecycle, or repository convention that is contradicted by local code, installed types, tests, or authoritative documentation available in the environment.

Common examples include calling a nonexistent method, handling the wrong return type, assuming an async operation is synchronous, or using a framework API in the wrong lifecycle.

Do not report an assumption as invalid until it has been checked against a reliable source.

### 5. Clear Security Or Data-Safety Violations

Apply this criterion only when the diff touches a trust boundary, authorization decision, sensitive data, persistent mutation, or migration.

Report only concrete violations such as:

- a required repository-standard authorization check is bypassed or removed
- untrusted input reaches a dangerous operation without the validation used by equivalent code paths
- a secret or sensitive value is committed or exposed
- an update or delete operation targets broader data than the surrounding contract permits
- a migration or changed write path demonstrably loses or corrupts existing data

Do not perform a generic security checklist when the diff does not touch these surfaces.

## Conditional Review

Review performance, concurrency, idempotency, retry behavior, timezone handling, accessibility, UX polish, observability, deployment, rollback, or general maintainability only when at least one of these conditions is true:

- the user explicitly asks for that review
- the supplied requirements identify it as a constraint
- the diff directly changes that behavior and a concrete defect can be demonstrated

Do not report style preferences, speculative future risks, optional refactors, or generic best-practice advice during the default review.

Do not report missing tests by itself.

Use tests and focused commands to prove or disprove a suspected defect.

Report a test defect only when the test demonstrably fails to exercise its claimed behavior, can pass while the changed behavior is broken, or was weakened by the PR.

## Review Procedure

1. Read applicable repository instructions and determine the target, base, and review mode.
2. Inspect the exact diff and summarize its apparent purpose in one or two sentences.
3. Read only the surrounding code needed to trace the changed paths.
4. Search the repository for changed symbols, callers, contracts, schemas, and configuration.
5. Run the smallest relevant build, type, lint, or focused test commands that can validate concrete concerns.
6. Report only findings that meet the finding bar below.

Stop expanding the review once all five core criteria have been evaluated for the changed surfaces.

Do not widen the review into unrelated pre-existing code.

## Finding Bar

Report a defect only when all of the following can be stated:

- the exact changed or affected code location
- the concrete input, state, or code path that triggers the problem
- the incorrect observable outcome
- evidence from code, requirements, repository search, types, documentation, or command output
- the condition required to resolve the problem

If the trigger or impact cannot be stated concretely, do not report the issue as a finding.

Do not use vague claims such as "might fail", "could be improved", "consider adding", or "may be a problem".

Use `Must fix` only for a defect established by direct evidence or a complete static reasoning chain.

Use `Human decision` only when missing intent prevents determining whether a material changed behavior is correct.

Do not convert uncertainty into a defect.

## Final Status

- `Needs Fix`: one or more `Must fix` findings remain.
- `Needs Human Decision`: no established defect remains, but missing intent prevents a material merge decision.
- `Ready for Human PR Approval`: no blocking defect or material unresolved decision was found within the reviewed scope.
- `Blocked`: the change set cannot be reliably determined or inspected.

`Ready for Human PR Approval` does not prove the absence of defects.

It means the focused review found no issue meeting the reporting bar.

## Output

For a standalone invocation, return the review in the conversation and do not create a file unless requested.

When the user or an orchestrated workflow supplies an output path, write the review there.

Use Vietnamese unless the user requests another language.

Preserve identifiers, paths, commands, and canonical status labels in English when useful for traceability.

Present findings first and order them by impact.

Use this structure:

```markdown
# PR Review — {target}

## Findings
- [High] PR-01 — `Must fix`
  - Location: ...
  - Trigger: ...
  - Outcome: ...
  - Evidence: ...
  - Required condition: ...

## Human Decisions
- PR-02 — `Human decision`
  - Missing intent: ...
  - Why it affects merge readiness: ...

## Review Scope
- Status: Needs Fix | Needs Human Decision | Ready for Human PR Approval | Blocked
- Mode: Context-Enriched Review | Diff-Only Review
- Target, base, and diff source: ...
- Commands run: ...
- Context used: ...
- Limitations: ...
```

If no findings exist, write `Không phát hiện lỗi nào đạt ngưỡng báo cáo.`

Do not add non-blocking suggestions unless the user explicitly requests them.

## Orchestrator Contract

When an orchestrator invokes the skill, read and follow [references/orchestrator-contract.md](references/orchestrator-contract.md).

Do not load or apply that contract during a standalone review.
