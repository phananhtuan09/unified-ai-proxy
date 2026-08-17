# Project Structure

> This document can be auto-generated via `generate-standards`. Edit manually as needed.

## Folders
- src/: source code
- tests/: all test files
  - unit/: unit test files (`*.spec.ts`, `*.test.ts`)
  - integration/: integration/E2E test files (`*.e2e.spec.ts`)
  - web/: orchestrated browser UI test files (`*.spec.ts`)
- docs/ai/project/: project docs (structure, conventions, patterns)
- docs/ai/planning/: epic and feature planning docs
- docs/ai/testing/: test plans per feature
  - `unit-{name}.md`: unit test docs (created by `/writing-test`)
  - `integration-{name}.md`: integration test docs (created by `/writing-integration-test`)
  - `web-{name}.md`: browser UI test docs (legacy/orchestrator-oriented flows only)
- docs/ai/verifications/: implementation and runtime verification artifacts per feature
  - `{name}.md`: verification record extended by `/verify-feature` and `/verify-runtime`
- docs/ai/tooling/: cross-tool workflow mapping and migration references

## Design Patterns (in use)
- Pattern A: short description + when to use
- Pattern B: short description + when to use

## Test Configuration

> This section is used by `/writing-test`, `/writing-integration-test`, `/test-web-orchestrator`, and `/run-test` commands.
> Run `/generate-standards` to auto-detect and populate this section.

### Unit Tests
- Framework: [Vitest/Jest/Mocha/pytest/etc.]
- Run command: `[npm test / npx vitest run / pytest / etc.]`
- Config file: [vitest.config.ts / jest.config.js / pytest.ini / none]
- Test location: `tests/unit/`
- File pattern: `*.spec.ts` or `*.test.ts`

### Integration Tests
- Framework: [Playwright/Cypress/etc.]
- Run command: `npx playwright test`
- Config file: `playwright.config.ts`
- Test location: `tests/integration/`
- File pattern: `*.e2e.spec.ts`

### Web Tests
- Framework: [Playwright/Cypress/WebdriverIO/etc.]
- Run command: `npx playwright test tests/web/*.spec.ts`
- Config file: `playwright.config.ts` or equivalent
- Test location: `tests/web/`
- File pattern: `*.spec.ts`

## Notes
- Import/module conventions
- Config & secrets handling (if applicable)

## AI Agent Workflow Assets
- `.claude/`: single source of truth for workflow content and migration inputs
- `.claude/commands/`: primary workflow commands to author and sync from
- `.claude/skills/`: primary skill definitions to author and sync from
- `.claude/agents/`: primary worker-role prompts to author and sync from
- `.claude/output-styles/`: primary response style definitions to author and sync from
- `.claude/themes/`: primary theme presets to author and sync from
- `.claude/scripts/`: reusable workflow scripts to sync from when needed
- `.agents/skills/`: Codex compatibility copies and native runtime skill mirrors
- `.agents/roles/`: Codex compatibility copies for worker roles
- `.agents/themes/`: Codex compatibility copies for theme assets

## AI Docs Roles (existing only)
- `docs/ai/project/`: repository-wide conventions and structure; workflow overview and navigation live in `README.md`.
- `docs/ai/planning/`: epic tracking docs and feature plans; use `epic-template.md` to decompose large requirements and `feature-template.md` to drive task execution.
- `docs/ai/testing/`: test plans and results
  - `unit-{name}.md`: unit test docs (from `/writing-test`)
  - `integration-{name}.md`: integration test docs (from `/writing-integration-test`)
  - `web-{name}.md`: browser UI test docs (legacy/orchestrator-oriented flows)
  - Run tests via `/run-test` command
- `docs/ai/verifications/`: implementation and runtime verification artifacts aligned to synced specs
- `docs/ai/tooling/`: cross-tool capability mapping and migration references used by sync workflows

## Guiding Questions (for AI regeneration)
- How is the codebase organized by domain/feature vs layers?
- What are the module boundaries and dependency directions to preserve?
- Which design patterns are officially adopted and where?
- Where do configs/secrets live and how are they injected?
- What is the expected test file placement and naming?
- Any build/deployment constraints affecting structure (monorepo, packages)?
