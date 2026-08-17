# React Conventions (Essential)

## Components
- Use PascalCase for component names; one component per file (primary export).
- Props are immutable; derive minimal state. Lift state up when shared.
- Use stable `key` for lists (not index) when order can change.

## Hooks
- Follow the Rules of Hooks (top-level, consistent order).
- Declare all hook dependencies explicitly; prefer stable callbacks with `useCallback` when passed to children.
- Memoize expensive computations with `useMemo` only when measured bottlenecks exist.

## State & Effects
- Keep state minimal and serializable; avoid duplicating derived data.
- Side effects belong in `useEffect` with correct deps; cleanup subscriptions/timers.

## Performance
- Avoid unnecessary re-renders: memo child components when props are stable.
- Prefer split code (lazy + Suspense) for large routes/widgets.

## Accessibility
- Use semantic elements first; include `aria-*` only when needed.
- Ensure interactive elements are keyboard-accessible and have discernible labels.

