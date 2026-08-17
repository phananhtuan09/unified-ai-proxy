---
name: refactor
description: Use when the user asks to refactor, restructure, clean up, or improve the internal design of code without changing its behavior. Requires a clear baseline, explicit motivation, and behavior parity validation. Do not use for bug fixes, feature additions, or behavior changes — those must be separate tasks.
---

# Refactor

Improve code structure without changing observable behavior. Refactoring without a baseline is rewriting.

## Inputs

- Target scope: file, module, function, or layer to refactor
- Refactor motivation (if provided)
- Optional: existing tests that cover the scope

## Tool Mapping

- File inspection/edit tools -> inspect files with the runtime's read/edit/write tools and prefer precise targeted edits over broad rewrites
- Search tools -> use repository search to find callers, usages, and related symbols
- User clarification -> ask the user directly when contract or motivation is ambiguous
- Shell/validation tools -> run tests or builds between transformation steps when shell execution is available and appropriate

## Workflow

### 1. Clarify scope

If any is unclear, ask before touching code:

- What code is being refactored (file, module, function, layer)?
- What is the primary motivation?
- Are there external contracts (API, public interface, DB schema) that must not change?
- Is there an existing test suite? Can it be run?

### 2. Establish baseline

**Before touching any code**, document:

- **Current behavior**: what does this code do from the caller's perspective?
- **Public contract**: inputs, outputs, side effects, events emitted
- **Invariants**: what must be true after the refactor?

Use `rg` to find all callers and understand usage context.

If the current behavior cannot be described clearly: read further or ask the user. Do not refactor without a baseline.

### 3. State motivation

Refactoring must have a clear reason. Choose at least one:

| Motivation | Good signal |
|-----------|-------------|
| **Readability** | Misleading names, unclear intent |
| **Maintainability** | Change in one place requires changes in many |
| **Reduce duplication** | Same logic copied in multiple places |
| **Isolate responsibility** | One unit does too many unrelated things |
| **Improve testability** | Hard to test in isolation |
| **Prepare for change** | Upcoming feature needs a cleaner seam |

If none applies: stop. Reconsider whether refactoring is justified.

### 4. Verify safety net

Check before any edit:

- [ ] Do existing tests cover the public contract?
- [ ] Do they cover main paths and key edge cases?
- [ ] If coverage is thin: add tests first, or document the manual verification plan.

If no safety net and code is non-trivial: stop and add tests first, or confirm the user accepts the risk.

If user proceeds without tests: mark output as `Behavior parity: UNVERIFIED — no automated safety net`. Do not claim parity was confirmed.

### 5. Define safe boundaries

State these constraints before writing code:

- **External contract is frozen**: listed interfaces, exports, API endpoints must not change.
- **No behavior change**: if a behavioral change is discovered as necessary, it must become a separate task.
- **No mixed concerns**: do not fix bugs or add features inline. Note them and create follow-up tasks.
- **Incremental if large**: if the refactor touches more than ~5 files or multiple layers, define the steps before starting.

### 6. Implement

One atomic transformation at a time (e.g., extract function, then rename, then move file — each is a separate step).

For each atomic transformation:
1. Apply the change with the runtime's precise file editing tools
2. Run tests or build to catch regressions
3. If a transformation introduces a bug: revert it, do not push forward

Rules:
- Do not reformat unrelated code.
- Do not silently expand the refactor scope.

### 7. Validate behavior parity

After all transformations:

- Run the full test suite.
- Manually verify the public contract matches the baseline from Step 2.
- If behavior changed unintentionally: this is a regression — fix before marking done.

### 8. Output summary

For small refactors (≤2 files, single transformation): provide a brief summary (scope, motivation, what changed, parity status). Full template below is for medium/large refactors.

```
## Refactor Summary

**Scope**: [what was refactored]
**Motivation**: [primary reason from Step 3]

**Baseline behavior** (before):
  - [key behavioral invariants]

**Structural problems addressed**:
  - [what was wrong]

**Transformation applied**:
  - [what changed structurally]

**Boundaries preserved**:
  - External contract: unchanged / [exceptions]
  - No behavior change: confirmed / [deviations noted as follow-ups]

**Impact**:
  - Files changed: [list with brief reason each was touched]
  - Callers affected: [should be: nothing visible]
  - Complexity delta: [coupling reduced / duplication removed / testability improved]
  - Blast radius:
    - L1 direct: [modules that import or call the refactored code]
    - L2 transitive: [callers of L1 if contract change propagates]
    - L3 shared infra: [config / DB / event bus — only if touched]

**How to verify**:
  1. Run test suite — all tests pass
  2. Compare public interface vs baseline — inputs/outputs/side effects match
  3. Spot-check 1–2 callers — behavior unchanged
  4. Review diff — structural change only, no logic change
  5. (if no tests) manual verification steps taken

**Safety net**: [tests coverage before/after or manual plan]
**Behavior parity**: confirmed / UNVERIFIED — no automated safety net

**Trade-offs**:
  - [what was gained vs any added complexity]

**Follow-ups** (bugs or features noticed but not addressed):
  - [item]: [reason deferred]
```

## When To Ask The User

Ask only when:
- The public contract or scope boundary is ambiguous
- A behavior change is unavoidable and user must decide whether to proceed or split into a separate task
- Test coverage is absent and the risk decision belongs to the user

## Quality Bar

- Never start without a documented baseline
- Never mix bugfix, feature, or behavior change into a refactor task
- Incremental atomic transformations only — one logical change at a time
- Behavior parity must be verified or explicitly marked UNVERIFIED
- Follow-ups must be listed, never silently dropped
