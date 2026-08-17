---
name: be-security-check
description: |
  Post-implementation backend security review scoped to a feature or flow: injection, broken auth, mass assignment, sensitive data exposure, missing rate limiting, improper authorization, business flow abuse, misconfiguration (OWASP API Security Top 10 2023).
  Use when: a backend feature has been implemented and needs a security pass before PR review or merge. Trigger phrases: "check security", "security review backend", "security audit", "security check after implementation". Skip when task is purely frontend styling, config-only change, or documentation update with no logic. SCOPE: always review only the files changed in the current feature/PR.
---

# Backend Security Check

This is a **post-implementation review** scoped to the current feature or flow. Read only the changed files — do not scan the entire codebase.

---

## Review Process

**Step 0 — Establish scope first:**
- If the user specified files or a feature name: use those as the boundary
- If not: run `git diff --name-only HEAD~1` (or ask the user) to get the list of changed files
- Only read and review those files. Follow imports one level deep if needed to trace a data flow.

1. Identify all entry points in scope: HTTP handlers, queue consumers, background jobs, webhooks
2. Trace data flow: user input -> validation -> business logic -> DB/external services -> response
3. Check each category below against actual code
4. Output: findings with severity, location, and fix

---

## 1. Injection

### SQL Injection
Look for raw query string concatenation:
```
// BAD
db.query(`SELECT * FROM users WHERE email = '${email}'`)

// GOOD
db.query('SELECT * FROM users WHERE email = $1', [email])
```
Flag: string interpolation or concatenation in any query, ORM `.raw()` calls with user input.

### Command Injection
```
// BAD
exec(`convert ${filename} output.pdf`)

// GOOD — validate filename against allowlist, or use API instead of shell
```
Flag: `exec`, `spawn`, `eval`, `Function()` with any user-controlled input.

### NoSQL Injection
```
// BAD — MongoDB operator injection
db.users.findOne({ email: req.body.email })  // if email = { $ne: null }

// GOOD
if (typeof req.body.email !== 'string') throw new ValidationError(...)
```
Flag: untyped user input passed directly into MongoDB/Redis queries.

---

## 2. Authentication & Authorization

### Missing Auth on Protected Routes
Verify every new route has auth middleware applied. Check for:
- Routes accidentally left public
- Auth middleware applied at wrong level (after business logic)

### Broken Object-Level Authorization (BOLA/IDOR)
```
// BAD — any authenticated user can access any order
app.get('/orders/:id', auth, async (req) => {
  return db.orders.findById(req.params.id)
})

// GOOD
return db.orders.findOne({ id: req.params.id, userId: req.user.id })
```
Flag: any resource fetch by ID that doesn't also filter by `userId`/`tenantId`.

### Broken Function-Level Authorization
Check that role/permission checks exist for privileged actions:
- Admin-only operations accessible by regular users
- Ownership not verified before update/delete

### JWT/Token Issues
- Token not verified (signature check skipped)
- `algorithm: none` accepted
- Token expiry not checked
- Sensitive data (passwords, secrets) in token payload

---

## 3. Input Validation & Mass Assignment

### Missing Input Validation
Every endpoint receiving external data must validate:
- Required fields present
- Types correct (string/number/boolean)
- String lengths bounded
- Enums validated against allowlist
- Numeric values in valid range

### Mass Assignment
```
// BAD — user can set any field including isAdmin
await user.update(req.body)

// GOOD — explicitly allowlist fields
await user.update({ name: req.body.name, email: req.body.email })
```
Flag: ORM update/create calls passing `req.body` directly without field allowlisting.

---

## 4. Sensitive Data Exposure

### Leaking Sensitive Fields in Response
```
// BAD
return db.users.findById(id)  // returns passwordHash, internalFlags, etc.

// GOOD
const { passwordHash, ...safeUser } = await db.users.findById(id)
return safeUser
```
Flag: response objects that include `password`, `passwordHash`, `secret`, `token`, `internalNote`, PII not needed by the client.

### Sensitive Data in Logs
```
// BAD
logger.info('Login attempt', { email, password })
logger.error('Payment failed', { cardNumber, cvv })
```
Flag: logging of passwords, tokens, card numbers, SSNs, PII in error/info logs.

### Secrets in Code
Flag: hardcoded API keys, DB passwords, JWT secrets in source files (should be env vars).

---

## 5. CSRF

Check when cookies are used for authentication:
- State-changing requests (POST/PUT/PATCH/DELETE) missing CSRF protection
- No `SameSite` attribute on session/auth cookies (`SameSite=Strict` or `Lax` for most SPAs)
- Cross-origin forms or mutations without CSRF token or double-submit cookie pattern

Not required for stateless JWT in `Authorization` header — CSRF only applies to cookie-based auth.

---

## 6. Rate Limiting & Abuse Prevention

Check for missing rate limits on:
- Authentication endpoints (`/login`, `/register`, `/forgot-password`) — brute force risk
- OTP/verification endpoints — enumeration risk
- Resource-intensive endpoints (search, export, report generation)
- Webhook receivers — replay attack risk

---

## 7. Unsafe File Upload

If the feature accepts file uploads, check:
- **No MIME type validation** — must check server-side using magic bytes/content inspection, not just file extension or client-supplied `Content-Type`
- **Executable upload** — files stored in a web-accessible directory without execution prevention (e.g., `.php`, `.sh` uploaded and served)
- **Missing size limit** — no `Content-Length` check or stream size cap before reading body
- **No malware scanning** — files made accessible without AV/malware scan when required by threat model
- **IDOR on file serve** — files served by ID without verifying the requesting user owns them
- **Storage in web root** — files stored at a public path instead of object storage or private directory

---

## 8. Webhook Signature & Replay Protection

If the feature exposes or consumes webhooks:
- **Missing signature verification** — incoming webhook payloads not verified against HMAC signature
- **No timestamp/replay check** — signature valid but `createdAt` not checked (allows replaying old events)
- **Signature compared with `===`** — must use timing-safe comparison (`crypto.timingSafeEqual`)
- **Secret hardcoded** — webhook secret in source instead of env var

---

## 9. Common Web Vulnerabilities

### Path Traversal
```
// BAD
fs.readFile(`./uploads/${req.params.filename}`)  // ../../etc/passwd

// GOOD
const safe = path.basename(req.params.filename)
if (!safe.match(/^[a-zA-Z0-9_\-.]+$/)) throw new Error('Invalid filename')
fs.readFile(path.join('./uploads', safe))
```
Flag: user input used in file paths without sanitization.

### SSRF (Server-Side Request Forgery)
```
// BAD — user controls the URL
fetch(req.body.webhookUrl)

// GOOD — validate against allowlist of domains
```
Flag: user-controlled URLs passed to `fetch`/`axios`/`http.get`.

### Timing Attacks
```
// BAD
if (token === storedToken)  // string comparison leaks timing

// GOOD
crypto.timingSafeEqual(Buffer.from(token), Buffer.from(storedToken))
```
Flag: direct string equality for token/password comparison.

---

## 10. Business Logic & Misconfiguration

### Sensitive Business Flow Abuse
Flows that can be abused without technical exploit — just automation or repeated calls:
- Promo/discount codes applied multiple times per user
- Cart/checkout flow with no per-user rate limit (inventory drain)
- Vote/like/referral endpoints callable in rapid succession
- Password reset with no cooldown (OTP enumeration)

Flag: high-value business operations with no per-user throttle or duplicate-action prevention.

### Security Misconfiguration
- CORS misconfiguration: `Access-Control-Allow-Origin: *` on endpoints that return user data or accept credentials
- Verbose error responses leaking stack traces, DB errors, or internal paths to clients
- Debug mode or profiling endpoints accessible in production (`/debug`, `/__admin`, `/actuator`)
- Security headers missing on API responses: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`

### Improper Inventory Management
- Old API versions still accessible (`/v1/...`) after `/v2` is live — old versions may lack newer security fixes
- Undocumented or shadow endpoints reachable that bypass auth/validation
- Third-party API responses trusted without validation: type-checking, size-checking, and sanitizing external API data before using it in DB writes or responses

---

## Output Format

For each finding:

```
[SEVERITY] Category — Short description
Location: src/routes/orders.ts:78
Issue: <what is vulnerable>
Fix: <concrete recommendation>
```

Severity levels:
- **CRITICAL** — exploitable immediately, must fix before merge
- **HIGH** — significant risk, fix before production deploy
- **MEDIUM** — should fix, lower exploitability or impact
- **LOW** — defense-in-depth, fix opportunistically

End with: total findings by severity, and the top 1-2 fixes required before merge.
