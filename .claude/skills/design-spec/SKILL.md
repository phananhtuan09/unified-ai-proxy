---
name: design-spec
description: Create or resume a local interactive design review for a feature, collect high-level human decisions through the bundled local runner, and persist an approved decision manifest before detailed spec creation. Use before create-spec for new features or material user-visible changes that need human choices about scope, behavior, business rules, compatibility, or risk. Do not use for small execute-task changes, pure refactors, or already-approved design decisions.
---

# Design Spec

Create the human-facing design approval artifact that precedes the detailed implementation spec.

The agent authors a **design plan JSON**.
A **fixed viewer** shipped with this skill renders it.
Never write feature-specific HTML or CSS.

## Required Resources

Before creating or resuming the artifact, read:

- `references/design-contract.md`

Use these bundled resources:

- `assets/example/order-upsert-product.json` as a worked example of a complete plan
- `assets/review-viewer.html` as the fixed renderer, served by the runner and never copied or edited
- `scripts/validate_design_plan.py` to validate the plan before serving it
- `scripts/design_review_server.py` to serve the review and persist approval or feedback
- `scripts/validate_design_decisions.py` to validate the final decision manifest

Resolve all relative paths from the directory containing this `SKILL.md`.

## Workflow

1. Resolve the kebab-case `feature_slug`.
   - Under orchestrator, use the provided `feature_slug` exactly.
   - Write the plan to `docs/ai/designs/{feature_slug}.json`.
   - Approved decisions land in `docs/ai/design-decisions/{feature_slug}.json`.
   - Human change requests arrive in `docs/ai/design-feedback/{feature_slug}.json`.
2. If a decision manifest already exists, validate it.
   - Run `scripts/validate_design_decisions.py docs/ai/design-decisions/{feature_slug}.json --repo-root <repo-root>`.
   - If validation passes, stop the runner if it is still running and emit `continue`.
   - If validation fails, report the validator error and emit `stop-blocked`.
3. If feedback exists for the current `design_revision`, apply it before reopening review.
   - Each comment carries the id it targets, so revise that exact statement.
   - Increment `design_revision` in the plan and reopen review.
   - If the feedback revision is older than the plan revision, treat it as already handled.
   - Do not create a manifest from feedback.
   - A comment that carries no instruction, is too vague to act on, or contradicts another comment is not something to interpret.
   - Quote that comment, name the statement it targets, ask what the content should become, and stop as `ask-human`.
   - Leave the plan and `design_revision` untouched until every comment is actionable, so one revision carries one coherent set of changes.
4. Inspect the codebase before writing the plan.
   - Confirm current behavior, affected surfaces, existing patterns, constraints, and conflicts.
   - Anything the repository can answer must be answered now and recorded under `evidence` with a real path and line.
   - Never park a verifiable fact in `assumptions`.
5. Choose exactly one result before opening review:
   - `review-design`
   - `ask-human`
   - `split-slices`
   - `run-spike`
   - `escalate-conflict`
6. Stop without creating an approval manifest when the result is not `review-design`.
   - Ask at most five focused questions for `ask-human`.
   - Propose the smallest valuable executable slice for `split-slices`.
   - State the feasibility question for `run-spike`.
   - State the concrete codebase or business conflict for `escalate-conflict`.
7. For `review-design`, sort every statement into one of three tiers using the test in the contract.
   - Decision: if the agent guessed, two reasonable implementations would give results the user can tell apart.
   - Business rule: a threshold or behavioral rule the human should read, but not a forced click.
   - Implementation constraint: a guardrail that does not change observable behavior.
   - Anything else the agent can settle goes in `ai_discretion`, not on the page as a question.
   - Keep decisions between one and seven, preferring six or fewer.
8. Do not ask the human to choose something the goal already answers.
   - If the answer came from the human request or an approved upstream artifact, record it as a constraint with `source` set to `human-request` or `approved-upstream`.
   - If the agent invented the answer, rewrite the goal so it stops pre-answering and keep the question as a decision.
9. Build `preview` so the human sees the change instead of reading about it.
   - Select one kind: `ui`, `api`, `full-stack`, `workflow`, `data`, or `generic`.
   - For `ui`, express the affected screens with the wireframe nodes in the contract, using real field, column, and button labels.
   - For `api` and `full-stack`, provide a concrete request, response, and error table.
   - For `data`, provide the record shape before and after.
   - Mark every changed surface with `badge: new` or `badge: mod` and leave unchanged surfaces unmarked.
   - Always provide `states` and a `flow` whose steps carry failure and alternative `branches`.
   - Do not repeat in prose what the wireframe already shows.
10. Write the plan in Vietnamese for all human-facing text.
    - Keep code symbols, paths, and JSON keys in English.
    - Keep JSON keys and structure exactly as the contract defines them.
11. Validate the plan before opening the review:

    ```bash
    python3 .claude/skills/design-spec/scripts/validate_design_plan.py docs/ai/designs/{feature_slug}.json --repo-root <repo-root>
    ```

    - Fix every validation error before starting the runner.
    - Treat warnings as a prompt to shorten the review or verify evidence properly.
12. Start or reuse the local runner:

    ```bash
    python3 .claude/skills/design-spec/scripts/design_review_server.py --repo-root <repo-root> start docs/ai/designs/{feature_slug}.json
    ```

    - The runner binds to `127.0.0.1` on a port the operating system picks, serves the fixed viewer, and exits after printing the local URL.
    - It keeps running as a local background process so the human can review asynchronously.
    - It shuts itself down after 30 minutes with no request and clears its own state file, so a forgotten review never leaves an orphan process behind.
    - The open review page heartbeats, so the runner stays up while the human still has the tab open.
    - Tell the human the runner expires, and that asking to reopen the review starts it again.
    - If the runner cannot start, stop as blocked and keep the plan.
13. Stop after opening the review and ask the human to review asynchronously.
    - Report the local runner URL and plan path in prose.
    - Tell the human to use `Duyệt hướng triển khai` to write the manifest or `Yêu cầu chỉnh sửa` to write feedback.
    - Tell the human to ask the agent or orchestrator to continue afterwards.
    - Under orchestrator, emit `stop-ask-human` because approval has not been collected yet.
14. On a later invocation, inspect artifacts instead of waiting on a live poll.
    - If the manifest exists and validates, emit `continue`.
    - If feedback exists for the current revision, apply it and reopen review.
    - If neither exists, restart or report the runner URL and emit `stop-ask-human`.
15. Do not create the detailed Markdown spec from this skill.
    - `create-spec` consumes the approved decision manifest in the next workflow step.

## Review Surface Rules

- The plan carries content; the viewer carries presentation.
- Never emit HTML, CSS, inline styles, or layout hints from this skill.
- Never ask the human to confirm each statement one by one.
- Put the recommendation first and state a concrete tradeoff beside every option.
- Say why the agent must not guess, in each decision's `why`.
- Name what the agent will decide alone in `ai_discretion`, so the human knows where their control ends.
- Give every assumption an `impact_if_wrong`, and every risk a `consequence`.
- Keep the plan short enough that the human can understand the feature in three to five minutes.

## Approval And Source Of Truth

- The plan JSON is the reviewed content.
- The local runner is the approval and feedback transport, and it derives the manifest from the plan so the browser cannot inject content.
- The decision manifest records provenance honestly: `human` for chosen decisions, `agent-proposed-not-objected` for statements the human merely did not object to.
- The later reviewed Markdown spec is the source of truth for implementation and verification.
- Never treat chat text, DOM snapshots, screenshots, or free-form prompts as durable approval.
- Never publish or share the artifact through third-party hosting unless the human explicitly asks.

## Allowed Outcomes

- `Design approved`
- `Design review opened`
- `Questions needed`
- `Slice proposed`
- `Spike required`
- `Conflict escalated`
- `Blocked`

## Orchestrator Contract

When run under `/orchestrator`, append exactly one HTML comment as the final output line.

- Design approved:
  `<!-- orchestrator: outcome=continue provides=design_path,design_decisions_path design_path=docs/ai/designs/{feature_slug}.json design_decisions_path=docs/ai/design-decisions/{feature_slug}.json -->`
- Design review opened, questions needed, or review ended without approval:
  `<!-- orchestrator: outcome=stop-ask-human -->`
- Slice proposed:
  `<!-- orchestrator: outcome=stop-split-slices -->`
- Spike required:
  `<!-- orchestrator: outcome=stop-run-spike -->`
- Conflict escalated:
  `<!-- orchestrator: outcome=stop-escalate-conflict -->`
- Local runner unavailable, invalid manifest, or another hard dependency is missing:
  `<!-- orchestrator: outcome=stop-blocked -->`

Emit `continue` only after both files exist and the decision manifest validator passes.

## Self-Check

- Did the skill write only JSON, with no feature-specific HTML or CSS?
- Does every tier-1 decision pass the test that two reasonable implementations would differ visibly?
- Does the goal avoid pre-answering any open decision?
- Does the preview show the change with `MỚI` and `ĐỔI` badges instead of describing it?
- Does the flow include the failure and alternative branches?
- Is every `evidence` path real, with no verifiable fact parked in `assumptions`?
- Does `ai_discretion` state what the human is not being asked about?
- Did the plan validator pass before the runner started?
- Did the initial review path stop without polling or writing a manifest?
- On resume, did the skill inspect manifest and feedback artifacts before reopening review?
- When a comment was unclear, did the skill ask the human instead of guessing, leaving `design_revision` unchanged?
- Did the manifest validator pass before emitting `continue`?
