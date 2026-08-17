---
name: be-performance-check
description: |
  Post-implementation backend performance review scoped to a feature or flow: N+1 queries, missing indexes, unbounded queries, blocking I/O, memory leaks, caching gaps.
  Use when: a backend feature has been implemented and needs a performance pass before PR review or merge. Run after implementation is functionally complete.
  Trigger phrases: "check performance", "performance review", "optimize backend", "review queries", "N+1 check".
  Skip when: task is purely frontend, config change, or documentation update.
  SCOPE: Always review only the files changed in the current feature/PR — never scan the entire codebase.
---

# Backend Performance Check

This is a **post-implementation review** scoped to the current feature or flow. Read only the changed files — do not scan the entire codebase.

---

## Review Process

**Step 0 — Establish scope first:**
- If the user specified files or a feature name: use those as the boundary
- If not: run `git diff --name-only HEAD~1` (or ask the user) to get the list of changed files
- Only read and review those files. Follow imports one level deep if directly relevant to a flagged issue.

1. Identify all new/changed DB queries, API calls, and data processing loops
2. Check each category below — flag only real issues found in the code, not hypotheticals
3. For each finding, include **evidence** — not just a claim:
   - Query issue: cite the query pattern and why it's problematic at scale
   - Missing index: reference the specific WHERE/ORDER BY columns in the query
   - Cache opportunity: state how often the call repeats and what the invalidation trigger would be
   - Slow path: cite timing evidence (EXPLAIN output, latency logs) if available, or note it's an estimate
4. Output: list of findings with severity, location, evidence, and fix recommendation

---

## 1. Query Issues

### N+1 Queries
Look for loops that execute queries:
```
// BAD
for (const user of users) {
  user.orders = await db.orders.findMany({ where: { userId: user.id } })
}

// GOOD
const orders = await db.orders.findMany({ where: { userId: { in: userIds } } })
```
Check: any `findOne`/`findById` inside a loop, ORM lazy-loading inside iteration.

### Unbounded Queries
Any query fetching all rows without a LIMIT:
```
// BAD — will load entire table as data grows
db.query('SELECT * FROM events WHERE status = ?', ['pending'])

// GOOD
db.query('SELECT * FROM events WHERE status = ? LIMIT ? OFFSET ?', ['pending', limit, offset])
```
Flag: missing pagination on list endpoints, bulk operations without batching.

### Missing Indexes
Cross-reference query WHERE/ORDER BY columns against schema:
- FK columns not indexed
- Frequently filtered columns (`status`, `created_at`, `user_id`) without index
- Composite index column order mismatched with query pattern

When flagging a missing index, state the specific query pattern as evidence:
```
Query: WHERE tenant_id = ? AND status = ? ORDER BY created_at
Missing: composite index (tenant_id, status, created_at)
```

### Over-fetching
Selecting all columns when only a few are needed:
```
// BAD
SELECT * FROM users  -- returns 40 columns for a name display

// GOOD
SELECT id, name, email FROM users
```
Flag: `SELECT *` or ORM `.findMany()` without `select`/`fields` when only subset needed.

---

## 2. I/O & Async Patterns

### Sequential Async Where Parallel Is Possible
```
// BAD — sequential, ~300ms total
const user = await getUser(id)
const settings = await getSettings(id)
const permissions = await getPermissions(id)

// GOOD — parallel, ~100ms total
const [user, settings, permissions] = await Promise.all([
  getUser(id), getSettings(id), getPermissions(id)
])
```
Flag: multiple `await` calls in sequence when results are independent.

### Blocking Operations in Request Handler
- CPU-intensive work (large array sorts, crypto operations) on the main thread
- Synchronous file I/O (`fs.readFileSync`, `open()` without async)
- Missing timeouts on external HTTP calls

### Missing Connection Pooling
DB connections created per-request instead of using a shared pool.

---

## 3. Caching Opportunities

Flag when:
- Same expensive query runs on every request with same input (no cache)
- External API called on every request (rate-limit risk + latency)
- Static/rarely-changing data fetched from DB on every request

For each cache opportunity, provide:
- What to cache (key pattern, e.g., `user:{id}:permissions`)
- Suggested TTL based on how often data changes
- Invalidation trigger — must be explicit (on which write operation? which event?)
- Why cache is better than query optimization here (if not obvious)

Do not flag cache opportunities where the query is fast and called infrequently — cache adds complexity that needs justification.

---

## 4. Memory & Resource Issues

- Large arrays/objects built in memory before streaming/paginating
- Missing cleanup: unclosed DB connections, file handles, event listeners
- Result sets loaded fully into memory when streaming would suffice
- Recursive operations without depth/size limits

---

## 5. Response Payload Size

- Response includes unnecessary nested relations (over-fetching via ORM eager loading)
- Large base64-encoded binary data in JSON response
- No compression (gzip/brotli) for large payloads

---

## Output Format

For each finding:

```
[SEVERITY] Category — Short description
Location: src/services/orderService.ts:45
Issue: <what is wrong>
Fix: <concrete recommendation>
```

Severity levels:
- **HIGH** — will cause timeouts or failures under load
- **MEDIUM** — measurable degradation, fix before production traffic
- **LOW** — minor, fix opportunistically

End with a summary: total findings by severity, and the 1-2 highest-priority fixes.
