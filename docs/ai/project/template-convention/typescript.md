# TypeScript Conventions (Essential)

## Types & Interfaces
- Use `interface` for public APIs and object shapes, as they are easily extended.
- Use `type` for unions (`|`), intersections (`&`), tuples, or complex type aliases.

## Naming Conventions
- Use `PascalCase` for all type-level identifiers (Types, Interfaces, Classes, Enums, Generics).
- Use `camelCase` for all value-level identifiers (variables, functions, properties).
- Avoid the `I` prefix for interfaces (e.g., `User`, not `IUser`).
- Use the `private` keyword instead of an underscore (`_`) prefix for private members.

## Type Safety
- Avoid `any`. It disables type-checking and defeats the purpose of TypeScript.
- Prefer `unknown` over `any`. It is a type-safe alternative that forces type-checking before use.
- Use `readonly` for properties that should not be mutated after initialization to enforce immutability.
- Explicitly type all public function signatures to ensure a clear and stable API.

## Modules & Organization
- Always use ES Modules (`import`/`export`).
- Prefer named exports over `default` exports. They improve consistency and refactorability.
- Avoid `namespace`. ES Modules provide superior encapsulation and are the language standard.

## Documentation
- Use JSDoc (`/** ... */`) for all exported members (functions, classes, types). This provides context and enables rich editor tooltips.