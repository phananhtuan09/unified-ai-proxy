---
name: idea-review
description: |
  Use when the user has a rough product, workflow, or system idea and wants to decide whether it is worth doing before creating a spec. Focus on value, fit, build-vs-reuse/open-source/SaaS alternatives, smallest useful scope, and a clear verdict: do-now, defer, reject, reuse-or-integrate, or research-more. Do NOT use when a spec already exists, when the user only wants open-ended brainstorming, or when the task is already a small implementation request.
---

# Idea Review

## Purpose

Decide whether a rough idea should become a spec, be reused/integrated, deferred, researched further, or rejected.

This is a pre-spec decision workflow. The goal is `idea -> decision`, not `idea -> spec` by default.

## Hard Boundary

Do not implement code in this skill.

Do not create a full spec unless the human explicitly asks to continue into `/create-spec` after the review verdict.

Create a durable idea evaluation artifact only when the human asks for a saved handoff or the review is complex enough to preserve. Otherwise, respond inline.

## When To Use

Use this skill when the human asks things like:

- `I have an idea... should we do it?`
- `Evaluate this idea before spec`
- `Should we build this or use open-source/SaaS?`
- `What scope makes sense for this idea?`
- `Is this worth turning into a feature?`

## When Not To Use

Do not use when:

- the human already approved a clear feature and asks for a spec directly; use `create-spec`
- a spec already exists; use `execute-spec`, `sync-spec`, or `verify-feature` as appropriate
- the task is a small local update; use `execute-task`
- the human wants open-ended thinking only; use `brainstorm-partner`
- the task is to evaluate an AI workflow itself; use `workflow-evaluation`

## Core Workflow

### 1. Normalize the idea

Restate the raw idea in clearer terms.

Capture:
- raw idea
- interpreted goal
- safe assumptions
- critical missing context, if any

Ask only 1-3 focused questions if missing context blocks the decision. Do not start with a long questionnaire.

### 2. Value check

Answer:

```text
Does this solve a real problem for a real user?
```

Evaluate:
- target user
- pain point
- frequency or urgency
- expected business/user value
- cost of doing nothing
- existing workaround

Classify value as:

```text
weak | moderate | strong
```

### 3. Fit and feasibility check

Answer:

```text
Does this fit the product and implementation context?
```

Evaluate lightly:
- product fit
- technical fit
- complexity
- security, privacy, abuse, data, or operational risks
- dependencies on existing systems

Use codebase recon only when repository context materially affects the decision.

### 4. Build vs reuse check

Never assume custom build is the default.

Check options in this order:

1. existing codebase capability
2. simpler workflow using existing features
3. open-source library or base project
4. SaaS or external tool
5. custom build

Use web research when current external options, licenses, maturity, security, or provider fit materially affect the decision.

Recommend one of:

```text
build | reuse | integrate | research more
```

### 5. Scope recommendation

If the idea is worth doing, recommend the smallest useful scope.

Use three levels:

```md
### Small
Smallest useful version that solves the real pain.

### Medium
Practical product version with common supporting behavior.

### Large
Full vision; only appropriate when value is already validated.
```

Prefer `Small` unless it would fail to solve the real pain.

### 6. Decision

End with exactly one verdict:

```text
do-now | defer | reject | reuse-or-integrate | research-more
```

Decision rules:

- `do-now`: value is strong or clearly moderate, scope is small/medium, fit is acceptable, and no better reuse path exists.
- `reuse-or-integrate`: need is valid and an existing feature, open-source project, library, or SaaS solves most of it at lower cost.
- `research-more`: decision depends on external options, licensing, maturity, security, cost, or missing evidence.
- `defer`: value is plausible but not urgent, scope is larger than need, or workaround is acceptable.
- `reject`: value is weak, user/pain is unclear, or cost/risk is high relative to evidence.

## Standard Response Format

Use this concise format by default:

```md
# Idea Review: [Name]

## Verdict
[do-now | defer | reject | reuse-or-integrate | research-more]

## Why
- ...

## Build vs Reuse
- ...

## Recommended Scope
- Small: ...
- Medium: ...
- Large: ...
- Recommended: ...

## Not In Scope
- ...

## Next Step
- ...
```

If the verdict is `do-now`, the next step should usually be:

```text
Create a spec for the recommended scope.
```

If the verdict is `reuse-or-integrate`, the next step should usually be:

```text
Create an integration spec or run focused option research.
```

If the verdict is `research-more`, define the smallest research question or spike needed.

If the verdict is `defer` or `reject`, do not create a spec.

## Durable Artifact

Only when needed, save the review as:

```text
docs/ai/ideas/YYYY-MM-DD-{slug}-evaluation.md
```

Use the same standard response sections. Keep it concise.

## Relationship To Other Workflows

Use this before the standard coding workflow:

```text
raw idea
  -> idea-review
      -> reject
      -> defer
      -> research-more
      -> reuse-or-integrate
      -> do-now
  -> /create-spec
  -> /execute-spec
  -> /sync-spec
  -> /verify-feature
```

The detailed project standard is documented at:

```text
docs/ai/project/WORKFLOW_IDEA_REVIEW.md
```

The runnable workflow config is:

```text
docs/ai/workflows/idea-review.json
```

## Quality Bar

A good idea review:

- gives a clear verdict
- does not force weak ideas into specs
- considers reuse/open-source/SaaS before custom build
- recommends the smallest useful scope
- separates verified facts from assumptions
- makes human-owned business decisions visible
