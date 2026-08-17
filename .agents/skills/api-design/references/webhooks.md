# Webhook Design

## Payload Structure

Keep payloads consistent across all event types:

```json
{
  "id": "evt_01HX1234",
  "type": "order.completed",
  "createdAt": "2024-01-15T10:30:00Z",
  "data": {
    "object": { "...full resource snapshot..." }
  }
}
```

Rules:
- `id`: unique per event (use for idempotency on receiver side)
- `type`: `resource.action` format (`order.completed`, `payment.failed`, `user.deleted`)
- `data.object`: full resource snapshot, not just changed fields — receiver shouldn't need to call back
- Never send only a partial diff; receivers may miss prior state

## Delivery

- Use HTTPS only — never HTTP
- Retry with exponential backoff: 1m, 5m, 30m, 2h, 8h (5 attempts)
- Mark delivery as permanently failed after max retries
- Deliver within 30s; return 2xx to acknowledge receipt, process async

## Security: Signature Verification

Sign every payload so receivers can verify authenticity:

```
X-Webhook-Signature: sha256=<HMAC-SHA256(secret, raw_body)>
```

Receiver must:
1. Compute HMAC using shared secret
2. Compare with constant-time comparison (prevent timing attacks)
3. Reject if signature mismatch
4. Reject if `createdAt` > 5min old (replay attack prevention)

## Registration API

```
POST /webhooks
{ "url": "https://customer.com/hooks", "events": ["order.completed", "payment.failed"] }

GET    /webhooks          — list subscriptions
DELETE /webhooks/{id}     — remove subscription
POST   /webhooks/{id}/test — send test event
```

## Consumer Rules (for teams implementing receivers)

- Return `200` immediately, process in background queue
- Use `id` for idempotency — same event may be delivered more than once
- Verify signature before processing
- Log raw payload before processing for debugging
