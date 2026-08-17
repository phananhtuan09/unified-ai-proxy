# GNHF Task Router

Read this reference only when the user explicitly selects the `feature-implement-gnhf` router for a task.
Read [Task Session Routing](task-session-routing.md) first for the shared store and target-resolution rules.
The normal direct-send router does not use these rules.

## Contents

- [Eligibility](#eligibility)
- [Initial Dispatch](#initial-dispatch)
- [Synchronization](#synchronization)
- [Resume A Paused GNHF Run](#resume-a-paused-gnhf-run)
- [Follow-Up And Human Handoff](#follow-up-and-human-handoff)

## Eligibility

Before dispatching, require all conditions:

1. The task resolves uniquely and is not `done`.
2. The user explicitly selected GNHF for this task.
3. The task or current request supplies a spec under `docs/ai/specs/`.
4. The user has approved that spec for implementation.
5. The spec exists inside the task project, is tracked by Git, and has no staged or unstaged changes.
6. The repository working tree is otherwise clean.
7. The repository is not currently on a `gnhf/*` branch.
8. `docs/ai/workflows/feature-implement-gnhf.json` exists in the project.
9. The feature slug equals the spec filename without `.md`.
10. Any requested GNHF agent, budget, or runtime customization is explicit enough to forward without interpretation risk.

Do not create, commit, stash, reset, or clean files to satisfy eligibility.
If any condition fails, leave the task `todo` and report the exact blocker.

## Initial Dispatch

Resolve an eligible idle Herdr agent using the shared target rules in [Task Session Routing](task-session-routing.md).
Send this prompt directly with `herdr pane run`:

```text
TASK_ID: <task-id>
ROUTER: feature-implement-gnhf

Run the repository orchestrator for the approved spec:

/orchestrator start docs/ai/workflows/feature-implement-gnhf.json --slug <feature-slug> --input spec_path=<spec-path> <explicit-input-overrides>

Do not replace this workflow with normal direct implementation.
Follow the workflow until it completes or reaches a configured stop.

After the orchestrator stops, report:

HERDR_TASK_RESULT
Task: <task-id>
Status: workflow-paused | workflow-blocked | awaiting-human
Workflow run:
Run state:
Current step:
Stop reason:
Implementation workspace:
Summary:
Verification:
Blocker:
Next:
```

Never route this prompt through a file.
Omit `<explicit-input-overrides>` when the user accepts defaults.
When requested, forward supported values such as `agent`, `iterations_per_epoch`, `tokens_per_epoch`, `max_epochs`, `max_total_tokens`, `max_no_progress_epochs`, `auto_continue`, or `runtime_request` as separate `--input <key>=<value>` arguments.
Preserve a natural-language runtime request as one quoted input value; the executing skill must resolve it through the installed `gnhf --help` output.
After confirmed submission, set task `status` to `doing`, store the normal Herdr assignment, and write:

```jsonc
{
  "execution": {
    "state": "working"
  },
  "workflow": {
    "router": "feature-implement-gnhf",
    "workflow_path": "docs/ai/workflows/feature-implement-gnhf.json",
    "spec_path": "<spec-path>",
    "feature_slug": "<feature-slug>",
    "status": "starting",
    "updated_at": "<ISO>"
  }
}
```

Do not mark the task assigned if prompt submission cannot be confirmed.

## Synchronization

Prefer the recorded orchestrator run state over transcript inference.
When `workflow.run_state_path` exists, read it from the original project workspace.
Otherwise, read at most 120 recent unwrapped lines and extract the latest task result block to obtain `run_id` and `run_state_path`.

Copy these values when available:

- orchestrator `run_id`
- run state path
- current step id
- workflow status
- `implementation_workspace_path` from run artifact paths
- `gnhf_budget_path` from run artifact paths
- last outcome and exact stop reason
- epochs completed, cumulative tokens, total limits, and estimated-token status from the GNHF budget ledger when available

Map orchestrator state to the task:

| Orchestrator state | Task mapping |
| --- | --- |
| `running` | Keep task `doing`; set workflow `running` and execution `working`. |
| `paused` with `stop-budget` | Keep task `doing`; set workflow `paused` and execution `blocked`; explain that automatic continuation is disabled and the run is resumable. |
| `paused` with `stop-total-budget` | Keep task `doing`; set workflow `paused` and execution `blocked`; report consumed epoch/token budget and require an explicit increase to the binding limit. |
| `paused` with `stop-no-progress` | Keep task `doing`; set workflow `paused` and execution `blocked`; report the last verified progress and require a compatible runtime or budget-policy change, otherwise start a new narrower run. |
| Other `paused` state | Keep task `doing`; set workflow `paused` and execution `blocked`; preserve the exact human input needed. |
| `blocked` | Keep task `doing`; set workflow `blocked` and execution `blocked`; preserve the hard failure. |
| `completed` | Keep task `doing`; set workflow and execution to `awaiting_human`; preserve workspace and verification evidence. |

Never map workflow `completed` directly to task `done`.
The implementation remains an unmerged candidate until the human accepts it.

## Resume A Paused GNHF Run

Require workflow status `paused`, a resumable outcome, and an existing `run_id`.

For `stop-budget`, continue with the stored policy unless the user supplies overrides.
For `stop-total-budget`, require the user to increase whichever of `max_epochs` or `max_total_tokens` is currently binding before continuing.
For `stop-no-progress`, require a user-approved runtime or budget-policy change that does not alter the approved spec or worker prompt.
Changing the agent, approved scope, or worker prompt requires a new workflow run because resume must preserve the original GNHF run identity.

Send the stored policy overrides or human-approved changes directly to the assigned agent:

```text
TASK_ID: <task-id> WORKFLOW_RESUME

/orchestrator continue --run <run-id> <explicit-input-overrides>

Continue the same GNHF run and report the updated HERDR_TASK_RESULT block when the orchestrator stops again.
```

Set workflow status to `running` and execution state to `working` only after confirmed submission.
Do not create a new orchestrator run for a budget continuation.

## Follow-Up And Human Handoff

For a paused product question or drift decision, send the user's answer to the assigned agent with the task ID, then instruct it to continue the same orchestrator run.
Never answer the decision on the user's behalf.

When workflow state becomes `awaiting_human`, show:

- preserved implementation workspace
- branch or commit range when reported
- summary and verification paths
- runtime verification result
- total epochs and cumulative tokens consumed
- next human action such as review, merge, cherry-pick, or reject

Only an explicit human completion action moves the task to `done`.
Route requeue through `task-manager`; it removes workflow and execution metadata but performs no implicit worktree cleanup.
