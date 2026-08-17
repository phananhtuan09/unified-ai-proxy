---
name: react-best-practices
description: |
  React/Next.js performance optimization (NOT styling): async patterns, bundle, rendering performance.
  Use when: data fetching, bundle optimization, async/Promise handling, Suspense/streaming, performance review.
  Keywords: React, Next.js, performance, optimization, bundle, async, Promise, waterfall, rerender, memo, useMemo, useCallback, Suspense, RSC, server components, client components, SWR, data fetching, cache
  Do NOT use for: non-React projects, backend-only, CSS/styling (→ frontend-design-fundamentals).
---

# React Best Practices

Performance optimization guide for React and Next.js applications.

For detailed code examples, see `references/code-patterns.md`.

---

## Rule Categories by Priority

| Priority | Category | Impact |
|----------|----------|--------|
| 1 | Eliminating Waterfalls | CRITICAL |
| 2 | Bundle Size Optimization | CRITICAL |
| 3 | Server-Side Performance | HIGH |
| 4 | Client-Side Data Fetching | MEDIUM-HIGH |
| 5 | Re-render Optimization | MEDIUM |
| 6 | Rendering Performance | MEDIUM |
| 7 | JavaScript Performance | LOW-MEDIUM |

---

## Quick Rules Summary

### 1. Eliminating Waterfalls (CRITICAL)

- **Use `Promise.all()`** for independent async operations
- **Use Suspense boundaries** to show wrapper UI faster
- **Don't await** in parent when child can fetch independently

### 2. Bundle Size (CRITICAL)

- **Avoid barrel imports** - import directly from source files
- **Use `next/dynamic`** for heavy components (Monaco, charts, etc.)
- **Configure `optimizePackageImports`** in Next.js 13.5+
- **Affected libs**: lucide-react, @mui/material, lodash, date-fns

### 3. Server-Side Performance (HIGH)

- **Use `React.cache()`** for server-side request deduplication
- **Use primitives** as cache arguments (not objects)
- **Restructure RSC** for parallel data fetching via composition

### 4. Client-Side Data Fetching (MEDIUM-HIGH)

- **Use SWR** for automatic deduplication and caching
- **Avoid useEffect + fetch** pattern

### 5. Re-render Optimization (MEDIUM)

- **Extract to memoized components** for early returns
- **Subscribe to derived booleans**, not raw values
- **Note**: React Compiler makes manual memo unnecessary

### 6. Rendering Performance (MEDIUM)

- **Use explicit ternary** for number conditionals
- `count && <X/>` renders "0" when count is 0
- Use `count > 0 ? <X/> : null` instead

### 7. JavaScript Performance (LOW-MEDIUM)

- **Build Map/Set** for repeated lookups (O(1) vs O(n))

---

## Quick Reference Checklist

### CRITICAL Priority
- [ ] Use `Promise.all()` for independent async operations
- [ ] Avoid barrel file imports (or use `optimizePackageImports`)
- [ ] Use `next/dynamic` for heavy components
- [ ] Defer third-party scripts (analytics, logging)

### HIGH Priority
- [ ] Use `React.cache()` for server-side deduplication
- [ ] Structure RSC for parallel data fetching
- [ ] Use Suspense boundaries strategically

### MEDIUM Priority
- [ ] Use SWR for client-side data fetching
- [ ] Extract expensive work to memoized components
- [ ] Subscribe to derived booleans, not raw values
- [ ] Use explicit ternary for conditionals with numbers

### LOW Priority
- [ ] Build Maps for repeated lookups
- [ ] Cache property access in loops
- [ ] Use Set/Map for O(1) lookups

---

## References

- [Vercel React Best Practices Repository](https://github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices)
- [React.cache documentation](https://react.dev/reference/react/cache)
- [SWR documentation](https://swr.vercel.app)
- [Next.js Package Imports Optimization](https://vercel.com/blog/how-we-optimized-package-imports-in-next-js)
- [React Compiler](https://react.dev/learn/react-compiler)

For detailed code patterns and examples: `references/code-patterns.md`
