---
phase: project
title: AI Workflow Build Docs
description: Guide for AI agents to read the workflow-building documents in this repository
---

# AI Workflow Build Docs

## Purpose
This folder contains only the documents needed to design, evolve, and maintain the AI agent workflow of this repository.

It is not a general project implementation guide.
It does not describe coding conventions or project structure for a real application codebase cloned from this workflow.

Read these files when the task is about workflow logic, routing, commands, artifacts, harness design, or agent behavior.

## Read Order
Use this order when the task is about building or changing the workflow system:

1. `docs/ai/project/README.md`
2. `docs/ai/project/WORKFLOW_CONSTITUTION.md`
3. `docs/ai/project/WORKFLOW_IDEA_REVIEW.md`
4. `docs/ai/project/WORKFLOW_CODING_STANDARD.md`
5. `docs/ai/project/WORKFLOW_EVALUATION_STANDARD.md`

If documents conflict, follow the more specific file for the current workflow task.

`WORKFLOW_CONSTITUTION.md` is the exception.
A more specific file decides how work is executed, but it cannot override the constitution's principles.
When a document conflicts with the constitution, surface the conflict instead of merging the two directions or inferring a resolution.

## When To Read Which File

### `docs/ai/project/WORKFLOW_CONSTITUTION.md`
Read when:
- designing or changing the AI agent workflow
- deciding whether a workflow proposal matches the long-term direction
- adding commands, skills, phases, or artifacts
- resolving trade-offs between AI autonomy, human control, evidence, and simplicity
- checking whether a new abstraction, gate, artifact, or runtime mechanism is over-engineered
- deciding whether a workflow pattern is worth keeping
- changing the three-phase `Plan → Implement → Validate` architecture or authority boundaries

This file is the single source of principles for building the workflow.
It defines the highest-level intent and the mandatory constraints that every workflow change must follow.

### `docs/ai/project/WORKFLOW_IDEA_REVIEW.md`
Read when:
- evaluating a rough idea before spec creation
- deciding whether to build, reuse, integrate, defer, research, or reject an idea
- choosing the smallest useful scope before `/create-spec`
- deciding whether a proposed feature should enter the interactive `/design-spec` review before detailed spec creation
- checking whether an idea should enter the standard coding workflow

This file defines the lightweight pre-spec decision workflow.

### `docs/ai/project/WORKFLOW_CODING_STANDARD.md`
Read when:
- implementing or updating the standard coding workflow
- deciding routing for feature, bug fix, refactor, or small update tasks
- aligning new commands with the repository's execution flow
- checking which artifact should be produced in each phase

This file defines the standard end-to-end workflow used by the agent system.

### `docs/ai/project/WORKFLOW_EVALUATION_STANDARD.md`
Read when:
- evaluating whether a workflow design is worth adopting, promoting, or replacing
- comparing two workflow variants
- reviewing an existing workflow before a promotion or retirement decision
- updating or changing any workflow artifact and need evidence for the decision

This file defines the standard workflow for evaluating workflows themselves.

## Task Routing Guide

### Workflow design task
Read:
- `docs/ai/project/README.md`
- `docs/ai/project/WORKFLOW_CONSTITUTION.md`
- `docs/ai/project/WORKFLOW_IDEA_REVIEW.md` when changing pre-spec idea evaluation
- `docs/ai/project/WORKFLOW_CODING_STANDARD.md`

### Workflow implementation task
Read:
- `docs/ai/project/README.md`
- the workflow document that defines the behavior being implemented or changed
- when touching the standard feature flow, also read the `design-spec`, `create-spec`, and `review-spec` contracts
- when touching verification or PR-readiness flow, also read the command contracts for `/verify-feature`, `/verify-runtime`, `/manual-checklist`, and `/review-pr`
- when touching orchestrated execution, also read `docs/ai/workflows/*.json` and the orchestrator skill contract

### Workflow review task
Read:
- `docs/ai/project/README.md`
- whichever workflow document defines the expected behavior

### Workflow evaluation task
Read:
- `docs/ai/project/README.md`
- `docs/ai/project/WORKFLOW_EVALUATION_STANDARD.md`
- the workflow artifact being evaluated (the subject under review)

## Agent Behavior Expectations
- Treat this folder as workflow-building documentation only.
- Do not use this folder as the coding standard for an external application project.
- Do not invent workflow phases that are not documented.
- Do not add commands, artifacts, or roles unless they satisfy the mandatory rules.
- Prefer the simplest workflow that still preserves reviewability and control.
- Keep workflow artifacts readable by humans.
- Escalate when the repository documents do not cover an ambiguous decision.

## Related Files
- `docs/ai/project/WORKFLOW_CONSTITUTION.md`
- `docs/ai/project/WORKFLOW_IDEA_REVIEW.md`
- `docs/ai/project/WORKFLOW_CODING_STANDARD.md`
- `docs/ai/project/WORKFLOW_EVALUATION_STANDARD.md`
- `docs/ai/workflows/`
- `docs/ai/workflow-evals/`
