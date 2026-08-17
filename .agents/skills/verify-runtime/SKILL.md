---
name: verify-runtime
description: Use when the user asks to run real browser-driven end-to-end verification against an approved spec and its spec-derived checklist. Loads the project E2E tool and environment definitions, exercises observable user flows through the configured browser tool, writes detailed evidence to the verification artifact, and updates checklist testcase status.
---

# Verify Runtime

Run browser-driven end-to-end verification for the testcases defined by the approved spec checklist.

## Bundled Project Configuration

Read these files from this skill directory before planning or executing any testcase:

- `tools.yaml` defines the available E2E drivers and the default driver.
- `project.env` defines project runtime targets, startup settings, viewports, artifact location, and credential environment-variable names.

Treat these files as the project defaults.
Explicit orchestrator or command inputs may override matching non-secret values for the current run.
Do not silently choose an undeclared tool or invent missing project settings.
Each project may update both files to match its own stack and environment.

Parse `project.env` as dotenv-style data.
Do not execute or shell-source it.
Resolve secrets only from the process environment variables named by `E2E_ACCOUNT_PASSWORD_ENV` and `E2E_API_KEY_ENV`.
Never print, persist, screenshot, or return secret values.

## Input

- Required: approved spec path, for example `docs/ai/specs/{feature-name}.md`.
- Required: checklist path, for example `docs/ai/checklists/{feature-name}.md`.
- Required: existing verification path, for example `docs/ai/verifications/{feature-name}.md`.
- Required: readable `tools.yaml` and `project.env` files in this skill directory.
- Optional: explicit runtime target, tool, auth, fixture, or setup overrides for the current run.

## Output

Update detailed E2E evidence in `docs/ai/verifications/{feature-name}.md`.
Update final evidence status in `docs/ai/checklists/{feature-name}.md`.
Store screenshots or other browser artifacts under the configured `E2E_ARTIFACT_DIR` when the selected tool supports file output.
The checklist is the primary human-facing output.

## Source Of Truth And Ownership

- Treat the approved spec as the only source of truth for expected behavior.
- Treat checklist testcase definitions and spec mappings as a projection of that spec.
- Use browser observations only as runtime evidence.
- Do not add, delete, split, merge, or rewrite checklist testcases from observed implementation behavior.
- Do not change expected results to match what the application currently does.
- If runtime behavior conflicts with the spec, record drift and mark the affected testcase red.
- If a required expected result is unclear, record a spec gap instead of guessing.

## End-To-End Definition

- Execute the tested behavior through the configured browser driver from the user-visible entry point.
- Verify the integrated application, including relevant UI state, navigation, browser-visible API traffic, persistence, and refresh behavior required by the testcase.
- Use real application routes and services from `project.env` unless the approved spec explicitly requires a mock or stub.
- Use direct API calls only for declared fixture setup or cleanup when that API interaction is not the behavior under test.
- Never use DOM injection, direct database mutation, internal state mutation, or route shortcuts to simulate the user action being verified.
- Do not substitute code inspection, unit tests, builds, lint output, or component rendering for E2E execution.
- Keep execution bounded to approved checklist testcases and the minimum setup, diagnostics, and cleanup needed to verify them.

## E2E Workflow

1. Read the approved spec, checklist, existing verification artifact, `tools.yaml`, and `project.env`.
2. Stop as blocked if a required artifact or configuration file is missing or unreadable.
3. Resolve the configured default tool and confirm that its declared MCP server or driver is available.
4. Resolve the base URL and other non-secret settings from explicit inputs first, then `project.env`.
5. Resolve required credentials from their named process environment variables without exposing their values.
6. Check the configured healthcheck or base URL.
7. If the target is unavailable and `E2E_START_COMMAND` is configured, run only that command and wait up to `E2E_START_TIMEOUT_SECONDS` for readiness.
8. Stop as blocked if the application, required account, test data, or selected browser driver is unavailable.
9. Select checklist testcases whose evidence strategy requires runtime, E2E, environment validation, or human judgment.
10. Convert each selected testcase into a browser scenario without changing its action, expected result, preconditions, role, viewport, or data variants.
11. Use the configured browser driver to establish preconditions, perform user actions, and assert the exact expected result.
12. Inspect browser console errors and relevant network requests for each scenario, especially when the testcase crosses frontend and backend boundaries.
13. Verify required persistence by revisiting, refreshing, or opening a clean browser context when the testcase requires it.
14. Capture the actual URL, viewport, role, test data identity, actions, assertions, visible result, network result, console result, and screenshot or artifact pointer.
15. Run declared cleanup when needed without deleting unrelated project or user data.
16. Record detailed evidence and update checklist status using the deterministic rules below.
17. Recalculate checklist counts, percentages, fully covered AC count, and verification evidence path.
18. Update the verification artifact and checklist before returning any orchestrator outcome, including fail or blocked outcomes.

## Browser Execution Rules

- Start from a clean browser context unless the testcase explicitly depends on an existing session.
- Authenticate through the UI when authentication is part of the testcase.
- Pre-authenticated state may be reused only when login is setup rather than the behavior under test, and the evidence must state that choice.
- Exercise every role, viewport, dataset, and branch explicitly required by the testcase.
- Use `E2E_DESKTOP_VIEWPORT` and `E2E_MOBILE_VIEWPORT` only as project test defaults when the spec does not mandate exact dimensions.
- Wait on observable readiness conditions instead of fixed sleeps whenever the driver supports it.
- Treat unexpected severe console errors, failed required requests, uncaught exceptions, or broken navigation as failures when they affect the tested flow.
- Do not mark a scenario passed if only the final UI state was observed without performing its required user actions.
- Do not continue destructive or state-changing scenarios when their exact target is ambiguous.

## Runtime Artifact Format

Append or update these sections without deleting valid implementation evidence:

```markdown
## Runtime Target
- Tool / driver
- Base URL / API URL / healthcheck
- Browser, viewport, role, and setup actually used

## Runtime Testcase Evidence
| Testcase | Spec mapping | E2E actions and assertions | Browser diagnostics | Artifact | Result | Checklist status |
|---|---|---|---|---|---|---|
| TC-001 | AC1 | [user steps and observed result] | [network and console result] | [screenshot or trace pointer] | Pass | 🟢 |
| TC-002 | AC2 | [evidence and limitation] | [diagnostic result] | [artifact pointer] | Partial | 🟡 |

## Automated Runtime Checks
- [browser scenario actually executed] → [result]

## Manual-Only Testcases
- [testcase requiring subjective human judgment and why]

## Runtime Failures / Blocks
- [failed or blocked testcase with reason]

## Spec Gaps / Drift
- [runtime behavior that contradicts or is not defined by the spec]

## Final Checklist Update
- Green: {green}/{total}
- Yellow: {yellow}/{total}
- Red: {red}/{total}
- Fully covered ACs: {covered}/{total_ac}

## Runtime Status
Pass | Fail | Partial | Blocked
```

## Checklist Evidence Rules

Icons are evidence classifications, not self-reported confidence.

- Set `🟢` only when the exact testcase was executed and passed through the configured browser driver with direct evidence covering its action, expected result, preconditions, and required environment.
- Set `🟡` when E2E evidence covers only part of the testcase, uses a narrower environment or dataset, or still requires subjective human judgment.
- Set `🔴` when the testcase was not run, failed, was blocked, has contradictory evidence, or depends on an unclear spec rule.
- Direct E2E evidence may upgrade an implementation-phase yellow testcase to green.
- E2E evidence must downgrade a previous green result when it contradicts earlier evidence.
- CSS declarations, code inspection, build output, or agent confidence cannot prove responsive, overflow, persistence, visible feedback, or other observable behavior.
- Prefer concrete evidence such as viewport dimensions, visible text, accessible state, URLs, DOM counts, screenshots, console output, network responses, and persisted values after refresh.
- Do not use one viewport, user role, dataset, happy path, or mocked dependency to mark broader testcase variants green.
- Subjective UX or business-judgment testcases remain yellow unless the spec defines objective assertions that were directly observed.
- Never change `[ ]` to `[x]` or erase an existing `[x]`.
- Green testcase items may remove an unchecked `[ ]` task marker.
- Yellow and red testcase items retain their human task marker.
- Keep the checklist AI verification note to one short method-and-result phrase.
- Put all detailed E2E evidence in the verification artifact.

## Checklist Integrity Rules

- Preserve testcase IDs, wording, expected results, order, and spec mappings.
- Preserve the checklist's Vietnamese section headings.
- Write updated summary labels, short AI verification notes, and spec-gap or drift findings in Vietnamese while preserving technical identifiers.
- Update only the icon, short AI verification note, human task marker, summary counts, percentages, fully covered AC count, evidence path, and spec-gap or drift section.
- Derive all percentages from testcase counts.
- Count an AC as fully covered only when every testcase mapped to it is green.
- Preserve implementation evidence sections in the verification artifact.
- If E2E setup prevents execution, update affected checklist items to red with a short blocked reason before stopping.

## Runtime Status Rules

- `Pass`: every E2E testcase selected for this phase passed with adequate direct browser evidence; manual-only cases may remain yellow.
- `Partial`: meaningful browser evidence was collected but one or more selected testcases remain incomplete or manual-only.
- `Fail`: at least one selected E2E testcase failed.
- `Blocked`: tool, application, auth, data, environment, checklist, or verification artifacts prevented meaningful E2E execution.
- Use `Not automatically verifiable` only in detailed evidence, never as the final runtime status.

## Artifact Boundaries

- Do not modify code, tests, specs, or testcase definitions during E2E verification.
- Do not repair failures in this phase.
- Do not recreate the verification artifact from scratch.
- Do not write secrets into artifacts, logs, screenshots, commands, or final output.
- Keep automation bounded to the approved spec checklist.

## Orchestrator Contract

When this skill is run under `/orchestrator`, append exactly one HTML comment as the final output line:

- Runtime status `Pass` or `Partial`:
  `<!-- orchestrator: outcome=continue provides=runtime_verified,checklist_path checklist_path=docs/ai/checklists/{feature-name}.md -->`
- Runtime status `Fail`:
  `<!-- orchestrator: outcome=stop-fail -->`
- Runtime status `Blocked`:
  `<!-- orchestrator: outcome=stop-blocked -->`

Rules:

- Emit the comment only after the verification artifact and checklist have been updated.
- Use `stop-blocked` if required artifacts or configuration are missing, the configured driver is unavailable, or runtime setup prevents meaningful execution.
- Report the checklist path as the primary human-facing output.
- If this skill runs standalone, the comment is optional.
