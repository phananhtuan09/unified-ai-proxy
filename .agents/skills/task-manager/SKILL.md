---
name: task-manager
description: Manage the centralized local task backlog before and around implementation workflows. Use when the user wants to add, list, view a stored-state task board, start, requeue, complete, delete, clean, or recover tasks without contacting Herdr agents. The store is shared with herdr-orchestrate-agents, so task views preserve and display assignment, execution, and workflow-router metadata when present. Do not use for live task dispatch, session synchronization, behavioral session audit, or direct Herdr resource control.
---

# Task Manager

Manage human-intent tasks in `~/.ai-workflow/tasks.json`.
This skill owns standalone backlog actions.
Use `herdr-orchestrate-agents` when the user wants to assign tasks to agent sessions or synchronize agent outcomes.
Use `herdr-audit-session` when the user wants a read-only diagnosis of one exact agent session.

## Store Contract

Accept version 1 and version 2 stores.
Write version 2 on the next mutation and preserve unknown fields.

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
      "title": "ý định của người dùng",
      "details": "optional",
      "status": "todo | doing | done",
      "assignment": "optional orchestration metadata",
      "execution": "optional orchestration metadata",
      "workflow": "optional workflow-router metadata",
      "created": "ISO",
      "updated": "ISO"
    }
  ]
}
```

The `assignment`, `execution`, and `workflow` objects are owned by orchestration.
Always preserve them unless an explicit requeue removes the active assignment and workflow run link.

## Project Resolution

1. Normalize the current working directory to an absolute path.
2. Match `projects[].root` exactly first.
3. Use a name match only for legacy data when it refers to the same root.
4. If no project matches, register `p_` plus 6 random lowercase hexadecimal characters.
5. Allow different roots with the same folder name to be separate projects.

## Operations

### `task add "<title>"`

1. Resolve the current project.
2. Generate `t_` plus 6 random lowercase hexadecimal characters.
3. Create a `todo` task with ISO timestamps.
4. Write the store.
5. Return the task ID, title, and project.

Task titles must be concise Vietnamese text.

### `task list`

1. Resolve the current project.
2. Group tasks by `doing`, `todo`, then `done`.
3. Show assignment, execution, and workflow annotations when present.
4. Show at most 10 completed tasks unless the user requests more.
5. Before the groups, show a progress summary for all tasks in the current project: completion percentage, completed count, and remaining count.

Calculate `completion percentage` as `done / total * 100`, rounded to the nearest whole percent.
Treat `remaining` as every task not in `done`.
When the project has no tasks, show `0%`, `0 completed`, and `0 remaining`.

### `task list --all`

1. Read all projects and tasks.
2. Omit projects with no tasks.
3. Sort projects by name, then group tasks by status.
4. Show assignment, execution, and workflow annotations when present.
5. Before the project groups, show an overall progress summary for all stored tasks: completion percentage, completed count, and remaining count.

Use the same calculation rules as `task list`.
For `task list --all`, calculate the summary across every stored task, including tasks in projects omitted from display because their metadata is incomplete.

Use this compact format:

```text
[payment-api]
  Tiến độ: 25% (Hoàn thành: 1, còn lại: 3)
  DOING (1)
    t_ab12cd  "Sửa lỗi hoàn tiền"  -> payment-codex  [GNHF: AWAITING HUMAN]
  TODO (2)
    t_ef34ab  "Bổ sung test idempotency"
  DONE (1)
    t_98cd76  "Cập nhật tài liệu API"  [verified]
```

For `task list --all`, show one leading line such as `Tổng tiến độ: 25% (Hoàn thành: 3, còn lại: 9)`.
For `task list`, show the progress line immediately beneath the current project heading rather than repeating the heading.

Use the stored agent name when present, otherwise the agent type.
Do not query Herdr from this skill.
Display stored execution state only.
Display workflow router and current step when present.

### `task board`

Show a cross-project board from stored task, assignment, execution, and workflow data only.
Use `task board --current` when the user explicitly wants the current project instead of all registered projects.

Classify each task into exactly one lane:

- **Chưa giao**: `status: todo`.
- **Đang chạy**: `status: doing` with an explicit active non-terminal stored execution state.
- **Cần xử lý**: `status: doing` with missing execution state, execution `blocked` or `unknown`, or workflow `paused` or `blocked`.
- **Chờ duyệt**: `status: doing` with execution or workflow `awaiting_human`.
- **Đã xong**: `status: done`.

Show overall counts, per-project counts, and task cards grouped by lane.
Sort projects alphabetically and show at most 10 recently completed task cards unless the user requests more.
Label the output as stored state and do not imply that it is a live Herdr refresh.
If the user asks for live reconciliation, route that request to `herdr-orchestrate-agents` first.

### `task start <id>`

1. Find the task across all projects.
2. Reject missing or already completed tasks.
3. Refuse when another task in the same project is already `doing`.
4. Set the task to `doing` and update its timestamp.
5. Do not create assignment metadata.

Use `herdr-orchestrate-agents` instead when starting means dispatching to an agent.

### `task todo <id>`

1. Find the task across all projects.
2. Set it to `todo`.
3. Remove `assignment`, `execution`, and `workflow` because the task is no longer owned by a session or workflow run.
4. Update its timestamp and write the store.

### `task done <id>`

1. Find the task across all projects.
2. Reject missing or already completed tasks.
3. Set it to `done`.
4. Set `execution.state` to `completed` while preserving any existing execution evidence.
5. Preserve assignment history.
6. Update its timestamp and write the store.

### `task update <id>`

Use when the user asks to revise a task's title or details.

1. Find the task across all projects.
2. Apply only the fields the user explicitly changes.
3. Keep task titles concise Vietnamese text.
4. Preserve `status`, timestamps other than `updated`, and all assignment, execution, workflow, and unknown fields.
5. Update `updated` and write the store.
6. Return a full preview of the updated, persisted task, including every stored field and the resolved project name/root.
   Include ID, title, details (show `Không có` when absent), status, assignment, execution, workflow, created, and updated fields even when their values are empty or unavailable.

Do not return only a success message or a diff after an update.
Use the values re-read from the stored task for the preview.

### `task delete <id>`

Find the task across all projects and permanently remove it.
Return the deleted task ID and title.

### `task clean`

1. Resolve the current project.
2. Count completed tasks.
3. Ask for confirmation before removing them.
4. Write once after confirmation.

### `task current`

Show every `doing` task in the current project with stored assignment, execution state, and workflow step when present.
If none exists, point the user to `task list`.

### `task reset`

Use only for corrupted-store recovery after explicit user confirmation.
Replace the store with:

```json
{"version":2,"projects":[],"tasks":[]}
```

## File I/O Rules

- Read the complete file immediately before every mutation.
- Create a missing store with version 2 and empty arrays.
- Write atomically through a temporary file and rename.
- Preserve unknown fields at every level.
- If JSON parsing fails, do not overwrite the file.
- Warn about corruption and suggest `task reset`.
- Resolve `~` through the user's home directory.

## Constraints

- Standalone `task start` allows at most one `doing` task per project.
- Task titles and user-facing output must be in Vietnamese.
- The store is local only and must not trigger network calls.
- This skill does not inspect or control Herdr sessions.
