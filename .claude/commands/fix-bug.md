---
name: fix-bug
description: Fix a bug using reproduce → isolate → minimal fix → regression prevention discipline.
---

Fix the reported bug with strict scope control: reproduce first, isolate root cause, apply minimal fix, prevent regression.

## Workflow Alignment

- Provide brief status updates (1–3 sentences) before each step.
- For medium/large bugs, create todos (≤14 words, verb-led). Keep only one `in_progress` item.
- **Do not expand scope**: a bugfix must not become a refactor or feature addition.

---

## Step 1: Clarify Bug Shape

If any of the following is unclear, use `AskUserQuestion` before proceeding:

- **Expected behavior**: What should happen?
- **Actual behavior**: What happens instead?
- **Trigger condition**: Under what inputs/state/sequence does it occur?
- **Reproduction path**: Can it be reproduced reliably? How?
- **Environment**: Which OS, runtime version, config, or deployment environment does this occur in? Does it occur in all environments or only specific ones?

If reproduction path is ambiguous, state confidence level explicitly before attempting a fix.

---

## Step 2: Reproduce

**Goal**: Confirm the bug exists and understand its boundaries.

- Attempt to reproduce using the provided trigger condition.
- If reproduced: document the exact reproduction steps.
- If **not** reproduced:
  1. State `Confidence: Low — bug not reproduced`.
  2. Use `AskUserQuestion` to ask: proceed speculatively or stop for more evidence?
  3. If user says **stop**: exit, request logs / repro steps / environment details.
  4. If user says **proceed**: prefix every finding in Steps 3–7 with `⚠ Speculative (not reproduced)`. Do not present analysis as fact.

Never silently continue after failed reproduction.

---

## Step 3: Isolate Root Cause

**Goal**: Distinguish between symptom, immediate cause, and underlying cause.

Trace the issue layer by layer:

| Layer | Description | Example |
|-------|-------------|---------|
| **Symptom** | Visible wrong behavior | "Button does nothing" |
| **Immediate cause** | Direct code failure | "Event handler not attached" |
| **Underlying cause** | Why the immediate cause exists | "Component re-renders wipe the ref" |

**Start from the trigger condition and trace backward**: identify where the actual state first diverges from expected, then follow the call chain upstream to the root.

**Tools:**
- Read relevant files to trace the execution path
- Grep for related symbols, event handlers, config flags
- Run relevant tests if available

State all three layers before proposing a fix.

---

## Step 4: Choose Fix Strategy

Select the **most minimal** strategy that correctly addresses the underlying cause:

1. **Minimal safe fix** — targeted change, zero behavior change outside bug scope *(prefer this)*
2. **Slightly broader fix** — only if minimal fix is brittle or treats a symptom only
3. **Refactor-level fix** — only if the root cause is structural and a targeted fix is impossible

**Rules:**
- Do not change behavior outside the bug scope unless explicitly justified.
- If strategy 3 is chosen, stop and confirm with the user before implementing.
- Document why you chose the selected strategy.

---

## Step 5: Implement Fix

**Tools:**
- Read(file_path=...) to verify context before editing
- Edit(file_path=...) for targeted modifications

**Process:**
1. Apply only the changes necessary to address the root cause.
2. Do not reformat, rename, or reorganize code unrelated to the fix.
3. If you notice other bugs while fixing: **note them separately**, do not fix inline.

---

## Step 6: Regression Prevention

Before closing, check:

- [ ] Is there an existing test that should have caught this? If so, why didn't it?
- [ ] Should a new test be added to prevent recurrence? **If yes, add it now** unless the user explicitly asks to defer.
- [ ] Can this bug recur via a similar code path elsewhere?
- [ ] Should any logging or monitoring be added at the failure point?

---

## Step 7: Output Summary

Provide a concise summary:

```
## Bug Fix Summary

**Bug**: [one-line description]
**Reproduction**: [steps or "could not reproduce — confidence low"]
**Environment**: [OS, runtime version, config — or "all environments"]
**Root cause**:
  - Symptom: [...]
  - Immediate cause: [...]
  - Underlying cause: [...]
**Fix applied**: [what changed and why this strategy was chosen]

**Impact**:
  - Files changed: [list with brief reason each was touched]
  - Behavior changed: [exactly what changed from the user's perspective]
  - Blast radius: [what else could be affected — callers, related paths, edge cases]
    - L1 direct callers: functions/modules that call the changed code
    - L2 transitive: callers of those callers, if behavior propagates
    - L3 shared infra: config, DB schema, event bus — if the fix touches these

**How to verify** (steps for reviewer):
  1. [Reproduce the original bug using the trigger condition → should no longer occur]
  2. [Run: specific test file or command]
  3. [Manually check: specific screen / endpoint / state]
  4. [Check adjacent paths: similar code that may have the same bug]

**Residual risks**: [known limitations or similar paths not yet addressed]
**Follow-ups**: [any deferred issues noted during fix]
```
