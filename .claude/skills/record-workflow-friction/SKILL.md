---
name: record-workflow-friction
description: Record any observable problem, blocker, ambiguity, retry, rework, workaround, human correction, tool failure, runtime issue, task-context issue, or workflow friction encountered while an AI agent is running. Use only when the human explicitly asks to note, capture, or log the current execution difficulty for later analysis. Do not assume the workflow caused the problem, and do not fix or replace the active task.
---

# Record Workflow Friction

Capture one concise, durable observation about agent execution without interrupting or evaluating the active task.

## Scope

Record friction even when it is not clearly a workflow problem. The observation may concern:

- workflow instructions, gates, artifacts, or handoffs
- model understanding, decisions, or execution
- runtime, tool, command, or integration behavior
- environment or external state
- task ambiguity or missing context
- human-agent coordination
- unknown cause

The workflow name and phase may be `unknown`. Do not force a workflow attribution.

## Boundaries

- Run only after an explicit human request.
- Record observable facts separately from hypotheses.
- Do not expose chain-of-thought, secrets, credentials, or unnecessary transcript content.
- Do not diagnose, fix, or redesign the workflow or active task.
- Do not label the issue as workflow-caused without evaluation evidence.
- After recording, return to the active task unless the human asks to stop.

## Procedure

1. Identify the active task, runtime, session, workflow or skill when known, and current execution context. Use `unknown` when unavailable.
2. Describe expected behavior, observed friction, first visible divergence, practical impact, extra work, human intervention, and current state.
3. Choose an `observation_scope` without claiming root cause: `workflow`, `model-execution`, `runtime-tool`, `task-environment`, `coordination`, or `unknown`.
4. Add direct evidence references such as a session, command result, tool result, file, artifact, or human correction. Do not invent evidence.
5. Write one Markdown artifact to:

```text
docs/ai/agent-observations/YYYY-MM-DD-{subject}-{short-slug}.md
```

If the path exists, add a numeric suffix. Keep one observation per file.

## Artifact Format

```md
---
phase: agent-execution-observation
schema_version: ai-agent/observation-v2
status: agent-reported-observation
recorded_at: YYYY-MM-DDTHH:MM:SSZ
subject: task-or-workflow-name
observation_scope: unknown
workflow_name: unknown
workflow_version: unknown
workflow_phase: unknown
runtime: claude
model: unknown
session_reference: unknown
task_class: unknown
category: rework
outcome_state: in-progress
human_intervention: unknown
---

# Agent Execution Observation: Short title

## Task Context
What the agent was trying to accomplish when the friction occurred.

## Expected Behavior
The observable behavior expected from the agent, workflow, tool, or environment.

## Observed Friction
What actually happened. Keep this factual and concise.

## First Visible Divergence
The earliest observable point where execution began differing from the expected behavior, or `unknown`.

## Impact
How the friction affected outcome, progress, quality, safety, cost, or human effort.

## Extra Work or Recovery
Visible retries, repeated reads or commands, rework, recovery, workaround, or coordination cost. Use `unknown` when it cannot be measured.

## Human Intervention
What the human had to clarify, correct, approve, repair, or redirect, or `none observed`.

## Workaround or Current State
The workaround used and whether the task recovered, remains blocked, or completed with friction.

## Evidence References
- `path-command-session-or-tool-reference`, or `unavailable`

## Agent Attribution Hint (Unverified)
Optional suspected source. Keep it separate from facts and do not present it as root cause.
```

## Categories

Use a short category such as:

- `misunderstanding`
- `missing-clarification`
- `unnecessary-question`
- `premature-action`
- `assumption`
- `decision-error`
- `repeated-read`
- `repeated-command`
- `retry-loop`
- `rework`
- `human-correction`
- `instruction-conflict`
- `unused-artifact`
- `tool-failure`
- `runtime-gap`
- `environment`
- `blocker`
- `safety`
- `other`

## Evidence Rule

The artifact is discovery evidence with status `agent-reported-observation`. It does not prove a workflow failure or root cause. `workflow-evaluation` must corroborate it with session traces, tool or artifact evidence, human correction, repeated observations, or controlled exercise before upgrading its evidence status.

Observation counts do not establish a failure rate without a known session denominator.

## Done When

- one observation exists in `docs/ai/agent-observations/`
- facts, impact, and unverified attribution are separated
- first divergence and extra work are recorded or marked `unknown`
- missing evidence is marked `unavailable`
- the active task was not modified by this skill
