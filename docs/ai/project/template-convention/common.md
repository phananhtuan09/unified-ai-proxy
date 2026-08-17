# CODE_CONVENTIONS Common Rules (Template Preload)

> These rules are preloaded by the standards generator before analyzing the codebase. They establish non-negotiable, high-signal conventions. The generator should merge these rules first, then append auto-discovered patterns.

## Principles — The "Why"
- DRY (Don't Repeat Yourself): Avoid duplicating code. Abstract common logic into reusable functions, classes, or modules.
- KISS (Keep It Simple, Stupid): Prefer simple, straightforward solutions over clever or unnecessarily complex ones. Code should be as simple as possible, but no simpler.
- YAGNI (You Ain't Gonna Need It): Do not add functionality or create abstractions until they are actually needed. Avoid premature optimization.

## Naming — Clarity & Descriptiveness
- Prefer meaningful, verbose names over abbreviations.
- Avoid 1–2 character identifiers (except conventional counters like `i`, `j`, `k` in very small loop scopes).
- Be consistent. If you use `getUser` in one place, use `getProduct` in another, not `fetchProduct`.
- Function names should be verbs; variables and classes should be nouns. (e.g., `calculateTotal()`, `const user`, `class Order`).

## Functions / Methods
- Single Responsibility Principle: A function should do one thing and do it well. If you can't describe what a function does in one simple sentence, it's likely doing too much.
- Limit Arguments: Prefer 0-2 arguments per function. If you need more than two, consider passing a single configuration object.
- Avoid Side Effects: Functions should be predictable. They should avoid modifying external state, global variables, or their own input arguments whenever possible. A function `calculateTotal(items)` should return a new value, not modify the original `items` array.

## Control Flow
- Prefer guard clauses (early returns) to reduce nesting depth.
- Handle errors and edge cases first.
- Avoid deep nesting beyond 2–3 levels. Refactor deeply nested logic into smaller, well-named functions.

## Variables & Data Structures
- Minimize Scope: Declare variables in the smallest possible scope (e.g., inside a loop or conditional block if they are only used there).
- Avoid Magic Values: Do not use "magic" strings or numbers directly in code. Define them as named constants to provide context and avoid typos.
  - *Bad:* `if (status === 2) { ... }`
  - *Good:* `const STATUS_APPROVED = 2; if (status === STATUS_APPROVED) { ... }`
- Prefer Immutability:** Do not reassign variables or mutate data structures if it can be avoided. Creating new data structures instead of changing existing ones leads to more predictable code.

## Error Handling
- Throw errors with clear, actionable messages.
- Do not catch errors without meaningful handling (e.g., logging, retrying, or re-throwing a more specific error). Swallowing exceptions silently is a major source of bugs.

## Comments
- Add comments only for complex or non-obvious logic; explain **"why"**, not **"how"**. Well-written code should explain the "how" by itself.
- Place comments above code blocks or use language-specific docstrings.
- Avoid trailing inline comments as they can clutter code.
- Remove Dead Code: Do not leave commented-out code in the codebase. Source control (like Git) is the history.

## Formatting
- Match existing repository formatting tools and styles (e.g., Prettier, Black, gofmt). Consistency is key.
- Prefer multi-line over long one-liners or complex ternaries for readability.
- Wrap long lines and do not reformat unrelated code in a change that is focused on logic.

## Types (for statically typed languages)
- Explicitly annotate public APIs and function signatures.
- Avoid unsafe casts or overly broad types (e.g., `any`, `Object`). Be as specific as possible.

## Change Discipline
- Perform changes via file editing tools; avoid pasting large code blobs in reviews.
- Re-read target files before editing to ensure accurate context.
- After edits, run only fast, non-interactive validation on changed files:
  - If a linter is configured, run it on changed paths (e.g., `eslint --max-warnings=0 <changed-paths>`). Use `--fix` when safe to auto-fix.
  - If a type checker is used, run type-check only (e.g., `tsc --noEmit` or `mypy <changed-paths>`).
  - Do NOT run Prettier as part of validation (formatting is enforced separately by tooling/CI).
  - Do NOT run a full build or dev server (to avoid unnecessary time cost).
  - Attempt auto-fixes up to 3 times for linter issues before requesting help.

