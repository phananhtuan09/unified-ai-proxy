---
name: review-pr
description: Review a completed feature before PR creation. Use when a feature has implementation, verification, runtime evidence, and a manual checklist, and Codex or Claude Code must produce an evidence-bound PR readiness review that separates required fixes from human decisions and manual verification.
---

# Review PR

Review a completed feature for PR readiness. Do not create a PR and do not modify feature code, tests, specs, summaries, or verification artifacts.

## Input

- Required: feature slug and base ref for the PR diff
- Required: `docs/ai/specs/{feature}.md`
- Required: `docs/ai/summaries/{feature}.md`
- Required: `docs/ai/verifications/{feature}.md`
- Required: `docs/ai/checklists/{feature}.md`

If a required artifact or the base ref is unavailable, write the review with `Blocked` status and explain what is missing.

## Output

Write `docs/ai/reviews/{feature}.md` in Vietnamese. Preserve identifiers, paths, commands, and canonical status labels in English where that improves traceability.

## Review Workflow

1. Read the spec, summary, verification artifact, and manual checklist. Read them as evidence, not as proof that the implementation is correct.
2. Inspect the PR diff against the supplied base ref. State the exact diff command and reviewed commit range in the report.
3. Map changed surfaces to acceptance criteria, runtime evidence, and manual checks. Identify changed files outside the declared scope.
4. Review only these concerns when relevant to the diff:
   - scope or spec drift
   - correctness, error paths, and meaningful edge cases
   - security, authorization, validation, data loss, migration, or configuration risk
   - evidence strategy: focused automated evidence only where it has clear regression value; runtime/E2E and human checks for their assigned scope
   - maintainability issues that materially increase future change risk
   - PR hygiene: debug code, secrets, generated noise, unexplained unrelated changes, and missing release/rollback notes when applicable
5. Classify every finding by evidence status before assigning an action:
   - `Verified`: direct, reviewable evidence proves the claim, such as a failing command, a concrete spec contradiction, a committed secret, or a clearly missing required authorization check.
   - `Observed, limited scope`: behavior was observed for stated inputs/environment only. Do not generalize beyond that scope.
   - `Inferred risk`: static reasoning suggests a risk but does not prove it. Do not present it as a bug.
   - `Human decision required`: business intent, acceptable trade-off, rollout, migration, or scope needs a human decision.
   - `Manual verification required`: a real environment, real data, device, third party, or UX judgment is required.
6. Classify the required action:
   - `Must fix`: only for a `Verified` blocker or a direct violation of an approved spec/safety rule.
   - `Human decision`: do not guess; state the decision, options, and concrete impact.
   - `Human verify`: provide an executable check and expected result.
   - `Suggestion`: non-blocking improvement with rationale.
7. Determine final status:
   - `Needs Fix`: one or more `Must fix` items remain.
   - `Needs Human Decision`: no blocker remains, but a human decision is unresolved.
   - `Ready for Human PR Approval`: no blocker or unresolved decision remains; required manual checks are complete or explicitly accepted as deferred by the human.
   - `Blocked`: review scope, base ref, or a required artifact is unavailable.

## Evidence Rules

- Never claim that an absence of findings proves the absence of defects.
- Never upgrade `Inferred risk`, `Human decision required`, or `Manual verification required` into `Verified` without direct evidence.
- Never demand unit or integration tests merely as ceremony. Treat missing focused automated coverage as a gap only when the changed behavior is risk-sensitive and lacks a credible runtime/manual evidence path.
- Do not trust an implementer summary by itself. Anchor every finding to the diff, a source artifact, a command result, or an observed runtime record.
- If review findings require code changes, stop after recording them. The feature must return to implementation and the verification sequence must be repeated before another PR review.

## Review Artifact Format

```markdown
# PR Review — {feature}

## PR Readiness
Needs Fix | Needs Human Decision | Ready for Human PR Approval | Blocked

## Review Scope
- Base ref and diff command: ...
- Reviewed commits/files: ...
- Source artifacts: ...

## Must Fix
- PR-01 — `Verified` — [severity]
  - Evidence: ...
  - Required fix: ...

## Risks / Human Decisions
- PR-02 — `Inferred risk` | `Human decision required`
  - Evidence or assumption: ...
  - Decision / options / impact: ...

## Manual Verification
- PR-03 — `Manual verification required`
  - Steps: ...
  - Expected result: ...

## Non-blocking Suggestions
- PR-04 — [rationale]

## Evidence Reviewed
- [commands, tests, runtime observations, checklist items, and their limits]

## Suggested PR Summary
- [scope, behavior, verification, risk/rollback notes]
```

Keep empty sections with `- Không có.` so a human can distinguish no findings from omitted review.

## Orchestrator Contract

When run under `/orchestrator`, append exactly one HTML comment as the final output line:

- `Ready for Human PR Approval`:
  `<!-- orchestrator: outcome=continue provides=pr_review_path pr_review_path=docs/ai/reviews/{feature-name}.md -->`
- `Needs Fix`:
  `<!-- orchestrator: outcome=stop-fail -->`
- `Needs Human Decision`:
  `<!-- orchestrator: outcome=stop-ask-human -->`
- `Blocked`:
  `<!-- orchestrator: outcome=stop-blocked -->`

Emit the comment only after the human-readable review is complete. If run standalone, the comment is optional.
