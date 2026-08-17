# Design Spec Contract

## Artifact Roles

`docs/ai/designs/{feature_slug}.json` is the design plan.
The agent authors only this file.

`assets/review-viewer.html` is the fixed renderer shipped with this skill.
Never copy it per feature, never edit it to fit one feature, and never generate feature-specific HTML or CSS.

`docs/ai/design-decisions/{feature_slug}.json` is the durable approval provenance consumed by `create-spec` and `review-spec`.

`docs/ai/design-feedback/{feature_slug}.json` is the latest change-request payload from the review surface.

`docs/ai/specs/{feature_slug}.md` becomes the source of truth for implementation and verification after `review-spec` passes.

The decision manifest must not become a second implementation specification.
Keep it limited to approved intent, scope, decisions, rules, constraints, and evidence.

## What Approval Means

Approval means the human understood the direction, chose the open decisions, and let the agent settle the rest inside the stated constraints.
Approval does not mean the human verified every line of the plan.

Never build a review surface that requires the human to confirm each statement one by one.
Never record a statement as human-approved when the human only failed to object to it.

## The Three Tiers

Every statement in a plan belongs to exactly one tier.

**Tier 1 — decision.**
The human must choose.
Use this only when the options differ in user-visible outcome, business rule, feature scope, data compatibility or migration, access or security, or an expensive and hard-to-reverse commitment.

**Tier 2 — business rule.**
Shown prominently, read by the human, objectable per line, but never gated behind a required click.
Use this for thresholds, limits, and behavioral rules that the human should see because they change what the product does.

**Tier 3 — implementation constraint.**
Shown as a plain read-only list, objectable per line.
Use this for guardrails that keep implementation aligned without changing observable product behavior.

The operational test for tier 1 is this: **if the agent guessed, would two reasonable implementations produce results the user can tell apart?**
If yes, it is a decision.
If it merely changes code shape, it is a constraint.

Keep tier 1 between one and seven items, and prefer six or fewer.
Anything the agent can settle without changing observable behavior belongs in `ai_discretion`, not on the page as a question.

## Rules That Prevent Known Failures

Do not ask the human to choose something the goal statement already answers.
If the answer came from the human's own request or an approved upstream artifact, record it as an implementation constraint with `source` set to `human-request` or `approved-upstream`.
If the agent invented the answer, rewrite the goal so it stops pre-answering, and keep the question as a decision.

Do not repeat in text what the wireframe already shows.

Do not list something as an assumption when the repository can answer it.
Verify it, then record it under `evidence`.
`assumptions` is only for facts outside the codebase, such as business intent, external consumers, or organisational process.

Every `evidence` entry must resolve in the repository.
`validate_design_plan.py --repo-root` fails on any path that does not exist.
Only a deliberate teaching example may set `illustrative: true`, and the viewer then shows a warning banner on the page.

## Design Plan Schema

Write UTF-8 JSON with a trailing newline.
The file name must equal `{feature_slug}.json`.

```json
{
  "schema_version": 3,
  "feature_slug": "order-upsert-product",
  "design_revision": "1",
  "title": "Short human title",
  "goal": "One sentence that does not pre-answer any open decision",
  "target_user": "Who performs the flow",
  "current_problem": "What is wrong or missing today",
  "preview": { "kind": "ui", "summary": "...", "screens": [], "states": [], "flow": [] },
  "scope": { "in": ["..."], "out": ["..."] },
  "decisions": [
    {
      "id": "D-001",
      "question": "...",
      "why": "Why the agent must not guess this",
      "options": [
        { "label": "Short name", "answer": "Full answer stored in the manifest", "tradeoff": "...", "recommended": true },
        { "label": "Short name", "answer": "...", "tradeoff": "..." }
      ]
    }
  ],
  "business_rules": [{ "id": "BR-001", "statement": "...", "why": "optional" }],
  "implementation_constraints": [{ "id": "IC-001", "statement": "...", "source": "agent" }],
  "ai_discretion": ["What the agent will settle without asking"],
  "assumptions": [{ "id": "A-001", "statement": "...", "impact_if_wrong": "..." }],
  "risks": [{ "id": "R-001", "statement": "...", "consequence": "..." }],
  "evidence": [{ "path": "src/x.ts", "line": 42, "observation": "What the agent saw there" }]
}
```

Limits enforced by `validate_design_plan.py`:

- `decisions` 1 to 7, each with 2 to 4 options and at most one `recommended`
- `business_rules` at most 6, `implementation_constraints` at most 8, `ai_discretion` at most 8
- `assumptions` and `risks` at most 5 each
- `scope.in` and `scope.out` 1 to 5 items each
- `preview.flow` 2 to 6 steps
- `preview.screens` at most 3
- ids unique across every section, matching `D-###`, `BR-###`, `IC-###`, `A-###`, `R-###`
- `implementation_constraints[].source` is one of `agent`, `human-request`, `approved-upstream`

## Preview Kinds

`preview.kind` is one of `ui`, `api`, `full-stack`, `workflow`, `data`, `generic`.

For `ui`, provide `screens`, which the viewer renders as a low-fidelity grey wireframe.
For `api` and `full-stack`, provide `preview.exchange` with `request`, `response`, and an `errors` table.
For `data`, provide `preview.shape` with `before` and `after`.
Other kinds may use either block.

Always provide `states` and `flow`.
`flow` steps carry `branches` for failure and alternative paths, because the branch is usually where the product decision lives.

Mark changed surfaces with `badge`: `new` renders as `MỚI`, `mod` renders as `ĐỔI`, and no badge means unchanged.
The before-and-after delta is the point of the preview.

## Wireframe Nodes

The wireframe is structure and labels only.
The viewer owns all styling; a plan that tries to carry visual design is wrong.

| type | required | optional |
| --- | --- | --- |
| `titlebar` | `label` | `right`, `badge` |
| `columns` | `columns` (array of node arrays) | `weights` |
| `box` | — | `label`, `badge`, `children` |
| `tabs` | `items[].label` | `items[].active`, `items[].badge` |
| `field` | `placeholder` | `badge` |
| `grid` | `cells[].lines` | `columns`, `cells[].badge` |
| `rows` | `rows[].label` | `rows[].right`, `rows[].chips`, `rows[].badge` |
| `total` | `label` | `value`, `badge` |
| `actions` | `buttons` | — |
| `text` | `text` | `badge` |

Nesting is limited to four levels.
Use real field, column, and button labels so the human can spot a wrong or missing one.

## Approval Payload

The viewer posts to `POST /api/design-approval`.

```json
{
  "schema_version": 3,
  "event": "design-spec-approval",
  "approved": true,
  "approval_meaning": "direction-approved",
  "feature_slug": "order-upsert-product",
  "design_revision": "1",
  "submitted_at": "2026-08-14T02:05:00.000Z",
  "answers": { "D-001": "The exact answer text of the chosen option" },
  "extra_constraints": ["Optional constraints typed by the human"]
}
```

The runner rejects the payload unless every decision is answered exactly once and every answer matches an option offered by the plan.
The runner reads the plan itself for all other content, so the browser cannot inject statements the plan never contained.

## Change Request Payload

The viewer posts to `POST /api/design-feedback`.

```json
{
  "schema_version": 3,
  "event": "design-change-request",
  "feature_slug": "order-upsert-product",
  "design_revision": "1",
  "submitted_at": "2026-08-14T02:05:00.000Z",
  "comments": [{ "target": "BR-001", "text": "The correct ceiling is 50%." }]
}
```

`target` must be an id present in the plan, or `general`.
When feedback exists for the current revision, the agent revises the plan, increments `design_revision`, and stops for another human review.

A comment is only a change request when it says what should change.
A comment that carries no instruction, is too vague to act on, or contradicts another comment must not be interpreted into a plan edit.
Ask the human what the targeted statement should become, leave the plan and `design_revision` unchanged, and stop as `ask-human`.
Hold the actionable comments until the unclear ones are resolved, so the human never sees a revision that changed for reasons they did not finish stating.

## Decision Manifest

The runner derives the manifest from the plan plus the human answers.
The agent never writes it by hand.

Provenance is recorded honestly:

- `decisions[].source` is `human` because the human chose it
- `business_rules[].source` is `agent-proposed-not-objected`
- `implementation_constraints[].source` keeps `human-request` or `approved-upstream` when the plan declared it, otherwise `agent-proposed-not-objected`

Other rules:

- `design_path` is repository-relative and points at the plan JSON
- `design_sha256` matches the plan bytes
- `approval_source` is `local-runner` and `approval_meaning` is `direction-approved`
- `goal`, decision questions, rule statements, and constraint statements must match the plan exactly
- `create-spec` must translate `output_preview`, `business_rules`, and answered decisions into behavior requirements and acceptance criteria without changing their meaning
- `create-spec` must not turn `ai_discretion` into a human commitment
- blocking product questions are not allowed in an approved manifest
- do not include secrets, raw transcripts, or unrelated annotations

## Local Runner Rules

- Validate the plan with `scripts/validate_design_plan.py <plan-path> --repo-root <repo-root>` before starting the runner.
- Start with `scripts/design_review_server.py --repo-root <repo-root> start <plan-path>`.
- The runner serves the fixed viewer at `/`, the plan at `/api/plan`, and health at `/api/status`.
- Keep the default loopback binding.
- The runner exits on its own after 30 minutes without a single request, and removes its own state file on the way out.
- The viewer heartbeats while the review page is open, so an open tab keeps the runner alive and a closed one lets it expire.
- Override the window with `--idle-timeout <seconds>`, or pass `0` to disable it, only when a review genuinely needs to outlive an unattended session.
- `stop` and `status` work even when the plan was deleted or became invalid, so an orphan runner never needs to be killed by hand.
- Do not publish internal design artifacts through third-party hosting.
- Do not add CDN dependencies to the viewer.
- Do not keep the agent waiting while the human reviews.
- On resume, check runner status and validate any manifest before continuing.
- Stop with `scripts/design_review_server.py --repo-root <repo-root> stop <plan-path>` after a valid approval manifest exists.
