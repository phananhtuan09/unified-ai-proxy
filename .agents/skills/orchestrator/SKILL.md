---
name: orchestrator
description: "Use when the human wants to execute a documented workflow config end-to-end until the next gate, blocker, or manual stop. Reads `docs/ai/workflows/*.json`, runs inline/skill/subagent steps in order, persists run state, and enforces contract-based dependencies. Do not use for free-form planning or when no workflow config exists."
---

# Orchestrator

Run a predefined workflow config until the next stop condition.

## Supported Invocations

- `/orchestrator start docs/ai/workflows/{workflow}.json`
- `/orchestrator start docs/ai/workflows/{workflow}.json --slug <feature-slug>`
- `/orchestrator start docs/ai/workflows/{workflow}.json --slug <feature-slug> --input <key>=<value>`
- `/orchestrator next --run <run-id>`
- `/orchestrator continue --run <run-id>`
- `/orchestrator continue --run <run-id> --input <key>=<value>`
- `/orchestrator next --run <run-id> --skip`
- `/orchestrator status`
- `/orchestrator status --run <run-id>`
- `/orchestrator list`
- `/orchestrator cleanup`
- `/orchestrator cleanup --force-release-lock --run <run-id>`

## Inputs

- Required on `start`: path to a workflow JSON file
- Required on `start`: feature slug for artifact naming
- Optional on `start`: runtime notes if the workflow needs them
- Required on `next` / `continue`: explicit `run_id`
- Optional on `next --skip`: explicit human request to skip the current step
- Optional on `start` / `continue`: one or more `--input <key>=<value>` overrides for the step being executed
- Optional on `status`: explicit `run_id`; without it, summarize active runs
- Optional on `cleanup --force-release-lock`: owner run id of the stale repo lock

`feature_slug` rules:

- Prefer explicit `--slug <feature-slug>` on `start`
- `feature_slug` must be kebab-case because it becomes the shared artifact stem for workflow artifacts
- If the human omits it, derive a kebab-case slug from the feature description or workflow subject, show the derived slug to the human, and record that derived value in state before running the first step

## State Files

Use one state file per run:

- Path: `docs/ai/workflows/runs/{run-id}.json`
- Keep it human-readable JSON
- Overwrite only the fields that changed
- Keep an active-run registry at `docs/ai/workflows/.orchestrator-runs.json`
- Keep the global repo lock at `docs/ai/workflows/.orchestrator-lock.json`

Minimum fields:

```json
{
  "run_id": "feature-standard--my-feature--2026-07-12T10-30-00Z",
  "workflow_id": "feature-standard",
  "workflow_path": "docs/ai/workflows/feature-standard.json",
  "feature_slug": "my-feature",
  "status": "paused",
  "updated_at": "2026-07-12T10-35-00Z",
  "current_step_id": "verify-feature",
  "holds_repo_lock": false,
  "contracts": {
    "design_path": true,
    "design_decisions_path": true,
    "spec_path": true,
    "spec_reviewed": true,
    "summary_path": true,
    "checklist_path": true
  },
  "artifact_paths": {
    "design_path": "docs/ai/designs/my-feature.json",
    "design_decisions_path": "docs/ai/design-decisions/my-feature.json",
    "spec_path": "docs/ai/specs/my-feature.md",
    "summary_path": "docs/ai/summaries/my-feature.md",
    "checklist_path": "docs/ai/checklists/my-feature.md"
  },
  "step_inputs": {
    "execute-spec": {
      "spec_path": "docs/ai/specs/my-feature.md"
    }
  },
  "history": [
    {
      "step_id": "design-spec",
      "action": "run",
      "outcome": "continue"
    },
    {
      "step_id": "spec",
      "action": "run",
      "outcome": "continue"
    },
    {
      "step_id": "review-spec",
      "action": "run",
      "outcome": "continue"
    },
    {
      "step_id": "execute-spec",
      "action": "run",
      "outcome": "continue"
    },
    {
      "step_id": "manual-checklist",
      "action": "run",
      "outcome": "continue"
    }
  ]
}
```

## Execution Model

1. Load the workflow config.
2. If `start`, initialize a new run state with empty contracts and artifact paths, then add it to the run registry.
   - resolve and persist `feature_slug` before the first step runs
3. Determine the next pending step from config order.
4. Before running a step:
   - run lightweight cleanup: remove orphan locks, archive terminal runs if policy allows, and report stale locks
   - if a repo lock is held by another run for the next step that declares `uses_repo_lock`, stop immediately and report the owner run instead of advancing
   - verify every `requires` contract exists in state
   - reject `--skip` if `skippable` is `false`
   - collect step inputs only when the step is reached; resolve `default:<value>` metadata without prompting, persist inputs under `step_inputs.<step-id>`, and reuse them when a resumable step runs again
   - merge explicit `--input` overrides into the current step's stored inputs before execution
   - if the step declares `cwd_from`, resolve that key from `artifact_paths`, require an existing absolute directory, and use it as the working directory for the complete step
   - if the step declares `uses_repo_lock`, acquire the repo lock before execution
5. Execute by `exec` type:
   - pass the step's recorded inputs, required contract values, and artifact paths explicitly to the executor
   - `inline`: perform the documented step directly in chat
   - `skill`: run the named skill in the resolved working directory
   - `subagent`: dispatch to the named subagent with the resolved working directory in its task contract
   - a skill may contain a foreground human interaction loop; keep the invocation attached until the skill emits an outcome
   - do not treat an artifact created before interactive approval as a satisfied `provides` contract
   - if an interactive skill is interrupted before emitting an outcome, leave the step pending and resume it from its deterministic artifact path on the next explicit continuation
6. Parse the last occurrence of an orchestrator HTML comment when the step is a skill or subagent.
   - Match the last `<!-- orchestrator: ... -->` block in the output
   - Do not require it to be the literal final line if later whitespace or harmless trailing text appears
   - If no orchestrator comment exists, treat outcome as `unknown`
7. Update state:
   - write outcome to history
   - add any emitted contracts to `contracts`
   - add any emitted `*_path` fields to `artifact_paths`
   - update `updated_at`
   - release any lock held by the step after the state write completes
8. Continue automatically only when:
   - the step outcome is `continue`
   - the next step has `auto: true`
   - no `human_gate`, `stop_on_outcome`, missing contract, or unknown outcome blocks progress

## Run Status Lifecycle

Allowed `status` values:

- `running`: the current invocation is actively executing this run
- `paused`: the run is waiting for human review, human input, lock availability, or an explicit resume
- `blocked`: the run hit a hard failure or invariant violation and should be treated as terminal for cleanup
- `completed`: the workflow reached its final step successfully

Use `paused` for:

- `human_gate: true`
- `unknown`
- `stop-ask-human`
- `stop-split-slices`
- `stop-run-spike`
- `stop-escalate-conflict`
- `stop-too-broad`
- `stop-drift`
- `stop-budget`
- `stop-total-budget`
- `stop-no-progress`
- repo lock held by another run

Use `blocked` for:

- `stop-blocked`
- `stop-fail`
- declared `provides` contracts missing from a `continue` outcome
- missing workflow file, run state, or required contracts/artifacts that make the next step impossible to run

## Outcome Rules

Accepted outcomes:

- `continue`
- `stop-ask-human`
- `stop-split-slices`
- `stop-run-spike`
- `stop-escalate-conflict`
- `stop-blocked`
- `stop-too-broad`
- `stop-fail`
- `stop-drift`
- `stop-budget`
- `stop-total-budget`
- `stop-no-progress`
- `unknown`

Rules:

- If a skill/subagent output contains no orchestrator comment, treat outcome as `unknown`
- Do not infer paths or outcomes heuristically from prose
- If a step ends with `continue` but does not emit every declared `provides` contract, stop and mark the run blocked
- Every paused stop outcome keeps the current step pending because it did not satisfy the step's declared `provides`
- `continue` reruns that same pending step with its stored inputs and any explicit overrides
- `stop-budget`, `stop-total-budget`, and `stop-no-progress` additionally preserve their implementation workspace and budget state
- `skip` never satisfies `requires`

## Workflow Rules

- Do not insert or run skills that are absent from the selected workflow config
- `requires` means contract or artifact presence in state, not "a previous step once ran"
- `provides` means contracts that must be recorded when the step succeeds with `continue`
- a human interaction performed inside a skill is complete only when that skill emits `continue`; do not add a second inferred human gate
- `cwd_from` names an `artifact_paths` key whose absolute directory becomes the working directory for that step
- `uses_repo_lock` means the step needs the single shared repo lock; if another run already holds it, the current invocation must stop immediately
- If a step hits `human_gate: true`, stop immediately after writing state
- If a step outcome matches `stop_on_outcome`, stop immediately after writing state
- If the workflow path or run state is missing, stop and report the missing file
- `status` must read and summarize state without advancing, skipping, or mutating the workflow
- `list` must show every non-archived run with run id, feature slug, current step, status, and whether the run holds the repo lock
- Only `start` / `next` / `continue` are blocked by another run's repo lock; `status`, `list`, and `cleanup` remain available

## Cleanup Rules

- `cleanup` archives terminal runs into `docs/ai/workflows/runs/archive/`
- terminal means `completed` or `blocked`
- `cleanup` may also archive stale paused runs after the configured TTL
- `cleanup` removes orphan locks automatically when the owner run is missing, archived, or no longer on the locked step
- `cleanup` does not auto-release a stale lock whose owner still appears `running`; require explicit `cleanup --force-release-lock --run <run-id>`
- Keep run state human-readable after archival; do not silently delete the only copy

## Output

After each invocation, return:

- current run status
- current or next step id
- why execution stopped or continued
- run state path
- any artifact paths newly recorded this turn
- any lock owner that blocked progress
- the recorded `checklist_path` as the primary human validation artifact whenever it exists

For `status --run <run-id>`, return:

- workflow id
- feature slug
- current step id
- current run status
- last stop reason if present
- recorded contracts
- recorded artifact paths

If the run completed or stopped during verification and `checklist_path` exists, show that path prominently even when it was recorded by an earlier step.

For `status` without `--run`, return:

- active run ids
- feature slugs
- current steps
- statuses
- repo lock holder if any

For `list`, return:

- active run ids
- feature slugs
- current steps
- statuses
- repo lock holder if any

For `cleanup`, return:

- archived run ids
- removed orphan locks
- stale locks still requiring human force-release

## Done When

- `start` created a run state and either advanced until a stop condition or reported why it could not start
- `next` / `continue` advanced exactly until the next stop condition
- global repo lock, run registry, and per-run state remain consistent with the workflow config
