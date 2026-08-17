---
name: delete-feature
description: Use when the user asks to remove, delete, or disable a feature. Inventories all surfaces, analyzes dependencies, chooses deletion strategy, and verifies completeness. Do not use for bug fixes, refactoring, or partial UI changes that are not feature removals.
---

# Delete Feature

Delete a feature without leaving dead paths, dangling dependencies, or hidden entry points. Surface-level deletion is not enough.

## Inputs

- Feature name or description
- Optional: specific files, routes, or components known to belong to the feature

## Tool Mapping

- File inspection/edit tools -> inspect files with the runtime's read/edit/write tools and prefer precise targeted edits over broad rewrites
- Search tools -> use repository search and file listing tools to find all feature surfaces by symbol, route, event name, or path
- User clarification -> ask the user directly for scope decisions when needed
- Shell/validation tools -> run build or test commands when shell execution is available and appropriate

## Workflow

### 1. Clarify scope

If any is ambiguous, ask before proceeding:

- What is the feature name or entry point?
- Should related data (DB records, config, env vars) also be removed?
- Are any parts intentionally shared and must be kept?

Do not ask the user to choose a deletion strategy — that decision is made in Step 4 after dependency analysis.

### 2. Feature inventory

Find every surface before touching any code. Use `rg` and `find` to scan:

| Surface | What to look for |
|---------|-----------------|
| **UI entry points** | Buttons, links, nav items, modals |
| **Routes / navigation** | URL routes, page files, redirects |
| **Business logic** | Services, controllers, handlers, use cases |
| **API calls** | Endpoints, HTTP clients, GraphQL queries/mutations |
| **Permissions / roles** | Feature flags, RBAC rules, auth checks |
| **Config / env vars** | Feature toggles, `.env` keys, app config |
| **Analytics / tracking** | Event names, telemetry calls |
| **Tests** | Unit, integration, E2E tests covering this feature |
| **Docs** | README sections, API docs, user guides |
| **Background jobs** | Cron jobs, event listeners, queues, workers |
| **Data / DB** | Tables, migrations, seed data owned by this feature |

Mark items you cannot confirm as `[uncertain]` — do not skip them.

### 3. Dependency analysis

For each surface found, classify:

- `safe to delete` — used only by this feature
- `needs extraction` — shared code, must decouple first
- `[uncertain]` — not enough evidence to decide

**Gate before Step 4:**

If any items are `[uncertain]`:
1. List them explicitly.
2. Ask the user: block until resolved, or accept risk and proceed?
3. If **block**: stop, request more context.
4. If **proceed**: **strategy is forced to Soft disable — skip Step 4 selection entirely.**

### 4. Choose deletion strategy

Only reached when the gate above passed cleanly (no unresolved `[uncertain]` items).

| Strategy | When to use |
|----------|-------------|
| **Hard delete** | All items confirmed `safe to delete` |
| **Soft disable** | Strategy forced by gate (unresolved `[uncertain]` items remain), OR uncertain deps need verification first |
| **Phased deprecation** | External consumers exist; migration period needed |

Choose one. Do not mix strategies.

### 5. Execute deletion

**Before starting**: Hard delete is irreversible. If the codebase has uncommitted changes or no recent backup, confirm with the user before proceeding.

Remove in dependency order: consumers before providers.
Apply one surface category at a time using the runtime's precise file editing tools (e.g., UI first, then routes, then logic, then data/DB last).

For data/DB: drop tables or migrations only after confirming no other feature references them and a backup exists or data loss is accepted.

Rules:
- Do not delete items marked `needs extraction` until extraction is complete.
- Do not clean up unrelated code discovered along the way.

### 6. Cleanup completeness

After deletion, verify each category:

- [ ] Imports / exports referencing deleted code removed
- [ ] Routes and navigation entries removed
- [ ] Tests deleted or updated
- [ ] Docs sections removed or updated
- [ ] Analytics / telemetry calls removed
- [ ] Config keys and env vars cleaned or documented
- [ ] Permissions / RBAC rules removed
- [ ] Fallback messages or dead copy removed
- [ ] Background jobs deregistered
- [ ] DB tables / migrations addressed (dropped, or marked for follow-up if deferred)

### 7. Validate

Run build and test suite to confirm no dangling references or broken imports.
If soft disable was chosen: verify the feature is unreachable through all entry points.

### 8. Output summary

```
## Deletion Summary

**Feature**: [name]
**Strategy**: hard delete / soft disable / phased deprecation

**Surfaces found**: [list]
**Dependencies impacted**: [list with safe/extraction/uncertain status]

**Deleted**:
  - [file or section]: [what was removed]

**Impact**:
  - Entry points removed: [UI / routes / API — what users can no longer access]
  - Data / config affected: [DB tables, env vars, flags — left, removed, or stale]
  - Blast radius:
    - L1 direct: [modules that imported or called deleted code]
    - L2 transitive: [callers of L1 if impact propagates]
    - L3 shared infra: [DB schema / config / event bus — only if touched]

**How to verify**:
  1. Build passes — no missing import errors
  2. Tests pass — no refs to deleted code
  3. Navigate manually — entry points (routes, buttons, menu items) are gone
  4. Check config/env — cleaned or disabled
  5. Deferred items — list what still needs follow-up

**Intentionally retained**:
  - [item]: [reason — shared utility, insufficient evidence, etc.]

**Deferred / follow-ups**:
  - [item]: [what needs to be done and why not done now]
```

## When To Ask The User

Ask only when:
- Feature scope is ambiguous enough that the wrong boundary would cause unintended deletions
- `[uncertain]` items cannot be resolved by reading the codebase
- Strategy selection requires a business decision (e.g., external consumers)

## Quality Bar

- Do not delete only visible UI — check all surface categories
- Hard delete requires all items confirmed `safe to delete`; anything else → Soft disable
- Intentionally retained and deferred items must be listed explicitly, not silently omitted
