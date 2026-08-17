---
name: verify-feature
description: Use when the user asks to verify implementation evidence against an approved spec and its spec-derived manual checklist. Writes detailed evidence to the verification artifact and updates checklist testcase status without redefining expected behavior.
---

# Verify Feature

Verify implementation-level evidence for the testcases defined by the approved spec checklist.

## Input

- Required: approved spec path, for example `docs/ai/specs/{feature-name}.md`.
- Required: checklist path, for example `docs/ai/checklists/{feature-name}.md`.
- Optional: execution summary path, for example `docs/ai/summaries/{feature-name}.md`.
- Optional: focused file or module scope when the feature touches a narrow area.

## Output

Write detailed evidence to `docs/ai/verifications/{feature-name}.md`.
Update evidence status in `docs/ai/checklists/{feature-name}.md`.

## Source Of Truth And Ownership

- Treat the approved spec as the only source of truth for expected behavior.
- Treat checklist testcase definitions and spec mappings as a projection of that spec.
- Read code, tests, and build output only as evidence about whether the implementation satisfies the spec.
- Do not add, delete, split, merge, or rewrite checklist testcases from code or implementation behavior.
- Do not change a testcase expected result to match the implementation.
- If code behavior conflicts with the spec, record drift and mark the affected testcase red.
- If a required behavior is missing from the checklist because the spec is unclear, record a spec gap instead of inventing a testcase expectation.

## Verification Workflow

1. Read the approved spec and checklist completely.
2. Read the execution summary when provided, but treat it only as a navigation aid and claimed implementation handoff.
3. Map every checklist testcase to implementation surfaces and the smallest suitable evidence strategy.
4. Run only relevant implementation-level checks.
5. Record detailed commands, inspections, assertions, results, failures, and limitations in the verification artifact.
6. Update each checklist testcase status using the deterministic evidence rules.
7. Recalculate checklist counts, percentages, fully covered AC count, and verification evidence path.
8. Update the checklist before returning any orchestrator outcome, including fail or blocked outcomes.

## Verification Artifact Format

```markdown
## Sources
- Approved spec: docs/ai/specs/{feature-name}.md
- Checklist: docs/ai/checklists/{feature-name}.md
- Summary: [optional]

## Implementation Surfaces
- TC-001 / AC1 → [file/module/function]
- TC-002 / AC1 → [file/module/function]

## Evidence Strategy
- TC-001 → [focused automated test | implementation check | runtime/E2E | human] → [reason]

## Executed Checks
- [command or inspection] → Pass/Fail/Blocked

## Testcase Evidence
| Testcase | Spec mapping | Detailed evidence | Result | Checklist status |
|---|---|---|---|---|
| TC-001 | AC1 | [command, assertion, observed output, and limitation] | Pass | 🟢 |
| TC-002 | AC1 | [evidence and remaining gap] | Partial | 🟡 |

## Failed
- [failed testcase or check with concrete reason]

## Coverage Gaps
- [testcase lacking sufficient implementation evidence]

## Needs Runtime Verification
- [testcase requiring direct runtime or E2E evidence]

## Spec Gaps / Drift
- [implementation conflict or unclear expected behavior]

## Checklist Update
- Green: {green}/{total}
- Yellow: {yellow}/{total}
- Red: {red}/{total}

## Final Status
Pass | Fail | Partial | Blocked
```

## Checklist Evidence Rules

Icons are evidence classifications, not self-reported confidence.

- Set `🟢` only when the exact testcase was executed and passed with direct evidence covering its stated action, expected result, preconditions, and required environment.
- Set `🟡` when evidence is indirect, incomplete, narrower than the testcase, environment-limited, or still requires human judgment.
- Set `🔴` when the testcase was not run, failed, was blocked, has contradictory evidence, or depends on an unclear spec rule.
- Lint, typecheck, build, compilation, code inspection, or apparent implementation correctness alone can never produce green.
- For observable UI behavior, implementation verification can produce at most yellow unless this phase executed direct runtime or E2E evidence for the exact testcase.
- A focused unit or integration test may produce green only for a deterministic non-UI rule when it directly exercises the testcase inputs, state transition, and expected output.
- One passing happy-path check must not change negative, boundary, persistence, responsive, fallback, or error-path testcases to green.
- Later contradictory evidence must downgrade green to yellow or red.
- Never change `[ ]` to `[x]` or erase an existing `[x]`.
- Green testcase items may remove an unchecked `[ ]` task marker.
- Yellow and red testcase items retain their human task marker.
- Keep the checklist AI verification note to one short method-and-result phrase.
- Put all commands, logs, screenshots, assertions, and detailed reasoning in the verification artifact.

## Checklist Integrity Rules

- Preserve testcase IDs, wording, expected results, order, and spec mappings.
- Preserve the checklist's Vietnamese section headings.
- Write updated summary labels, short AI verification notes, and spec-gap or drift findings in Vietnamese while preserving technical identifiers.
- Update only the icon, short AI verification note, human task marker, summary counts, percentages, fully covered AC count, evidence path, and spec-gap/drift section.
- Derive all percentages from testcase counts.
- Count an AC as fully covered only when every testcase mapped to it is green.
- If the verification artifact cannot be completed, update affected checklist items to red with a short blocked reason before stopping.

## Final Status Rules

- `Pass`: all implementation checks required for this phase passed and remaining runtime or human checks are explicitly assigned.
- `Partial`: implementation checks ran but material implementation evidence gaps remain.
- `Fail`: at least one required implementation check or testcase failed.
- `Blocked`: required inputs, environment, or artifacts prevented meaningful implementation verification.
- Runtime-only or human-only evidence assigned to later phases does not by itself make this phase partial.

## Artifact Boundaries

- Do not modify code, tests, specs, or testcase definitions during verification.
- Do not create test infrastructure or invent testcases in this phase.
- Existing relevant tests may be executed.
- Detailed evidence belongs in the verification artifact.
- The checklist remains concise and human-scannable.

## Orchestrator Contract

When this skill is run under `/orchestrator`, append exactly one HTML comment as the final output line:

- Final status `Pass` or `Partial`:
  `<!-- orchestrator: outcome=continue provides=verification_path verification_path=docs/ai/verifications/{feature-name}.md -->`
- Final status `Fail`:
  `<!-- orchestrator: outcome=stop-fail -->`
- Final status `Blocked`:
  `<!-- orchestrator: outcome=stop-blocked -->`

Rules:

- Emit the comment only after the verification artifact and checklist have been updated.
- `verification_path` must match the artifact actually written or updated.
- Report the checklist path prominently in the human-readable response.
- If this skill runs standalone, the comment is optional.
