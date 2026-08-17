---
name: execute-task
description: Use when the user asks for a small local update, narrow bug fix, or refactor that does not need a durable spec.
---

# Execute Task

Execute a small, bounded coding task directly from the user request.

## Input

Quoted task description (e.g. `"Refactor: simplify avatar formatter"` or `"Fix button spacing in settings modal"`)

## Use This For

- small local updates
- tightly scoped bug fixes
- refactors with no behavior change
- work that does not add durable product knowledge worth storing in a spec

## Rules

- Keep exploration minimal and local to the change
- Do not create a spec unless the task expands into user-visible behavior or long-term product logic
- If the task grows beyond a small bounded change, stop and suggest switching to `/spec` → `/execute-spec`
- Validate changed behavior where practical

## Output

- Make the requested change
- Summarize what changed
- State what was verified and what still needs human checking
