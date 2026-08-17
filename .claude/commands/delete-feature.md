---
name: delete-feature
description: Delete a feature cleanly — inventory all surfaces, analyze dependencies, choose deletion strategy, and verify completeness.
---

Delete a feature without leaving dead paths, dangling dependencies, or hidden entry points. Surface-level deletion is not enough.

## Workflow Alignment

- Provide brief status updates (1–3 sentences) before each step.
- For medium/large deletions, create todos (≤14 words, verb-led). Keep only one `in_progress` item.
- **Clarify before deleting**: if evidence for a dependency is insufficient, note it — do not delete blindly.

---

## Step 1: Clarify Scope

If feature name or scope is ambiguous, use `AskUserQuestion`:

- What is the feature name / entry point?
- Should related data (DB records, config, env vars) also be deleted?
- Are there any parts intentionally kept (e.g., shared utilities)?

Do not ask the user to choose a deletion strategy here — that decision is made in Step 4 after dependency analysis.

---

## Step 2: Feature Inventory

**Goal**: Find every surface where this feature exists before touching any code.

Scan the codebase for all of the following:

| Surface | What to look for |
|---------|-----------------|
| **UI entry points** | Buttons, menu items, links, nav items, modals |
| **Routes / navigation** | URL routes, page files, redirects |
| **Business logic** | Services, controllers, handlers, use cases |
| **API calls** | Endpoints, HTTP clients, GraphQL queries/mutations |
| **Permissions / roles** | Feature flags, RBAC rules, authorization checks |
| **Config / env vars** | Feature toggles, `.env` keys, app config |
| **Analytics / tracking** | Event names, tracking calls, telemetry |
| **Tests** | Unit, integration, E2E tests covering this feature |
| **Docs** | README sections, API docs, user guides |
| **Background jobs** | Cron jobs, event listeners, queues, workers |
| **Data / DB** | Tables, migrations, seed data owned by this feature |

**Tools:**
- Grep(pattern=...) to search for symbols, route names, event names
- Glob to find related files by path pattern
- Read to inspect related files

List everything found. If you are unsure whether an item belongs to this feature, mark it as `[uncertain]` — do not skip it.

---

## Step 3: Dependency Analysis

**Goal**: Identify what breaks or changes if this feature is removed.

For each surface found in Step 2, check:

- Is this code also used by other features? If so, **do not delete** — extract or isolate instead.
- Does this feature own a shared component/service, or just use it?
- Are there data models or DB schemas owned by this feature?
- Are there external integrations (webhooks, third-party APIs) that will dangle?

Mark each dependency as:
- `safe to delete` — only used by this feature
- `needs extraction` — shared, must decouple first
- `[uncertain]` — not enough evidence to decide

**Gate: Before proceeding to Step 4**

If any items are marked `[uncertain]`:
1. List them explicitly.
2. Use `AskUserQuestion`: block until resolved, or accept risk and proceed?
3. If **block**: stop here, ask user to investigate or provide more context.
4. If **proceed**: **strategy is forced to Soft disable — skip Step 4 selection entirely.**
   Hard delete is not available when unresolved items remain.

---

## Step 4: Choose Deletion Strategy

Only reached if **no unresolved `[uncertain]` items remain** (gate above passed cleanly).

Select one strategy and justify the choice:

| Strategy | When to use |
|----------|-------------|
| **Hard delete** | All items confirmed `safe to delete`, no shared dependencies |
| **Soft disable** (feature flag off → cleanup later) | Strategy forced by gate (unresolved `[uncertain]` items remain), OR uncertain deps need verification first |
| **Phased deprecation** | External consumers exist; need migration period |

Do not mix strategies.

---

## Step 5: Execute Deletion

**Before starting**: Hard delete is irreversible. If the codebase has uncommitted changes or no recent backup, confirm with the user before proceeding.

**Process:**
1. Remove in dependency order: remove consumers before providers.
2. Apply one surface category at a time (e.g., UI first, then routes, then logic, then data/DB last).
3. For data/DB: drop tables or migrations only after confirming no other feature references them and a backup exists or data loss is accepted.
4. Do not delete items marked `needs extraction` until extraction is complete.
5. Do not clean up unrelated code discovered along the way.

**Tools:**
- Edit / Write to remove code
- Bash(command="...") for any build/test verification after each category

---

## Step 6: Cleanup Completeness Check

After deletion, verify each item is addressed:

- [ ] Imports / exports referencing deleted code removed
- [ ] Routes and navigation entries removed
- [ ] Old tests deleted or updated
- [ ] Docs sections removed or updated
- [ ] Analytics events / telemetry calls removed
- [ ] Config keys and env vars removed (or documented as safe to remove)
- [ ] Permissions / RBAC rules cleaned up
- [ ] Fallback messages or dead copy removed
- [ ] Background jobs / cron jobs deregistered
- [ ] DB tables / migrations addressed (dropped, or marked for follow-up if deferred)

---

## Step 7: Validation

- Run existing tests to confirm no unintended breakage.
- Build the project to catch dead imports and missing references.
- If soft disable was chosen: verify the feature is unreachable through all entry points.

---

## Step 8: Output Summary

```
## Deletion Summary

**Feature**: [name]
**Strategy**: hard delete / soft disable / phased deprecation

**Surfaces found**: [list]
**Dependencies impacted**: [list with safe/extraction/uncertain status]

**Deleted**:
  - [file or section]: [what was removed]
  - ...

**Impact**:
  - Entry points removed: [UI, routes, API — what users/callers can no longer access]
  - Data / config affected: [DB tables, env vars, feature flags left, removed, or stale]
  - Blast radius: [shared components or services that now behave differently]
    - L1 direct: modules that imported or called deleted code
    - L2 transitive: callers of those modules, if impact propagates
    - L3 shared infra: DB schema, config, event bus, analytics — if touched

**How to verify** (steps for reviewer):
  1. [Build passes: run build command — expect no missing import errors]
  2. [Tests pass: run test suite — expect no failures referencing deleted code]
  3. [Navigate manually: confirm entry points (routes, buttons, menu items) are gone]
  4. [Check config: confirm env vars / feature flags are cleaned or disabled]
  5. [Check deferred items: list what still needs follow-up and why]

**Intentionally retained**:
  - [item]: [reason — shared utility, insufficient evidence, etc.]

**Deferred / follow-ups**:
  - [item]: [what needs to be done and why it was not done now]
```
