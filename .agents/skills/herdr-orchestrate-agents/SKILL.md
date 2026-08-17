---
name: herdr-orchestrate-agents
description: Assign stored or raw tasks to existing Codex, Claude Code, OpenCode, and other agent sessions in Herdr, synchronize task outcomes, inspect assigned-task progress, send task follow-ups, or route an explicitly selected GNHF workflow. Use for task-to-session coordination. Do not use for standalone backlog CRUD or task boards, behavioral root-cause audit of a session, general agent status overviews, or direct creation and control of Herdr resources.
---

# Herdr Task Orchestrator

Use `~/.ai-workflow/tasks.json` as the task source of truth and Herdr as the execution transport.
Organize coordination around task IDs instead of asking the user to remember which session owns each task.

## Operating Boundary

Before dispatch, sync, inspect, or follow-up, verify:

```bash
test "${HERDR_ENV:-}" = 1
```

If the check fails, stop and tell the user to open the orchestrator inside Herdr.
A prompt preview may run without Herdr because it does not contact an agent or mutate the store.

Operate against the current Herdr session only.
Never create, start, close, stop, focus, attach, move, or rename a workspace, tab, pane, session, or coding agent.
Never provide a general overview of all agent statuses.
Show live agent state only when it explains an assigned task.

Use the neighboring skills for other intents:

- Use `task-manager` for standalone task capture, lists, boards, manual transitions, deletion, cleanup, or reset.
- Use `herdr-audit-session` to diagnose why one exact session behaved unexpectedly without contacting the agent.
- Use `herdr-guide` for direct pane, terminal, workspace, or session control.

## Output Language

Return user-facing output in Vietnamese unless the user explicitly requests another language.
Preserve commands, paths, IDs, agent names, workspace labels, code, and Herdr status values exactly.
Write any task title created as part of dispatch in Vietnamese.

## Route The Request

Choose exactly one primary workflow:

- **Direct dispatch**: assign a stored task, or capture and assign a raw request in one operation.
- **GNHF dispatch or resume**: route an approved spec task through `feature-implement-gnhf` only when the user explicitly selects it.
- **Sync tasks**: reconcile assigned tasks with their agent sessions.
- **Inspect task**: answer a progress, result, or blocker question for one assigned task.
- **Follow-up**: send user-authorized clarification to the session already assigned to a task.
- **Prompt preview**: prepare the direct task prompt without sending or mutating the store.

For direct dispatch, sync, inspect, follow-up, prompt preview, task-store access, and live target resolution, read [Task Session Routing](references/task-session-routing.md) before acting.
For GNHF dispatch, resume, sync, inspect, or follow-up, read [Task Session Routing](references/task-session-routing.md) and [GNHF Router](references/gnhf-router.md) before acting.

When a raw request is dispatched, create its task only as part of the same dispatch workflow.
Do not support standalone capture or board requests here; route them to `task-manager`.

## Router Selection

Treat absence of task `workflow` metadata as the direct router.
Select `feature-implement-gnhf` only when the user explicitly asks for GNHF, the GNHF loop, or that workflow by name.
Never select GNHF automatically from task size or complexity.

## Coordination Safety

- Refresh live state immediately before every send because pane IDs can change.
- Never use the focused pane, recency, or sidebar order as an implicit target.
- Never send input to an ambiguous target.
- Allow cross-workspace dispatch only when the user explicitly names the target workspace or pane.
- For cross-workspace dispatch, treat the target workspace as the execution location while preserving the task's original `project_id`.
- After an explicit cross-workspace dispatch, sync, inspect, and follow-up must resolve the live target from stored assignment metadata first rather than from the task project's root.
- Never broadcast a task to multiple agents unless the user explicitly requests duplicate execution.
- Refuse implicit parallel dispatch in one project because agents may edit the same files.
- If a working agent is targeted, send only when the user explicitly requests immediate delivery.
- Never approve permissions or answer product decisions on the user's behalf.
- Treat Herdr status as operational evidence, not proof of task correctness.
- Preserve unknown task-store fields during every write.
- Do not mark a task done from `idle` or `done` status alone.
- Do not expose raw store JSON or full transcripts unless the user asks.
