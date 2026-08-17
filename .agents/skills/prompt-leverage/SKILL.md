---
name: prompt-leverage
description: Strengthen a raw user prompt into an execution-ready instruction set for an AI agent. Use when the user wants to improve an existing prompt, build a reusable prompting framework, wrap the current request with better structure, add clearer tool rules, or create a hook that upgrades prompts before execution. Do not use for already-specific requests, straightforward questions, or simple bug fixes.
---

# Prompt Leverage

Turn the user's current prompt into a stronger working prompt without changing the underlying intent. Preserve the task, fill in missing execution structure, and add only enough scaffolding to improve reliability.

## Workflow

1. Read the raw prompt and identify the real job to be done.
2. Infer the task type: coding, research, writing, analysis, planning, or review.
3. Rebuild the prompt with the framework below.
4. Keep the result proportional: do not over-specify a simple task.
5. Return both the improved prompt and a short explanation of what changed when useful.

## Transformation Rules

- Preserve the user's objective, constraints, and tone unless they conflict.
- Prefer adding missing structure over rewriting everything stylistically.
- Add context requirements only when they improve correctness.
- Add tool rules only when tool use materially affects correctness.
- Add verification and completion criteria for non-trivial tasks.
- Keep prompts compact enough to be practical in repeated use.

## Framework

Treat the upgraded prompt as:

`Objective -> Context -> Work Style -> Tool Rules -> Output Contract -> Verification -> Done`

Use these blocks selectively.

### Objective

- State the task and what success looks like.
- Define success in observable terms, ideally in one or two lines.

### Context

- List relevant files, URLs, constraints, assumptions, and information boundaries.
- Say when the agent must retrieve facts instead of guessing.

### Work Style

- Set depth, breadth, care, and first-principles expectations.
- Go broad first when system understanding matters.
- Go deep where risk or complexity is highest.
- Re-check with fresh eyes for non-trivial tasks.

### Tool Rules

- State when tools, browsing, file inspection, tests, or external tools are required.
- Prevent skipping prerequisite checks.

### Output Contract

- Define structure, formatting, tone, and level of detail.
- Prefer explicit deliverable shapes over vague quality adjectives.

### Verification

- Require checks for correctness, grounding, completeness, side effects, and better alternatives.

### Done Criteria

- Define what must be true before the agent stops.

## Intensity Levels

Use the minimum level that matches the task.

- `Light`: simple edits, formatting, quick rewrites.
- `Standard`: typical coding, research, and drafting tasks.
- `Deep`: debugging, architecture, complex research, or high-stakes outputs.

## Task-Type Adjustments

Apply the smallest set of adjustments that materially improves execution.

### Coding

- Emphasize repo context, file inspection, the smallest correct change, validation, and edge cases.

### Research

- Emphasize source quality, evidence gathering, synthesis, uncertainty, and citations.

### Writing

- Emphasize audience, tone, structure, constraints, and revision criteria.

### Review

- Emphasize fresh-eyes critique, failure modes, alternatives, and explicit severity.

## Output Modes

Choose one mode based on the user request.

- `Inline upgrade`: provide the upgraded prompt only.
- `Upgrade + rationale`: provide the prompt plus a brief list of improvements.
- `Template extraction`: convert the prompt into a reusable fill-in-the-blank template.
- `Hook spec`: explain how to apply the framework automatically before execution.

## Hook Pattern

When the user asks for a hook, model it as a pre-processing layer:

1. Accept the current prompt.
2. Classify the task and risk level.
3. Expand the prompt using the framework blocks.
4. Return the upgraded prompt for execution.
5. Optionally keep a diff or summary of injected structure.

Use a deterministic first-pass rewrite only when the user is explicitly asking for a hook or reusable automation layer. Otherwise, prefer direct prompt improvement in-context.

## Upgrade Heuristics

- Add missing blocks only when they materially improve execution.
- Do not turn a one-line request into a giant spec unless the task is genuinely complex.
- Preserve user language where possible so the upgraded prompt still feels native.
- Prefer concrete completion criteria over vague quality adjectives.

## Quality Bar

Before finalizing, check the upgraded prompt:

- still matches the original intent
- does not add unnecessary ceremony
- includes the right verification level for the task
- gives the agent a clear definition of done

## Upgrade Rubric

An upgraded prompt is good when it:

1. preserves original intent
2. reduces ambiguity
3. sets the right depth and care level
4. defines the expected output clearly
5. includes an appropriate verification step
6. tells the agent when to stop

If the prompt is already strong, say so and make only minimal edits.
