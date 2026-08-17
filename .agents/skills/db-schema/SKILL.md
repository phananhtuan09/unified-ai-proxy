---
name: db-schema
description: Use when designing new tables or collections, choosing between normalization approaches, planning indexes, writing migration scripts, modeling many-to-many or hierarchical data, or deciding on soft delete vs hard delete. Skip when writing queries against an already-defined schema with no design decisions.
---

# Database Schema Design

## Core Principles

- **Normalize by default** — denormalize only when query performance demands it (measure first)
- **Name consistently**: `snake_case` for tables/columns, plural table names (`users`, `orders`)
- **Standard tables**: include `id` (PK), `created_at`, `updated_at`
  - Exception: junction tables may use composite PK; event/audit tables often omit `updated_at`
- **Soft delete**: add `deleted_at TIMESTAMP NULL` only when data recovery or audit trail is required — not as a default on every table

## Column Naming Conventions

| Pattern | Example |
|---------|---------|
| Foreign key | `user_id`, `order_id` |
| Boolean | `is_active`, `has_verified` |
| Timestamp | `published_at`, `expires_at` |
| Status enum | `status` with CHECK or ENUM |
| Soft delete | `deleted_at` (NULL = active) |
| Tenant scoping | `tenant_id` on every multi-tenant table |

## Key Design Decisions

### Natural Key vs Surrogate Key
- **Surrogate (UUID/serial)**: default for most tables — stable, no business meaning leaking into schema
- **Natural key**: use when the business identifier is truly stable and unique (e.g., ISO country code, currency code)
- Avoid exposing auto-increment integers as public IDs — use UUIDs for externally visible IDs

### Cascade vs Restrict
- **RESTRICT** (default): parent row cannot be deleted if children exist — safe, explicit
- **CASCADE**: child rows deleted automatically with parent — use only when child records are truly "owned" (e.g., `order_items` owned by `order`)
- **SET NULL**: use for optional FK references (e.g., `assigned_user_id` when user is deleted)
- Always declare explicitly — never rely on DB default

### Soft Delete Caveats
When using `deleted_at`:
- Unique constraints must account for soft-deleted rows — use **partial unique index**:
  ```sql
  CREATE UNIQUE INDEX users_email_active ON users(email) WHERE deleted_at IS NULL;
  ```
- All queries must filter `WHERE deleted_at IS NULL` — enforce via ORM scope or view

## Multi-Tenant Schema Pattern

Every table with tenant-scoped data must have `tenant_id`:
```sql
orders(id, tenant_id, user_id, status, created_at)
```
- Add index on `tenant_id` on every tenant-scoped table
- Composite indexes: `tenant_id` first — `(tenant_id, status)`, `(tenant_id, created_at)`
- Never rely on application-layer filtering alone — add DB-level CHECK or RLS (Row Level Security) for sensitive tables

## Relationship Modeling

**One-to-many**: FK on the "many" side
```sql
orders.user_id -> users.id
```

**Many-to-many**: junction table
```sql
user_roles(user_id, role_id, created_at)  -- composite PK or surrogate id + unique constraint
```

**Self-referential** (e.g., category tree): `parent_id REFERENCES self(id)`

**Polymorphic** (avoid if possible): use separate FKs or a union table instead of `entity_type + entity_id`.

## Indexing Strategy

Add indexes for:
1. All foreign keys (most DBs don't auto-index FKs)
2. `tenant_id` on every multi-tenant table
3. Columns used in WHERE filters frequently (`status`, `user_id`, `created_at`)
4. Columns used in ORDER BY on paginated queries
5. Unique constraints (email, slug, external_id) — use partial index if soft-deleting

Composite index column order: **equality filters first, then range/sort columns**.
```sql
-- Query: WHERE tenant_id = ? AND status = ? ORDER BY created_at
INDEX (tenant_id, status, created_at)
```

Avoid over-indexing — each index slows writes. Add when query patterns are known.

## Migration Planning

**Zero-downtime migration rules:**
1. **Add nullable column** — safe, no lock
2. **Add NOT NULL column** — must have DEFAULT or backfill before adding constraint
3. **Rename column** — add new column, dual-write, migrate reads, drop old (3 deploys)
4. **Drop column** — remove all code references first, then drop in a later deploy
5. **Add index** — use `CREATE INDEX CONCURRENTLY` (Postgres) or equivalent

**Migration file naming:** `YYYYMMDD_NNN_description.sql` — ordered, descriptive.

## Common Patterns

**Audit log table** (immutable event history):
```sql
audit_logs(id, entity_type, entity_id, action, actor_id, payload JSONB, created_at)
```

**Status machine** — use an enum/CHECK constraint, never a boolean per state:
```sql
status VARCHAR CHECK (status IN ('pending', 'active', 'cancelled', 'completed'))
```

**Hierarchical data** (e.g., categories, comments):
- Simple trees: adjacency list (`parent_id`) — simple queries, slow deep traversal
- Deep traversal needed: Closure Table or Materialized Path

## Design Checklist

Before finalizing a schema:
- [ ] Standard tables have `id`, `created_at`, `updated_at` (note exceptions above)
- [ ] All FKs are indexed
- [ ] Multi-tenant tables have `tenant_id` indexed, composite indexes tenant-first
- [ ] Soft delete: only where recovery/audit needed; partial unique index added
- [ ] CASCADE/RESTRICT declared explicitly on all FK constraints
- [ ] Surrogate vs natural key decision documented
- [ ] No nullable column that should be required (tighten constraints early)
- [ ] Migration is zero-downtime compatible
- [ ] No polymorphic FKs unless justified
