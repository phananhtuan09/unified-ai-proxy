---
name: workflow-evaluation
description: Evaluate, compare, improve, promote, reject, or review an AI workflow. For workflow improvement and usage diagnosis, analyze real session history trace-first to find agent friction, rework, retries, human intervention, and first divergence before mapping findings back to workflow rules. For new-workflow adoption without traces, review artifacts and controlled exercises. Writes a Vietnamese HTML report to docs/ai/workflow-evals/.
---

# Workflow Evaluation

Evaluate observable agent behavior and determine whether, where, and how the workflow contributes to it.

## Required Source Of Truth

Always read:

- `docs/ai/project/WORKFLOW_EVALUATION_STANDARD.md`

Read when relevant:

- `docs/ai/project/WORKFLOW_CONSTITUTION.md`
- normalized session traces
- `.foreman/` when the repo is Foreman-managed: `done.md`, `log.md`, and pre-extracted traces in `.foreman/traces/`
- observations in `docs/ai/agent-observations/`
- legacy observations in `docs/ai/workflow-observations/`
- the workflow artifact under review, but only after the blind behavioral pass for trace-first evaluations
- existing reports in `docs/ai/workflow-evals/`

Do not reconstruct the evaluation process from memory when the standard is available.

## Route The Evaluation

Choose exactly one strategy and record it in the input contract.

Use `trace-first` when:

- the goal is workflow improvement, usage diagnosis, or regression review
- the user reports that the agent loops, retries, reworks, asks poorly, runs too much, or needs human correction
- representative session history is available

Use `artifact-first` when:

- the workflow is new and has no session history
- the goal is design, clarity, safety, portability, adoption, or promotion review
- only workflow artifacts and controlled scenarios are available

When improvement is requested but traces are unavailable, state `insufficient runtime evidence`. Do not substitute static review for observed behavior without making the scope change explicit.

## Boundaries

Do not use this skill to:

- perform the underlying product or coding task
- operate the workflow under review as the product task itself
- fix application code
- treat workflow compliance as the only quality signal
- attribute every agent problem to the workflow

Workflow changes are recommendations or improvement contracts. Implementing them is a separate task unless the user explicitly requests implementation.

## Input Contract

Required:

- `workflow_name`
- `workflow_artifacts`
- `workflow_version`
- `workflow_type`
- `claimed_purpose`
- `claimed_task_classes`
- `evaluation_goal`
- `evaluation_strategy`
- `expected_behavior`
- `prohibited_behavior`
- `runtime_context`

Required for `trace-first`:

- `session_corpus`
- selection, inclusion, and exclusion rules
- known denominator
- `behavior_questions`
- known cohort keys such as task class, workflow version, runtime, model, project, and outcome

Optional:

- comparison target
- known constraints and incidents
- change under test
- agent observations
- raw transcript paths or normalized traces
- baseline scenarios
- evaluation quality plan
- human reference review or calibration set
- quality thresholds

Use `unknown` for missing fields. Never invent evidence or a denominator.

## Foreman Corpus

When the repo is managed by `foreman-agent`, `.foreman/` already supplies corpus, denominator, and outcome labels.

- `.foreman/traces/<id>-<timestamp>/` holds raw worker transcripts that Foreman copied when an item was approved or rejected; normalize each file with `extract_session_trace.py --input` before analysis, since Foreman only pins them and never parses them
- a worker on a database-backed runtime such as opencode leaves no copied file; locate that session with `--runtime opencode` and the `TASK: <id>` prompt string instead, and note the coverage gap when it cannot be found
- `.foreman/done.md` is the denominator, with one line per approved item; friction counts convert to rates only against it
- `↻N` on a `done.md` line is the recorded rework count for that item, and a missing `↻` means zero
- `.foreman/log.md` lines record orchestration-level events with timestamp, id, `@agent`, type, and detail

A `log.md` line is direct evidence that the event happened, not evidence of its cause.
Foreman records events without diagnosing them, so cause attribution still needs trace or artifact corroboration.

A `flagged` line is the exception: it is the human's own observation rather than an event Foreman witnessed.
Treat it as `agent-reported-observation` strength, and corroborate it against the trace directory carrying the same timestamp before promoting it to a finding.

An item that was rejected and later approved has two trace directories.
Treat the pair as one rework trajectory rather than two independent sessions, and count the item once in the denominator.

`.foreman/` is gitignored and lives in each managed repo, not in the workflow repo.
Its path must be supplied explicitly in `session_corpus`.
Never assume it exists, and state `insufficient runtime evidence` when it does not.

Foreman only observes the assignment boundary, so `log.md` alone cannot explain in-session behavior.
Use the traces for first divergence, retries, repeated reads, and tool failures.

## Session Trace Preflight

Normalize raw history before analysis when possible.
Transcripts pinned under `.foreman/traces/` are raw copies, so run the extractor on each one with `--input`.

```bash
python3 .agents/skills/workflow-evaluation/extract_session_trace.py --input <raw-transcript-path> --runtime codex
python3 .claude/skills/workflow-evaluation/extract_session_trace.py --input <raw-transcript-path> --runtime claude
python3 .agents/skills/workflow-evaluation/extract_session_trace.py --runtime opencode --session-id <session-id>
```

Latest matching local sessions:

```bash
python3 .agents/skills/workflow-evaluation/extract_session_trace.py --runtime codex --latest --project <repo-cwd>
python3 .claude/skills/workflow-evaluation/extract_session_trace.py --runtime claude --latest --project <repo-cwd>
python3 .agents/skills/workflow-evaluation/extract_session_trace.py --runtime opencode --latest --project <repo-cwd>
```

For Opencode, inspect `opencode session list` and `opencode db path`; history is normally in `~/.local/share/opencode/opencode.db`.

After extraction:

- use `session-trace.json` for metadata and normalized event access
- audit `chat-history.ndjson`, `command-transcript.ndjson`, `tool-call-trace.ndjson`, and `artifact-trail.ndjson` directly
- treat `decision-log.json` and `failure-retry-log.json` as extractor output, not proof that semantic decisions or failures were captured completely
- keep the raw source reference for traceability

## Four-phase Flow

Use four human-facing phases only:

```text
Frame -> Diagnose -> Decide -> Validate
```

Outcome reconstruction, segmentation, work classification, aggregation, attribution, workflow mapping, exercises, and replay are internal checklists, not separate workflow phases or required artifacts.

### 1. Frame

- freeze workflow version, goal, strategy, behavior questions, expected behavior, prohibited behavior, and required outcome evidence
- for trace-first, record corpus selection, inclusion, exclusion, denominator, cohort keys, and incomplete traces
- scan related observations when paths are not supplied; treat them as leads, not findings
- read only workflow metadata needed to identify the version before the blind pass
- for artifact-first, identify the artifact set, claimed scope, and task classes
- if runtime improvement is requested without traces, return `insufficient runtime evidence` unless the user accepts an explicit artifact-first scope change

Output: one `Evaluation Brief`. Do not create a separate artifact when the brief can live in the final report.

### 2. Diagnose

For trace-first:

- reconstruct intent, acceptance criteria, outcome, human intervention, commands, tool results, file changes, tests, and artifacts
- find the first divergence before focusing on downstream retries or rework
- segment by intent, decision, or hypothesis only when useful
- check misunderstanding, missing or unnecessary clarification, premature execution, repeated work without new evidence, edit-revert, rework, recovery loops, unused artifacts, unsafe actions, missing stops, and unsupported conclusions
- classify work when it explains impact: `productive-discovery`, `productive-execution`, `workflow-overhead`, `rework`, `recovery`, `coordination-cost`, `avoidable-repetition`, or `unknown`
- aggregate comparable sessions by failure mechanism with numerator, denominator, and counterexamples
- only after behavioral findings exist, read workflow rules, attribute cause, consider alternatives, and map findings to workflow contracts or `not related`

For artifact-first:

- normalize entry points, phases, inputs, outputs, artifacts, responsibilities, dependencies, assumptions, and stop/escalation rules
- record static findings as `design risk` only until trace or exercise evidence corroborates them

`Evidence Gate`: every main behavioral finding needs direct evidence, first divergence or `unknown`, honest recurrence status, an alternative explanation, and an evidence status no stronger than the source.

### 3. Decide

Route each supported finding by attribution:

- `workflow-caused` or `workflow-exacerbated`: consider the smallest targeted workflow experiment
- `model-execution`: default to `do not change workflow`
- `runtime-tool`: route to runtime, tool, integration, or recovery
- `task-environment`: route to context, environment, or task contract
- `inconclusive`: collect more evidence

Each workflow experiment defines the observed pattern, first divergence, attribution, falsifiable hypothesis, smallest change, success threshold, regression protection, and regression/neighbor/control replay set.

Output one of: `do not change workflow`, `need more evidence`, or `run targeted improvement experiment`. Adoption verdicts are only for explicit adoption or promotion reviews.

`Attribution Gate`: the recommendation must match the attributed cause and must not use workflow changes to hide unsupported model, runtime, tool, or environment diagnoses.

### 4. Validate

Validate evaluator quality:

- audit claim-to-evidence for every main finding
- report `evaluation_quality_status` as `calibrated`, `partially-calibrated`, or `uncalibrated`
- show `Chưa đo` when human reference or calibration evidence is unavailable
- use `calibrated` only when the declared thresholds pass on a representative supported cohort; use `partially-calibrated` for incomplete coverage or unmet metrics, and `uncalibrated` when reference evidence is unavailable
- when a calibration set exists, measure evidence support, first-divergence agreement, attribution agreement, and unsupported workflow attribution
- use the standard's initial targets: 100% direct evidence for main findings, at least 80% first-divergence and attribution agreement across at least five human-reviewed sessions, and zero unsupported workflow attribution in the audited sample
- treat small samples as calibration evidence, not statistical proof

Validate workflow changes with comparable regression, neighbor, and control replay. Classify each result as `Resolved`, `Improved but below threshold`, `Inconclusive`, `Regressed`, or `Hypothesis rejected`.

For artifact-first, use representative success and stress exercises. Static findings remain `design risk` until corroborated. Use canonical adoption verdicts only when requested.

`Validation Gate`: quality status and limitations are visible, main findings pass the claim-to-evidence audit, and every improvement claim has comparable replay evidence.

## Finding Contract

Every behavioral finding includes:

- stable finding ID and severity
- evidence status
- cohort and denominator
- observed pattern and trigger
- first divergence
- impact and work classification
- human intervention
- cause attribution and alternative explanation
- root-cause hypothesis and confidence
- workflow mapping when applicable
- smallest change or `do not change workflow`
- re-test

Treat observations created by `record-workflow-friction` as `agent-reported-observation`. They may describe workflow, model, runtime, tool, environment, task, or coordination friction. Corroborate before upgrading evidence status or calling them workflow failures.

## Output

Write a self-contained Vietnamese HTML5 report to:

```text
docs/ai/workflow-evals/{name}.html
```

Use `docs/ai/project/templates/workflow-evaluation-report.html`.

Required section IDs:

- `#decision-summary`
- `#session-behavior`
- `#key-findings`
- `#cause-attribution`
- `#improvement-impact`
- `#action-plan`
- `#scope-limitations`
- `#evidence-appendix`

Report rules:

- keep the decision summary under 120 words and main findings at five or fewer
- put corpus coverage, repeated behavior, first divergence, rework/recovery/repetition, human intervention, and attribution on the main page
- show `evaluation_quality_status`, calibration coverage, and `Chưa đo` for unavailable quality metrics
- keep normalized workflow structure and detailed methodology in the appendix unless directly decision-relevant
- use only measured counts, rates, duration, tokens, or tool calls; show `Chưa đo` otherwise
- escape trace content and omit secrets, credentials, tokens, and unnecessary raw transcript
- replace every template placeholder
- use canonical adoption verdicts only when adoption or promotion is actually requested

## Done When

Every evaluation:

- uses `Frame`, `Diagnose`, `Decide`, and `Validate` as the only human-facing phases
- passes main findings through the Evidence Gate and recommendations through the Attribution Gate
- reports evaluator quality as `calibrated`, `partially-calibrated`, or `uncalibrated`
- does not treat process completeness or report quality as proof of diagnostic correctness

For `trace-first`, also require:

- corpus and denominator are explicit
- session outcomes are reconstructed from evidence
- blind behavioral findings exist before workflow mapping
- findings include first divergence, work classification, and cause attribution
- only workflow-related findings create workflow changes
- improvement claims have comparable replay evidence

For `artifact-first`, also require:

- normalized structure and design risks are explicit
- runtime evidence is not invented
- exercise coverage and limitations are visible
- adoption verdict, when requested, matches the evidence

Every full evaluation ends with the required Vietnamese HTML report and no unresolved template placeholders.
