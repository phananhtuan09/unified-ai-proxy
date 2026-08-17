# JavaScript Conventions (Essential)

## Language
- Use ES Modules (`import`/`export`).
- Prefer `const` (default) and `let`; avoid `var`.
- Use strict equality `===`/`!==`.
- Prefer optional chaining `?.` and nullish coalescing `??` over `||` when `0/''/false` are valid values.

## Syntax & Readability
- Use destructuring to access properties from objects and arrays.
- Use the spread syntax (`...`) for creating shallow copies of arrays and objects.
- Prefer array methods (`.map`, `.filter`, `.reduce`) over `for` loops for data transformation.
- Use `for...of` for simple iterations over iterables.

## Functions & Data
- Keep functions small and single-purpose.
- Use arrow functions (`=>`) for callbacks and to preserve `this` context.
- Avoid mutations; prefer immutable updates for objects/arrays.
- Return early (guard clauses) to reduce nesting.

## Errors & Async
- Use `async/await`; avoid unhandled promises (no floating promises).
- Throw `Error` objects with clear messages; catch only to handle meaningfully.

## Style & Safety
- Avoid implicit globals; module scope only.
- Prefer explicit returns over side effects.
- Keep imports minimal and ordered (std/third-party/internal).