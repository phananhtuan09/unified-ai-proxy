# Task Session Routing

Use these rules for direct dispatch, prompt preview, sync, inspect, follow-up, and the shared task or target resolution required by GNHF routing.

## Contents

- [Task Store Contract](#task-store-contract)
- [Resolve The Live Target](#resolve-the-live-target)
- [Capture Only For Immediate Dispatch](#capture-only-for-immediate-dispatch)
- [Preview Or Dispatch A Direct Task](#preview-or-dispatch-a-direct-task)
- [Sync Assigned Tasks](#sync-assigned-tasks)
- [Inspect One Task](#inspect-one-task)
- [Send A Follow-Up](#send-a-follow-up)
- [Setup Failures](#setup-failures)

## Task Store Contract

Read and write `~/.ai-workflow/tasks.json`, shared with `task-manager`.
Accept version 1 data and write version 2 on the next mutation.
Preserve every unknown field.

Use this compatible shape:

```jsonc
{
  "version": 2,
  "projects": [
    {
      "id": "p_<random6>",
      "name": "folder-name",
      "root": "/absolute/path"
    }
  ],
  "tasks": [
    {
      "id": "t_<random6>",
      "project_id": "p_xxx",
      "title": "ý định ngắn gọn của người dùng",
      "details": "optional durable context and constraints",
      "status": "todo | doing | done",
      "assignment": {
        "workspace_root": "/absolute/path",
        "workspace_label": "project-label",
        "project_root": "/absolute/path of the task's owning project",
        "agent_name": "optional unique name",
        "agent_type": "codex",
        "terminal_id": "term_...",
        "assigned_at": "ISO"
      },
      "execution": {
        "state": "working | blocked | awaiting_human | completed | unknown",
        "summary": "optional",
        "files_changed": "optional",
        "verification": "optional",
        "blocker": "optional",
        "next": "optional",
        "synced_at": "ISO"
      },
      "workflow": "optional router-owned metadata",
      "created": "ISO",
      "updated": "ISO"
    }
  ]
}
```

Do not persist a pane ID because pane IDs can change after moves or restores.
Match a project by normalized absolute root first.
Use a project-name match only as a legacy fallback when it resolves to the same root.
Register distinct projects when different roots share a folder name.
Preserve the task's owning project in `project_id`.
When a task is dispatched to a different workspace by explicit user choice, preserve the original `project_id` and store both the execution workspace in `assignment.workspace_root` and the owning project root in `assignment.project_root`.

Always read the complete store immediately before mutation.
Write the complete store atomically through a temporary file and rename.
If JSON parsing fails, do not overwrite the store.
Report corruption and point the user to `task-manager` recovery.

## Resolve The Live Target

Read only the live state needed for the requested task:

```bash
herdr workspace list
herdr agent list
```

By default, use agent `cwd` or `foreground_cwd` to match the task project's absolute root.
For a task with stored `assignment.workspace_root`, resolve that workspace first.
For a new dispatch where the user explicitly names a workspace or pane outside the task project, resolve the named target directly and treat it as authoritative for dispatch.
Run `herdr pane list` only when no matching agent exists and you must distinguish an empty project workspace from a missing workspace.

Resolve an existing assignment in this order:

1. Unique `agent_name` in the project workspace.
2. Exact `terminal_id` while that terminal is live.
3. Unique `agent_type` in the project workspace.

When `assignment.workspace_root` differs from `assignment.project_root` or the task project's root, use the assignment workspace for these checks instead of the project workspace.

For new dispatch, prefer a user-named target.
If the user did not name a target, auto-select only when exactly one eligible idle agent exists in the project.
If multiple targets remain, show only those candidates and ask the user to choose.
Treat all IDs as opaque.

## Capture Only For Immediate Dispatch

When the user supplies a raw request for immediate assignment:

1. Resolve the project from an explicit project, a matching Herdr workspace root, or the current working directory.
2. Reject dispatch when another task is already `doing` in that project unless the user explicitly confirms parallel work.
3. Create a concise Vietnamese title that preserves the user's intent.
4. Store durable requirements, context, constraints, and requested verification in `details` only when they exist.
5. Generate `t_` plus 6 random lowercase hexadecimal characters.
6. Create the task as `todo` with ISO timestamps.
7. Continue immediately to prompt construction and dispatch.

Do not use this path for standalone capture.
If submission cannot be confirmed, keep the newly captured task as `todo` without assignment metadata.

## Preview Or Dispatch A Direct Task

1. Determine whether the request is preview-only or an actual dispatch.
2. For preview, resolve a stored task read-only or construct a raw-request candidate in memory without creating a task.
3. For dispatch, resolve the task by exact ID or unique title, or capture a raw request using the immediate-dispatch rules.
4. Confirm the task has no `workflow` metadata.
5. Reject a `done` task unless the user reopens it through `task-manager` first.
6. Reject dispatch when another task is already `doing` in the same project unless the user explicitly confirms parallel work.
7. Resolve the project workspace and target agent from fresh Herdr state for dispatch only.
If the user explicitly names a different workspace or pane, allow dispatch to that target instead of the task project's workspace.
8. Inspect at most 60 recent unwrapped lines only when needed to avoid conflicting with the target's current task.
9. Preserve task intent and add only context supported by `details` or the confirmed current conversation.
10. If the target is `blocked`, send only a user-authorized answer or clarification.

Build the prompt with this shape and omit empty sections:

```text
TASK_ID: <task-id or generated-on-dispatch>

Mục tiêu
<task title>

Bối cảnh
<stored details or confirmed conversation context>

Yêu cầu
<confirmed requirements>

Ràng buộc
<confirmed constraints>

Xác minh
<requested checks>

Báo cáo kết quả
Khi dừng vì hoàn tất hoặc bị chặn, hãy kết thúc câu trả lời bằng đúng block sau và viết giá trị bằng tiếng Việt:

HERDR_TASK_RESULT
Task: <task-id>
Status: completed | blocked
Summary:
Files changed:
Verification:
Blocker:
Next:
```

Do not invent requirements, file paths, acceptance criteria, or permission decisions.
Keep a simple task prompt simple.

For raw-request preview, keep `generated-on-dispatch` as a visible placeholder.
Return the preview and stop without resolving a live pane, sending input, or mutating the store.

For dispatch, refresh live state and send directly:

```bash
herdr pane run <pane-id> "<task-prompt>"
```

Never route the prompt through a file or ask the target to read a path outside its workspace.
After sending, wait briefly for `working`.
If the wait times out, inspect the pane and confirm that the task ID was submitted or processed.
Only after submission is confirmed, update the task to `doing`, store the assignment, set `assignment.project_root` to the task project's root, set `execution.state` to `working`, and write the store.
If submission cannot be confirmed, leave assignment and execution unchanged.

Return the task ID, project, resolved agent, and exact prompt sent.
Do not wait for task completion unless requested.

## Sync Assigned Tasks

Sync one task, one project, or every task with `status: doing`.

1. Read the store and live Herdr state once.
2. Resolve each assignment without relying on a stored pane ID.
Prefer `assignment.workspace_root` when present.
3. Apply the live-state rules below.

| Live state | Action |
| --- | --- |
| `working` | Set `execution.state` to `working`; do not read the transcript. |
| `blocked` | Read at most 80 recent unwrapped lines, extract the exact blocker, and set `execution.state` to `blocked`. |
| `idle` or `done` | Read at most 120 recent unwrapped lines and find the latest `HERDR_TASK_RESULT` block for the task ID. |
| `unknown` or missing | Keep the task `doing`, set `execution.state` to `unknown`, and report that the assignment cannot be resolved safely. |

For `Status: completed`, copy the outcome into `execution`, set the direct-router task to `done`, and update timestamps.
For `Status: blocked`, keep the task `doing` and set `execution.state` to `blocked`.
If no trustworthy result block exists, use bounded transcript evidence but do not mark the task done unless completion is explicit.

Write the store once after processing all selected tasks.
Return only task changes, blockers, unresolved assignments, and the next useful action.

## Inspect One Task

Resolve the task first, then use its stored assignment.

- If the task is `todo`, report that it has not been assigned.
- If the task is `done`, return the stored outcome without contacting Herdr unless the user asks to refresh evidence.
- If the task is `doing` without workflow metadata, run direct sync for that task.
- If the task has workflow metadata, apply `gnhf-router.md` instead of direct result mapping.

For explicit inspect only, when an assigned agent is `idle` or `done` and no result block exists, send one bounded task-specific report request.
The request must prohibit new implementation work and ask only for the `HERDR_TASK_RESULT` block for that task ID.
Do not send this checkpoint during bulk sync.

Return:

```text
Task: <id> "<title>"
Project: <project>
Agent: <resolved target or "chưa giao">
Trạng thái: <todo | doing | blocked | awaiting_human | done | unknown>
Kết quả: <stored or synchronized summary>
Xác minh: <reported checks>
Trở ngại: <exact blocker or "không có">
Tiếp theo: <one concise recommendation>
```

Distinguish agent self-report from independent verification.

## Send A Follow-Up

1. Resolve the task and assignment.
2. Route workflow resume or paused-decision handling through `gnhf-router.md` when workflow metadata exists.
3. Refresh the target from live Herdr state.
4. Preserve the user's clarification exactly and prefix it with `TASK_ID: <task-id> FOLLOW-UP`.
5. Send directly with `herdr pane run` and never through a payload file.
6. Keep the existing assignment and update only the task timestamp after confirmed submission.

If the assignment is stale or ambiguous, do not guess.
Report the missing target and ask the user to choose a live agent for reassignment.

## Setup Failures

If the project workspace is missing and there is no explicit cross-workspace target or stored `assignment.workspace_root`:

```text
Không tìm thấy workspace cho project của task `<task-id>` trong Herdr session hiện tại.
Hãy mở project đó trong cùng Herdr session rồi yêu cầu tôi dispatch lại task.
```

If no eligible agent is running:

```text
Project của task `<task-id>` chưa có agent phù hợp đang chạy.
Hãy mở agent trong workspace đó, nên đặt tên duy nhất, rồi yêu cầu tôi dispatch lại task.
```

Do not create the missing workspace or agent automatically.
