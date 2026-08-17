---
name: manual-checklist
description: "Create a Vietnamese spec-derived testcase checklist after implementation so verification can update evidence status and the human can validate only the remaining cases. Runs automatically under supported orchestrator workflows and may be invoked directly to regenerate the checklist from an approved spec."
---

# Manual Checklist

Create the human validation checklist from the approved spec.

## Role In The Workflow

- Under `feature-standard`, run automatically after `/execute-spec` and before `/verify-feature`.
- Under `feature-implement-gnhf`, run automatically after `/execute-gnhf` and before `/verify-feature` in the implementation workspace.
- When invoked directly, regenerate the checklist from the provided approved spec.
- This skill creates testcase definitions but does not verify them.
- `/verify-feature` and `/verify-runtime` update testcase evidence and status later.
- The checklist is the primary human-facing output of the completed workflow.

## Input

- Required: approved spec path, for example `docs/ai/specs/{feature}.md`.
- The orchestrator may require `summary_path` to prove execution completed, but this skill must not read the summary to define expected behavior.

## Source Of Truth Boundary

- The approved spec is the only source of truth for testcase definitions and expected results.
- Derive testcases only from acceptance criteria, key behavioral rules, edge cases, failure states, validation, persistence, fallback, and reset behavior explicitly present in the spec.
- Do not read code, implementation summaries, verification artifacts, or runtime behavior to add, remove, or rewrite testcases.
- Do not convert implemented behavior into an expected result when the spec does not require it.
- If the spec does not define an expected result clearly enough, record a red spec-gap testcase instead of inventing behavior.
- Verification evidence may change testcase status but must never change the testcase definition or expected result.

## Testcase Generation Rules

- Each checklist item represents one independently executable testcase, not one acceptance criterion.
- One acceptance criterion may produce multiple testcases for happy path, negative path, boundary conditions, persistence, fallback, or relevant environments.
- Map every testcase to one or more explicit spec acceptance criteria such as `AC1` or `AC1, AC3`.
- Use stable sequential IDs such as `TC-001`, `TC-002`, and `TC-003`.
- Keep each testcase concise but executable by including action and expected result on one line.
- Do not generate testcases for behavior that is out of scope.
- Do not create generic testcases such as "works correctly" or "handle errors properly".
- If an existing checklist is regenerated from an unchanged testcase, preserve its testcase ID and human checkbox state.
- If the approved spec changes, refresh affected testcase definitions and mark their evidence status red until verification runs again.

## Evidence Status Rules

Icons represent evidence status determined by rules, never the agent's confidence.

- `🟢`: The exact testcase was executed and passed with direct evidence covering its stated action, expected result, preconditions, and relevant environment.
- `🟡`: Some evidence exists, but it is indirect, incomplete, narrower than the testcase, environment-limited, or still requires human judgment.
- `🔴`: The testcase was not run, failed, was blocked, has contradictory evidence, or cannot be evaluated because the spec is unclear.

The checklist generator initializes every testcase as `🔴` with evidence `Chưa chạy`.

AI agents must not assign `🟢` from any of the following alone:

- code inspection or an implementation that appears correct
- lint, typecheck, build, or compilation success
- the agent's intention, reasoning, confidence, or expected implementation behavior
- a test that does not execute the exact testcase behavior
- one happy-path test used to claim negative, boundary, persistence, responsive, or fallback cases
- a mocked or narrower environment when the testcase requires a different real environment

For observable UI behavior, `🟢` requires direct runtime or E2E evidence.
A focused unit or integration test may support `🟢` only for a deterministic non-UI rule when it directly exercises the testcase inputs, state transition, and expected output.
Later contradictory evidence must downgrade an existing `🟢` to `🟡` or `🔴`.

## Percentage Rules

- Percentages use testcase count as the denominator, not acceptance-criterion count.
- Green percentage is `green testcases / total testcases`.
- Yellow percentage is `yellow testcases / total testcases`.
- Red percentage is `red testcases / total testcases`.
- Round displayed percentages to the nearest whole percent and always show the exact fraction.
- An acceptance criterion is fully covered only when every testcase mapped to it is green.

## Language Rules

- Write the complete checklist file in Vietnamese.
- Write the human-readable command response in Vietnamese.
- Translate testcase actions, expected results, summaries, legends, status notes, and sign-off instructions into clear Vietnamese even when the approved spec is written in English.
- Preserve testcase IDs, acceptance-criterion IDs, file paths, commands, code symbols, product names, and technical identifiers in their original form.
- Avoid mixed English labels when a clear Vietnamese equivalent exists.
- Keep established evidence icons and orchestrator contract fields unchanged.

## Output

Write to `docs/ai/checklists/{feature}.md` in Vietnamese.

Use this format:

```markdown
# Checklist kiểm thử thủ công — {feature}

## Tóm tắt xác minh
- Tổng số ca kiểm thử: {N}
- 🟢 Đã xác minh đầy đủ: {green}/{N} ({green_percent}%)
- 🟡 Có bằng chứng một phần: {yellow}/{N} ({yellow_percent}%)
- 🔴 Cần người kiểm tra: {red}/{N} ({red_percent}%)
- Tiêu chí chấp nhận được bao phủ đầy đủ: {covered_ac}/{total_ac}
- Bằng chứng chi tiết: {verification_path | Chưa có — chưa chạy xác minh}

## Chú giải bằng chứng
- 🟢 Bằng chứng trực tiếp cho thấy ca kiểm thử đã đạt đầy đủ.
- 🟡 Bằng chứng gián tiếp, chưa đầy đủ, hoặc chưa bao phủ toàn bộ phạm vi.
- 🔴 Chưa chạy, không đạt, bị chặn, có bằng chứng mâu thuẫn, hoặc đặc tả chưa rõ.

## Các ca kiểm thử
- [ ] 🔴 TC-001 [AC1] — {hành động} → {kết quả mong đợi} — AI: Chưa chạy
- [ ] 🔴 TC-002 [AC1, AC2] — {hành động} → {kết quả mong đợi} — AI: Chưa chạy

## Lỗ hổng đặc tả / Sai lệch
- Không có.

## Xác nhận của người kiểm tra
- [ ] Đã hoàn thành các ca kiểm thử còn ô checkbox.
- [ ] Đã chấp nhận các ca kiểm thử màu xanh và không cần kiểm tra lại.
- [ ] Tính năng đạt yêu cầu hoặc đã ghi rõ các ca kiểm thử chưa đạt.

## Nguồn
- Đặc tả đã duyệt: docs/ai/specs/{feature}.md
- Bằng chứng xác minh: docs/ai/verifications/{feature}.md
```

## Checklist Mutation Rules

- This skill owns testcase definitions, spec mapping, expected results, the legend, and the initial red status.
- Verification skills own only the status icon, short AI verification note, calculated summary, evidence path, and `Lỗ hổng đặc tả / Sai lệch` findings.
- Verification skills may remove an unchecked `[ ]` only when the testcase becomes green.
- Verification skills must never change `[ ]` to `[x]` or erase an existing `[x]`.
- Yellow and red testcases must retain their human task checkbox.
- Keep the AI verification note to one short method-and-result phrase.
- Store commands, assertions, screenshots, logs, detailed reasoning, and full evidence in `docs/ai/verifications/{feature}.md`, not in the checklist.

## Done When

- Every relevant behavior explicitly required by the spec has at least one testcase.
- Every testcase maps back to an acceptance criterion.
- Every testcase starts red with no verification claim.
- The checklist contains no expected behavior derived from code or summaries.
- The complete human-facing checklist content is written in Vietnamese.
- The file is written to `docs/ai/checklists/{feature}.md`.

## Orchestrator Contract

When this skill is run under `/orchestrator`, append exactly one HTML comment as the final output line:

- Checklist written:
  `<!-- orchestrator: outcome=continue provides=checklist_path checklist_path=docs/ai/checklists/{feature}.md -->`
- The approved spec is missing or cannot produce honest testcases:
  `<!-- orchestrator: outcome=stop-blocked -->`

Rules:

- Emit the comment only after the main human-readable response is complete.
- `checklist_path` must match the file actually written.
- If this skill runs standalone, the comment is optional.
