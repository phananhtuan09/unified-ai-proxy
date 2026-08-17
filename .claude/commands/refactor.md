---
name: refactor
description: Refactor code while preserving behavior — establish baseline, justify motivation, enforce safe boundaries, and validate parity.
---

Improve code structure without changing observable behavior. Refactoring without a baseline is rewriting.

## Workflow Alignment

- Provide brief status updates (1–3 sentences) before each step.
- For medium/large refactors, create todos (≤14 words, verb-led). Keep only one `in_progress` item.
- **Never mix concerns**: no bug fixes, no new features, no behavior changes inside a refactor task.

---

## Step 1: Clarify Scope and Motivation

If scope or goals are unclear, use `AskUserQuestion`:

- What code is being refactored (file, module, function, layer)?
- What is the primary motivation?
- Are there any external contracts (API, public interface, DB schema) that must not change?
- Is there an existing test suite? Can it be run locally?

---

## Step 2: Establish Baseline Behavior

**Goal**: Describe what the code currently does before touching it.

Document:

1. **Current behavior**: What does this code do from the caller's perspective?
2. **Public contract**: What inputs does it accept? What does it return or emit? What side effects does it produce?
3. **What must be preserved**: Enumerate the behavioral invariants that must hold after refactoring.

If you cannot describe the current behavior clearly, do not proceed. Read the code further or ask the user.

**Tools:**
- Read relevant files to understand current implementation
- Grep for all callers to understand usage context

---

## Step 3: State Refactor Motivation

**Goal**: Make the reason for refactoring explicit. Without a clear motivation, refactoring drifts into cosmetic churn.

Choose the primary motivation (one or more):

| Motivation | Good signal |
|-----------|-------------|
| **Readability** | Code is hard to follow, misleading names, unclear intent |
| **Maintainability** | Change in one place requires changes in many |
| **Reduce duplication** | Same logic copied in multiple locations |
| **Isolate responsibility** | One unit does too many unrelated things |
| **Improve testability** | Code is hard to test in isolation |
| **Prepare for change** | Upcoming feature requires a cleaner seam |

If no motivation from the above applies, stop and reconsider whether refactoring is justified.

---

## Step 4: Verify Safety Net

**Goal**: Confirm there is sufficient coverage to detect regressions before writing any code.

Check:

- [ ] Are there existing tests that exercise the public contract?
- [ ] Do they cover the main paths and edge cases?
- [ ] If test coverage is thin: add tests before refactoring, or document the manual verification plan.

If no safety net exists and the code is non-trivial: stop and add tests first, or confirm with the user that the risk is accepted.

If user accepts risk without tests: mark the output section as `Behavior parity: UNVERIFIED — no automated safety net`. Do not claim parity was confirmed.

---

## Step 5: Define Safe Boundaries

Before writing any code, declare constraints:

- **External contract**: List the interfaces, exports, or API endpoints that must not change.
- **No behavior change**: If a behavioral change is discovered as necessary, it must be separated into a different task.
- **No mixed concerns**: Do not fix bugs or add features inline. Note them and create follow-up tasks.
- **Incremental if large**: If the refactor touches more than ~5 files or multiple layers, break it into steps. Describe the steps before starting.

---

## Step 6: Implement Refactor

**Process:**
1. Make one atomic transformation at a time (e.g., extract function, then rename, then move file — each is a separate step).
2. Run tests (or build) after each atomic transformation to catch regressions early.
3. If a transformation introduces a bug: revert that transformation, do not push forward.
4. Do not reformat code unrelated to the refactor (separate that into a linting/formatting pass if needed).

**Tools:**
- Read before editing to confirm current state
- Edit for targeted changes
- Bash to run tests or build verification between transformations

---

## Step 7: Validate Behavior Parity

After refactoring is complete:

- Run the full test suite.
- Manually verify the public contract matches the baseline documented in Step 2.
- If behavior has changed unintentionally: this is a bug introduced by the refactor — fix it before marking done.

---

## Step 8: Output Summary

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
  - Callers affected: [who calls this code and what they observe — should be nothing]
  - Complexity delta: [coupling reduced? duplication removed? testability improved?]
  - Blast radius: [any shared code that now behaves differently]
    - L1 direct: modules that import or call the refactored code
    - L2 transitive: callers of those modules, if contract change propagates
    - L3 shared infra: config, DB schema, event bus — only if touched

**How to verify** (steps for reviewer):
  1. [Run test suite: expect all tests pass with no behavior change]
  2. [Compare public interface: inputs/outputs/side effects match baseline above]
  3. [Check callers: spot-check 1–2 call sites — behavior unchanged]
  4. [Review diff: confirm no logic change, only structural change]
  5. [If no tests: describe manual verification steps taken]

**Safety net**:
  - Tests: [coverage before/after or manual verification plan]

**Behavior parity**: [confirmed / UNVERIFIED — no automated safety net]

**Trade-offs**:
  - [what was gained vs any added complexity]

**Follow-ups** (bugs or features noticed but not addressed):
  - [item]: [reason deferred]
```
