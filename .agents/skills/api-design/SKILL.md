---
name: api-design
description: Use when designing new API endpoints, reviewing API contracts, aligning FE/BE on schema, adding versioning strategy, defining error response structure, or any task involving API interface decisions. Skip when implementing existing well-defined API specs with no design decisions needed.
---

# REST API Design

**Base standard**: [Zalando RESTful API Guidelines](https://opensource.zalando.com/restful-api-guidelines/)
**Scope**: REST only. GraphQL/gRPC/WebSocket require separate guidelines.

**JS ecosystem deviations from Zalando** (explicit, intentional):
| Topic | Zalando | This skill |
|-------|---------|-----------|
| JSON property & query param naming | `snake_case` | `camelCase` (JS/TS convention) |
| Versioning default | Media-type versioning | URL path `/v1/` (more practical) |
| Error body format | RFC 7807 Problem JSON | Custom format below |

---

## URL / Path Rules

- **Nouns only, plural**: `/users`, `/orders`, `/products/{productId}/reviews`
- **kebab-case** for path segments: `/order-items`, `/payment-methods`
- **No trailing slash**: `/users` not `/users/`
- **Max one nesting level** for relationships: `/users/{userId}/orders` — avoid deeper nesting; flatten with query params when needed (e.g., `?userId=123` instead of `/users/{userId}/orders/{orderId}/items`)
- **No verbs** in path by default: `/orders` + POST, not `/create-order` — exception: RPC-style actions with no clean resource mapping (e.g., `/orders/{id}/cancel`, `/payments/{id}/refund`)

## HTTP Methods & Status Codes

| Action | Method | Success | Notes |
|--------|--------|---------|-------|
| List | GET | 200 | Must be paginated if unbounded |
| Get one | GET | 200 | 404 if not found |
| Create | POST | 201 | Return created resource in body |
| Full replace | PUT | 200 | Idempotent — requires full body |
| Partial update | PATCH | 200 | Send only changed fields |
| Delete | DELETE | 204 | No body |
| Batch operations | POST | 200 / 202 | 200 with per-item results in body; 202 if async. Avoid 207 (WebDAV-specific) |

Use most specific code available. Always include `429` with rate-limit headers when throttling.

Common error codes: `400` (validation), `401` (unauthenticated), `403` (unauthorized), `404` (not found), `409` (conflict), `422` (unprocessable entity).

## JSON Naming

- **Properties**: `camelCase` — `userId`, `createdAt`, `totalAmount`
- **Query parameters**: `camelCase` — `?sortBy=createdAt&pageSize=20`
- **Enum values**: `UPPER_SNAKE_CASE` — `"status": "ORDER_PENDING"`
- **Date/time properties**: end with `At` suffix — `createdAt`, `expiresAt`, `publishedAt`
- **Boolean properties**: start with `is` / `has` — `isActive`, `hasVerified`
- **Array property names**: plural — `items`, `tags`, `orderLines`

## Request/Response Structure

**Top-level must always be a JSON object** — never a bare array:

```
// List response
{ "data": [...], "pagination": { ... } }

// Single resource
{ "data": { "id": "...", ... } }
```

Follow project-existing envelope if one already exists.

**Error response** — consistent format across all endpoints:
```
{
  "error": {
    "code": "VALIDATION_ERROR",       // UPPER_SNAKE_CASE, machine-readable
    "message": "Human-readable text",
    "details": [{ "field": "email", "message": "Invalid format" }]
  }
}
```

## Filtering, Sorting, Pagination

- **Filtering**: query params — `?status=ACTIVE&userId=123`
- **Sorting**: `?sortBy=createdAt&order=desc`
- **Pagination** — prefer cursor-based for large/unbounded datasets:
  - Cursor: `?cursor=<token>&pageSize=20`
  - Offset: `?page=1&pageSize=20` (for small datasets / UI tables only)
- Pagination response shape must be consistent across all list endpoints

## Versioning

- **Default**: URL path versioning — `/v1/users`
- **Breaking changes** (require version bump): removing fields, changing field types, changing auth model
- **Non-breaking** (no bump): adding optional fields, new endpoints, relaxing validation
- Keep old version alive until all clients migrated; signal deprecation with `Sunset` header

## Design Checklist

Before finalizing an API contract:
- [ ] Path uses plural nouns, kebab-case, no trailing slash
- [ ] No verbs in URL
- [ ] All list endpoints paginated with consistent shape
- [ ] JSON properties are camelCase; enums are UPPER_SNAKE_CASE
- [ ] Error response follows project error format
- [ ] No sensitive data in URLs (use body or headers)
- [ ] Breaking vs non-breaking change assessed
- [ ] Idempotency considered for POST operations with side effects

## Optional Modules

Load these references only when the feature involves the specific concern:

- **Idempotency keys** (payment, order creation, notifications): see `references/idempotency.md`
- **File uploads**: see `references/file-upload.md`
- **Webhook design**: see `references/webhooks.md`
