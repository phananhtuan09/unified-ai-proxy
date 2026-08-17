---
name: execute-gnhf
description: Run a progress-gated, multi-epoch GNHF implementation from an approved feature spec, preserve each successful iteration as a Git commit in an isolated worktree, enforce feature-level budgets, and hand explicit workspace and artifact contracts back to the repository orchestrator. Use for the execute-gnhf step of an implementation workflow or when the user explicitly wants GNHF to implement an already reviewed spec.
---

# Execute GNHF

Run GNHF as the implementation engine for one approved feature spec.
Treat GNHF completion as a candidate implementation handoff, not final acceptance.

## Input

Required:

- Approved spec path under `docs/ai/specs/`.

Optional:

- Agent name.
  Default: `codex`.
- Iterations per epoch.
  Default: `5`.
- Maximum input plus output tokens per epoch.
  Default: `250000`.
- Maximum epochs for the complete feature run.
  Default: `4`.
- Maximum cumulative input plus output tokens for the complete feature run.
  Default: `1000000`.
- Maximum consecutive epochs without meaningful progress.
  Default: `1`.
- Automatic continuation after a productive budget-limited epoch.
  Default: `true`.
- Optional natural-language runtime request for additional GNHF behavior.
- Additional repository-specific verification commands or runtime notes.

Accept legacy `additional_iterations` and `max_tokens` inputs as aliases for `iterations_per_epoch` and `tokens_per_epoch`.
Persist the resolved values so a resumed run uses the same policy unless the human explicitly overrides it.
Treat `runtime_request=none` as no additional runtime customization.

Validate inputs before repository preflight:

- require positive integers for every iteration, epoch, and token limit
- require `auto_continue` to resolve to exactly `true` or `false`
- reject conflicting canonical and legacy alias values instead of choosing silently
- require the selected agent to match a form supported by the installed `gnhf --help`
- stop with `stop-ask-human` when an unsupported runtime request materially changes the requested execution behavior

## Preconditions

Complete every check before launching GNHF.

1. Run `gnhf --version` and `gnhf --help`, then confirm the installed CLI supports `--worktree`, `--agent`, `--max-iterations`, `--max-tokens`, and `--stop-when`.
   When the human supplies a runtime request, resolve it against the current help output before constructing the command.
   Use only flags shown by the installed CLI, preserve the human's requested value, and report any request that cannot be represented instead of guessing.
   Construct each flag at most once; a supported human override replaces the skill default instead of being appended as a duplicate.
   Never allow a runtime request to remove `--worktree`, replace the approved stop condition, or enable `--push` or `--current-branch`.
   When the installed help does not expose a task-local retry flag, read the effective `maxConsecutiveFailures` from `~/.gnhf/config.yml` when present and report that GNHF owns retry and backoff; do not mutate global GNHF config for one task.
   If the config does not exist or does not declare that value, report that the effective retry count is not exposed for task-level override instead of inventing one.
2. Resolve the Git repository root and use paths relative to that root in the worker prompt and orchestrator contract.
3. Confirm the spec exists, is inside the repository, is tracked by Git, and has no staged or unstaged changes.
   When orchestrator `feature_slug` is available, require it to equal the spec filename without `.md`.
4. Read the spec completely.
5. Stop with `stop-ask-human` if an open question or missing decision can change user-visible behavior.
6. Stop with `stop-too-broad` if the spec is an unsliced epic rather than one executable feature slice.
7. Keep orchestrator runtime files local by adding these exact entries to the repository's Git info exclude file when they are not already present:

   ```text
   docs/ai/workflows/runs/
   docs/ai/workflows/.orchestrator-runs.json
   docs/ai/workflows/.orchestrator-lock.json
   ```

8. Stop with `stop-blocked` if any of those runtime paths are already tracked.
9. Confirm the remaining working tree is clean.
10. Record the current HEAD commit, branch, and `git worktree list --porcelain` output.
11. Do not launch from an existing `gnhf/*` branch.
12. Do not use `--push` or `--current-branch`.

When resuming after `stop-budget`, `stop-total-budget`, or `stop-no-progress`:

1. Require the previous `implementation_workspace_path` from orchestrator state.
2. Confirm that worktree is still registered and contains the matching GNHF run metadata.
3. Load `.gnhf/runs/<run-id>/budget.json` from the preserved worktree and verify it matches the spec, prompt, agent, and base commit.
4. Count the completed `iteration-<n>.jsonl` files and reconcile them with the ledger before launching.
5. Reuse the exact worker prompt, agent, stop condition, and repository root so GNHF resumes the preserved run.
6. Apply only explicit human overrides to the stored budget policy or runtime request.

GNHF token totals restart for each resumed CLI invocation.
The skill-owned budget ledger provides the cumulative feature limit across invocations.

Do not commit, stash, reset, clean, or overwrite existing user changes to make preflight pass.
Report the exact dirty paths instead.

## Worker Contract

Build one stable worker prompt around the approved spec.
Keep the same prompt for resume compatibility.

The prompt must include these rules:

```text
Objective: implement the approved feature spec at <spec_path>.

You are the inner coding worker of an existing GNHF run.
Never invoke gnhf, execute-gnhf, or the repository orchestrator from inside this worker.

Treat these approved spec sections as immutable business contracts:
- Execution Contract
- Problem
- Scope
- Approved Design Decisions
- Behavioral Requirements
- Acceptance Criteria
- Out of Scope

For this iteration:
1. Read the GNHF notes and approved spec.
2. Select the smallest unresolved logical slice that is independently verifiable.
3. Implement only that slice and its directly related tests.
4. Reuse existing repository patterns.
5. Run the smallest relevant verification before returning.
6. Do not make unrelated refactors.
7. Do not modify approved business behavior or silently resolve product ambiguity.
8. Do not make Git commits manually.
9. Do not write the final implementation summary.
10. Stop every background process started during the iteration.

In the result summary, include one concise progress line:
PROGRESS: completed=<acceptance criteria or verified prerequisite>; remaining=<next unresolved slice>; verification=<checks run>

If an automatically loaded execute-spec skill applies, follow its product-safety and evidence rules, but the GNHF iteration contract controls scope: complete one logical slice, not the whole feature by default.

Report success only for a meaningful, verified contribution.
Report failure if progress requires an unresolved product decision.
Do not claim final acceptance.
```

Use this stop condition:

```text
Every acceptance criterion in the approved spec has an implementation, relevant automated checks pass, no unresolved product decision remains, no unrelated files are changed, and all background processes started by the worker are stopped.
```

## Budget Ledger

Store epoch accounting next to the matching GNHF run metadata:

```text
.gnhf/runs/<run-id>/budget.json
```

The ledger is local runtime state and must not be committed.
Use this minimum shape:

```json
{
  "version": 1,
  "spec_path": "docs/ai/specs/feature-name.md",
  "base_commit": "<commit>",
  "agent": "codex",
  "epochs_completed": 1,
  "cumulative_tokens": 210000,
  "tokens_estimated": false,
  "consecutive_no_progress_epochs": 0,
  "policy": {
    "iterations_per_epoch": 5,
    "tokens_per_epoch": 250000,
    "max_epochs": 4,
    "max_total_tokens": 1000000,
    "max_no_progress_epochs": 1,
    "auto_continue": true
  }
}
```

After every CLI invocation, parse its permanent exit summary and append the invocation token total to `cumulative_tokens`.
Preserve whether any token count was estimated.
Write the ledger atomically.
Stop with `stop-blocked` if existing ledger data conflicts with the current run or token accounting cannot be resolved safely.

## Launch Epochs

Run GNHF from the original repository root in Hands-Off mode:

```sh
gnhf \
  --worktree \
  --agent <agent> \
  --max-iterations <max-iterations> \
  --max-tokens <max-tokens> \
  --stop-when "<stop-condition>" \
  --prevent-sleep on \
  "<worker-prompt>"
```

Before each epoch:

1. Stop with `stop-total-budget` when `epochs_completed >= max_epochs` or `cumulative_tokens >= max_total_tokens`.
2. Set the absolute `--max-iterations` cap to completed iteration files plus `iterations_per_epoch`.
3. Set `--max-tokens` to the smaller of `tokens_per_epoch` and the remaining total token budget.
4. Apply only validated optional flags resolved from the current `gnhf --help` output.
5. Wait for the process to finish and do not edit the implementation workspace while GNHF owns it.

For the first epoch, use `iterations_per_epoch` directly as the iteration cap.
For later epochs, reuse the same prompt with `--worktree` so GNHF resumes the preserved matching run.
Provide concise progress updates during long-running epochs.

## Inspect The Result

Do not use process exit code alone as proof of success.

1. Read the permanent GNHF exit summary, including commit count, notes path, log path, and status.
2. Read the last `orchestrator:abort` event in `gnhf.log` when the log exists to determine the real stop reason.
3. Stop with `stop-fail` when the complete feature run still has zero commits.
   GNHF normally removes a no-commit worktree, so do not require a preserved path for this classification.
4. When at least one commit exists, resolve the preserved worktree path from the `gnhf: worktree preserved at ...` line.
5. Confirm the path is still registered by `git worktree list --porcelain`.
6. Resolve the implementation branch and HEAD commit inside that worktree.
7. Compare the implementation branch with the recorded base commit.
8. Stop with `stop-blocked` if the worktree, notes, log, branch, or commit range cannot be resolved safely after commits exist.
9. After each budget-limited epoch, compare the commit count with the epoch start and inspect the new successful iteration records for the required `PROGRESS:` evidence.
10. Treat the epoch as meaningful progress only when the commit count increased and the evidence identifies a verified completed slice or prerequisite.
11. Reset `consecutive_no_progress_epochs` after meaningful progress; otherwise increment it.
12. Return `stop-no-progress` when the configured no-progress limit is reached.
13. When the stop reason is `max iterations reached (...)` or `max tokens reached (...)` with meaningful progress:
    - start the next epoch automatically when `auto_continue` is true and total limits remain
    - return `stop-budget` when automatic continuation is disabled
    - return `stop-total-budget` when the next epoch would exceed the epoch or cumulative token limit
14. Return `stop-fail` for consecutive failures, an agent-reported incomplete state, a permanent provider error, or another non-resumable failure.
15. Continue to summary creation only when the stop reason is `stop condition met`, the worktree is preserved, and the implementation contains at least one commit.

Do not add an outer retry loop for provider failures.
GNHF already applies exponential backoff to retryable hard agent errors and aborts after its configured consecutive-failure limit.

Do not merge, push, cherry-pick, or remove the worktree.

## Create The Summary

Write `docs/ai/summaries/{feature-slug}.md` inside the preserved GNHF worktree.
Derive `feature-slug` from the approved spec filename.

Use this format:

```markdown
## Done
- [Material outcomes supported by the commit range and GNHF notes.]

## Not Done / Blocked
- [Remaining or blocked work, or `None reported by the GNHF run.`]

## Decisions
- [Durable implementation decisions supported by code or notes.]

## Verified
- [Checks actually executed during GNHF iterations.]

## Not Verified
- Downstream `manual-checklist`, independent `verify-feature`, and `verify-runtime` are still pending.

## GNHF Evidence
- Workspace: [absolute worktree path]
- Branch: [branch]
- Base commit: [commit]
- Head commit: [commit]
- Commit range: [base..head]
- Notes: [absolute path]
- Debug log: [absolute path]
- Stop reason: [reason]
```

Write claims only when supported by the commit range, notes, log, or command results.
Do not mark acceptance criteria as independently verified.
Do not modify the approved business sections of the spec.

## Orchestrator Contract

When this skill runs under `/orchestrator`, append exactly one HTML comment as the final output line.

Successful candidate implementation:

```html
<!-- orchestrator: outcome=continue provides=spec_path,summary_path,implementation_workspace_path,gnhf_budget_path spec_path=docs/ai/specs/{feature-slug}.md summary_path=docs/ai/summaries/{feature-slug}.md implementation_workspace_path=/absolute/path/to/worktree gnhf_budget_path=/absolute/path/to/budget.json -->
```

Missing product decision:

```html
<!-- orchestrator: outcome=stop-ask-human -->
```

Unsliced implementation scope:

```html
<!-- orchestrator: outcome=stop-too-broad -->
```

Missing tool, unsafe repository state, unresolved workspace, or agent/runtime failure:

```html
<!-- orchestrator: outcome=stop-blocked -->
```

GNHF ended without satisfying the stop condition:

```html
<!-- orchestrator: outcome=stop-fail -->
```

GNHF preserved useful commits but stopped after a productive epoch because automatic continuation is disabled:

```html
<!-- orchestrator: outcome=stop-budget provides=spec_path,implementation_workspace_path,gnhf_notes_path,gnhf_log_path,gnhf_budget_path spec_path=docs/ai/specs/{feature-slug}.md implementation_workspace_path=/absolute/path/to/worktree gnhf_notes_path=/absolute/path/to/notes.md gnhf_log_path=/absolute/path/to/gnhf.log gnhf_budget_path=/absolute/path/to/budget.json -->
```

The feature reached its total epoch or token budget:

```html
<!-- orchestrator: outcome=stop-total-budget provides=spec_path,implementation_workspace_path,gnhf_notes_path,gnhf_log_path,gnhf_budget_path spec_path=docs/ai/specs/{feature-slug}.md implementation_workspace_path=/absolute/path/to/worktree gnhf_notes_path=/absolute/path/to/notes.md gnhf_log_path=/absolute/path/to/gnhf.log gnhf_budget_path=/absolute/path/to/budget.json -->
```

The latest epoch did not produce meaningful verified progress:

```html
<!-- orchestrator: outcome=stop-no-progress provides=spec_path,implementation_workspace_path,gnhf_notes_path,gnhf_log_path,gnhf_budget_path spec_path=docs/ai/specs/{feature-slug}.md implementation_workspace_path=/absolute/path/to/worktree gnhf_notes_path=/absolute/path/to/notes.md gnhf_log_path=/absolute/path/to/gnhf.log gnhf_budget_path=/absolute/path/to/budget.json -->
```

On any budget or progress stop, report epochs completed, cumulative tokens, whether usage is estimated, current commit range, exact stop reason, and the explicit change required to continue.
Do not write the final implementation summary until the GNHF stop condition is met.

Emit only one orchestrator comment.
Make every emitted path match an existing file or directory.
Treat `implementation_workspace_path` as the working directory for downstream `manual-checklist`, `verify-feature`, and `verify-runtime` steps.

## Done When

- GNHF ran with explicit iteration and token limits in an isolated worktree.
- Productive epochs continued automatically within a bounded feature-level budget.
- The real stop reason was inspected from evidence.
- The preserved worktree and commit range were resolved.
- Epoch, token, and progress accounting was persisted in the matching GNHF run ledger.
- Budget exhaustion or lack of progress produced the matching resumable contract.
- The implementation summary was written inside the worktree.
- No merge, push, or cleanup occurred.
- The final orchestrator contract accurately reports the result.
