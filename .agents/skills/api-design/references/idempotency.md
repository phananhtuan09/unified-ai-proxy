# Idempotency Keys

## When to Use

Apply idempotency keys when a POST creates a resource and accidental retries (network timeout, client crash) would create duplicates:
- Payment processing
- Order creation
- Email/notification sending
- Any operation with real-world side effects

GET, PUT, DELETE are naturally idempotent — no key needed.

## Implementation Pattern

**Client sends a unique key per logical operation:**
```
POST /payments
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

**Server behavior:**
1. Check if key exists in `idempotency_keys` table
2. If found: return the stored response (do not re-execute)
3. If not found: execute operation, store `(key, response, expires_at)`

**Schema:**
```sql
idempotency_keys(
  key         VARCHAR(255) PRIMARY KEY,
  response    JSONB,         -- stored HTTP response body
  status_code INT,
  expires_at  TIMESTAMP      -- typically 24hr–7d
)
```

## Response Headers

Always return the key in the response so clients can confirm receipt:
```
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

Return `409 Conflict` if the same key is received while the first request is still in-flight.

## Key Generation

Keys must be generated client-side (UUID v4). Do not generate them server-side — the client needs the key before sending the request to handle retries correctly.
