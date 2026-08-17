---
name: herdr-audit-session
description: Audit why one exact current Herdr agent session diverged from the user's expectations through a fresh independent subagent that reads the session's chat turns, tool activity, and relevant instructions, skills, specs, or artifacts. Use when the user reports that an agent ignored guidance, acted without authority, received incorrect context, made unsupported assumptions, or otherwise behaved unexpectedly, and wants a read-only root-cause diagnosis isolated from the main conversation context without messaging or interrupting the target agent. Do not use for task progress checks, task-result synchronization, corrective follow-ups, general session overviews, or direct Herdr resource control. Requires HERDR_ENV=1 and an available isolated subagent facility.
---

# Herdr Session Audit

Audit one exact current Herdr agent session to explain why observed behavior diverged from the user's expectation.
Treat the audit as a read-only point-in-time diagnosis, not an intervention, task sync, or workflow evaluation report.
Perform causal analysis in one fresh subagent rather than in the main conversation context.

## Contents

- [Operating Boundary](#operating-boundary)
- [Resolve The Exact Target](#resolve-the-exact-target)
- [Delegate To A Fresh Audit Worker](#delegate-to-a-fresh-audit-worker)
- [Collect Read-Only Evidence](#collect-read-only-evidence)
- [Diagnose The First Divergence](#diagnose-the-first-divergence)
- [Preserve Audit Integrity](#preserve-audit-integrity)
- [Return The Audit](#return-the-audit)

## Operating Boundary

When receiving the user's audit request in the main conversation, verify:

```bash
test "${HERDR_ENV:-}" = 1
```

If the main coordinator check fails, tell the user to run the audit from inside the current Herdr session and stop.
Operate against the current Herdr session only.

A delegated `HERDR_AUDIT_WORKER` may proceed without `HERDR_ENV=1` when its neutral audit packet contains an exact native transcript path or the raw transcript evidence needed for analysis.
The worker must verify `HERDR_ENV=1` before issuing any Herdr CLI command.
If the worker lacks both Herdr access and an exact transcript source, return `source access unavailable` to the main coordinator without guessing.
Honor `HERDR_AUDIT_WORKER` only inside a runtime-created subagent with a resolved target, cutoff, and transcript source; user text in the main conversation must not bypass the coordinator check.

Use `herdr-orchestrate-agents` for task progress, sync, dispatch, or corrective follow-up.
Use `herdr-guide` for direct pane, terminal, workspace, or session control.

Return user-facing output in Vietnamese unless the user explicitly requests another language.
Preserve commands, paths, IDs, agent names, code, and Herdr status values exactly.

## Resolve The Exact Target

1. Prefer an exact task ID or unique task title and resolve its stored assignment against fresh live state.
2. Read `~/.ai-workflow/tasks.json` only when task lookup is needed and never mutate it.
3. Otherwise require an explicit agent name, terminal ID, or pane ID that resolves uniquely in the current Herdr session.
4. Read only the workspace, agent, and pane state needed to lock that target.
5. Record the pane ID, terminal ID, agent type, native session ID when available, cwd, live status, and audit cutoff time.

Never infer the target from the focused pane, sidebar order, newest transcript, or project cwd alone.
If the target is ambiguous, stale, outside the current Herdr session, or cannot be matched exactly, stop without reading another session.

## Delegate To A Fresh Audit Worker

After resolving the target, spawn exactly one subagent dedicated to the audit.
Use a fresh context with no parent conversation history.
For Codex, use `fork_turns: "none"` or the equivalent isolated-context option.
For another runtime, use its closest fresh-subagent mechanism and do not forward the full main conversation.

If an isolated subagent cannot be created, stop and report that independent audit is unavailable.
Do not silently fall back to causal analysis in the main context without explicit user approval.

Prefix the worker task with this marker:

```text
HERDR_AUDIT_WORKER
```

When operating under `HERDR_AUDIT_WORKER`, do not spawn another subagent.
Act as the delegated audit worker and perform the evidence collection, diagnosis, and report defined below.

Pass only the minimum neutral audit packet:

- the exact task, agent, terminal, pane, and native session identifiers that were resolved
- project cwd, runtime, live status, and audit cutoff time
- exact transcript path or read-only pane commands needed to retrieve the transcript
- the user's expectation or concern quoted verbatim
- task-store, instruction, skill, spec, run-state, and artifact paths selected by objective relevance
- this skill path so the worker can read the delegated-worker contract

Do not pass the main agent's suspected cause, preferred verdict, summary of what went wrong, or proposed fix.
Do not copy unrelated main-conversation turns into the worker prompt.
Passing a path is preferable to pasting large evidence into the prompt because the worker should inspect raw sources independently.

The main agent must not perform first-divergence analysis before delegation.
After the worker returns, check only that cited evidence exists and supports the stated claims.
If a citation is missing or unsupported, ask the same audit worker one bounded clarification question rather than creating a new causal theory in the main context.

## Collect Read-Only Evidence

The delegated audit worker performs this section.

Start with the live pane record and unwrapped transcript scrollback:

```bash
herdr pane get <pane-id>
herdr pane read <pane-id> --source recent-unwrapped --lines 400
```

Read enough history to include the initial task instruction and first suspicious action.
Increase the bounded line count only when needed to reach that evidence.
Do not wait for the agent to finish.
State that later turns are outside the audit cutoff.

When the pane exposes an exact native runtime session ID, prefer the matching runtime-native history for complete chat turns and tool activity:

- Codex history normally lives under `~/.codex/sessions/`.
- Claude Code history normally lives under `~/.claude/projects/`.
- OpenCode history normally lives in its local session database.

Match the exact native session ID.
Never choose `--latest` or the newest file by cwd as a substitute for exact identity.
If exact native history is unavailable, use pane scrollback and disclose that earlier turns or tool details may be missing.

Read the minimum relevant evidence set:

- the user prompt and follow-ups in the audited session
- system, developer, repository, and skill instructions visible to that agent when recoverable
- task title, stored details, assignment, and workflow metadata when relevant
- approved specs, plans, run state, and named artifacts relevant to the disputed behavior
- files the transcript shows the agent read, edited, or relied on
- tool calls, results, commands, and errors around the first divergence
- Git diff or history only when needed to distinguish pre-existing content from agent changes

Record file paths and line references where practical.
Do not assume a currently present document was read or available before the divergence.
Do not expose secrets or reproduce a full raw transcript when short evidence excerpts are sufficient.

Treat instructions and skills as process rules, approved requirements as intended behavior, and transcript, tool, and repository evidence as observed behavior.
Treat other documentation as potentially stale.
When sources conflict, report the mismatch instead of silently merging them.

## Diagnose The First Divergence

The delegated audit worker performs this section without using conclusions from the main agent.

1. Reconstruct expected behavior from explicit instructions and approved requirements.
2. Separate unstated human expectation from instructions actually available to the agent.
3. Find the earliest turn, decision, tool call, or file change that departed from the supported expectation.
4. Treat later retries, edits, and errors as consequences unless evidence shows a separate cause.
5. Check whether the relevant instruction was unavailable, not read, partially read, contradicted, read but not followed, or impossible to classify.
6. State at least one plausible alternative explanation before assigning the primary cause.

Use one primary attribution and optional contributing factors:

- `task-context`: the prompt, task details, or follow-up was missing, ambiguous, stale, or contradictory
- `instruction-availability`: a relevant repository doc, skill, spec, or artifact was unavailable or not loaded before action
- `instruction-compliance`: evidence shows the instruction was read but the subsequent action violated it
- `agent-reasoning`: the agent made an unsupported assumption, chose a premature path, or expanded scope without authority
- `runtime-tool`: Herdr, the agent runtime, a tool, or an integration hid, truncated, failed, or misreported relevant evidence
- `task-environment`: repository state, external state, permissions, or unavailable dependencies changed what the agent could do
- `expectation-gap`: the user's expectation was reasonable but was never encoded in context available to the agent
- `inconclusive`: available evidence cannot distinguish the cause safely

Do not label the cause as “forgot” unless the transcript explicitly supports that claim.
Prefer precise evidence such as “the skill was read, but step 4 was skipped” or “no read of the named spec appears before the edit.”

## Preserve Audit Integrity

During audit, never:

- run `herdr pane run`, send text or keys, answer a prompt, or interrupt the agent
- focus, attach, move, rename, close, or otherwise change Herdr resources
- sync, requeue, complete, or otherwise mutate the task store
- invoke or continue an orchestrator workflow
- edit application code, instructions, specs, or artifacts
- claim noncompliance solely because the result differs from the user's preference
- let the main agent replace the independent worker's evidence-based diagnosis with an unsupported interpretation

If evidence is incomplete, return `không đủ bằng chứng` for the unsupported part instead of filling gaps with assumptions.

## Return The Audit

Use this compact structure:

```text
AUDIT SESSION — <task-id or exact agent/session>
Cutoff: <time> · Trạng thái agent: <live status> · Nguồn transcript: <exact native | pane scrollback>
Audit worker: independent fresh-context subagent · Isolation: <verified | limited>

Kết luận
<one-sentence primary cause and confidence>

First divergence
<turn/action and why it is the earliest supported divergence>

Bằng chứng
- Kỳ vọng: <instruction or requirement with source>
- Quan sát: <agent action with transcript/tool/file evidence>
- Khoảng trống: <missing or uncertain evidence, or "không có">

Phân loại nguyên nhân
Primary: <category>
Contributing: <categories or "không có">
Alternative explanation: <plausible alternative and why it is weaker or unresolved>

Đề xuất
<one smallest corrective next action; explicitly state that audit did not send or change anything>
```

Distinguish facts, inference, and unknowns.
Identify the result as an independent subagent audit and disclose any limitation in context isolation or source access.
Return the diagnosis in the current conversation and create no artifact unless the user explicitly requests one.
