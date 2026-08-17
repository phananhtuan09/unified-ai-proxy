---
name: auth-patterns
description: |
  Authentication & authorization patterns: JWT vs sessions, token lifecycle, RBAC/ABAC, middleware design, security pitfalls.
  Use when: implementing login/auth flow, designing token strategy, adding permission/role checks, building auth middleware, reviewing auth security, or choosing between auth approaches.
  Skip when: auth system is already fully defined and task is routine implementation with no design choices. Skip when: running a post-implementation security audit — use be-security-check instead.
---

# Auth Patterns

## Authentication: JWT vs Sessions

| | JWT (stateless) | Sessions (stateful) |
|---|---|---|
| Storage | Client (localStorage/cookie) | Server (DB/Redis) |
| Revocation | Hard (until expiry) | Easy (delete session) |
| Scale | No server state needed | Requires shared session store |
| Best for | APIs, microservices, mobile | Web apps, when instant revocation needed |

**Recommendation**: Use JWT for APIs consumed by mobile/SPAs. Use sessions for traditional web apps where instant revocation matters (e.g., banking).

## JWT Best Practices

- **Access token**: short TTL (15min–1hr), stored in memory or httpOnly cookie
- **Refresh token**: long TTL (7–30d), stored in httpOnly cookie or secure storage, single-use or rotated
- **Never store JWT in localStorage** if XSS is a concern — use httpOnly cookies
- **Cookie auth requires CSRF protection**: when using httpOnly cookies, implement one of:
  - `SameSite=Strict` or `SameSite=Lax` (sufficient for most SPAs)
  - Double-submit cookie or `X-CSRF-Token` header for cross-origin forms
- Payload: include only `sub` (user id), `roles`/`permissions`, `exp`, `iat` — no sensitive data
- Sign with RS256 (asymmetric) for multi-service architectures; HS256 fine for single service

**Token refresh flow:**
```
Client --(access token expired)--> API returns 401
Client --(refresh token)--> /auth/refresh --> new access + refresh tokens
```

**Revocation with JWT**: maintain a `token_revocation_list` table (blocklist by `jti`) or use short TTL + refresh rotation.

## Authorization: RBAC vs ABAC

**RBAC** (Role-Based Access Control) — use for most applications:
```
User -> Roles -> Permissions
admin: [read:all, write:all, delete:all]
editor: [read:all, write:own]
viewer: [read:all]
```

**ABAC** (Attribute-Based Access Control) — use when rules depend on resource attributes:
```
allow IF user.department == resource.department AND user.clearance >= resource.classification
```

Use RBAC by default. Add ABAC only when role checks are insufficient (e.g., row-level security, multi-tenant isolation).

## Middleware Design

Auth middleware should do exactly two things:
1. **Authenticate** — verify token/session, attach `req.user`
2. **Authorize** — check permissions for the route

Separate these concerns:

```
authenticate middleware -> attach req.user (401 if invalid)
authorize('read:orders') middleware -> check req.user.permissions (403 if denied)
route handler -> business logic only
```

Attach authorization at the route/application layer as the primary enforcement point. For sensitive actions (e.g., financial operations, admin mutations), add defense-in-depth authorization checks at the service/domain layer as well — don't rely on route middleware alone.

## Session Security

- **Session fixation**: always regenerate session ID after successful login — never reuse the pre-login session
- **Session invalidation**: destroy server-side session on logout (not just clear cookie)
- **Concurrent session control**: decide whether multiple concurrent sessions are allowed; if not, invalidate previous session on new login
- Cookie attributes: `HttpOnly`, `Secure`, `SameSite=Lax` (or `Strict`), `Path=/`

## Password Handling

- Hash with **bcrypt** (cost 12) or **argon2id** — never MD5/SHA1/SHA256 alone
- Rate-limit login attempts (5 attempts, then lockout or CAPTCHA)
- Constant-time comparison for token/password checks to prevent timing attacks
- On password reset: invalidate all active sessions/refresh tokens

## Common Security Pitfalls

| Pitfall | Fix |
|---------|-----|
| JWT in localStorage | Use httpOnly cookie |
| Long-lived access tokens | Max 1hr TTL |
| No refresh token rotation | Rotate on each use, detect reuse |
| `isAdmin` boolean in JWT | Use permissions list, verify server-side |
| Missing authorization on internal endpoints | Auth middleware on all routes by default |
| Exposing user enumeration in login errors | Return same error for "user not found" and "wrong password" |
| CORS wildcard `*` with credentials | Explicit origin allowlist |
| Cookie auth without CSRF protection | Use SameSite + CSRF token or double-submit |

## Multi-Tenant Authorization

For multi-tenant apps, always scope queries by `tenant_id`:
```
authorize: req.user.tenantId === resource.tenantId
```

Add `tenant_id` to JWT claims and validate in middleware before hitting the DB. Never rely solely on frontend to filter tenant data.

## Checklist

Before shipping an auth implementation:
- [ ] Tokens stored in httpOnly cookies or secure memory (not localStorage)
- [ ] Access token TTL <= 1hr
- [ ] Refresh token rotated on use
- [ ] Rate limiting on /login and /refresh
- [ ] Passwords hashed with bcrypt/argon2id
- [ ] Same error message for "user not found" vs "wrong password"
- [ ] Authorization checked server-side on every protected route
- [ ] Sensitive actions have service-layer authorization as defense-in-depth
- [ ] Cookie auth: CSRF protection in place (SameSite + token if cross-origin)
- [ ] Multi-tenant: all queries scoped by tenant_id
