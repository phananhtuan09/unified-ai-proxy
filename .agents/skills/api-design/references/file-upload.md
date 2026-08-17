# File Upload API Design

## Choose the Right Approach

| Approach | When to Use | Max Size |
|----------|-------------|----------|
| **Multipart form-data** | Small files (<10MB), upload with metadata in one request | ~10MB |
| **Direct binary (PUT)** | Single file, no metadata needed | Depends on server |
| **Presigned URL (S3-style)** | Large files, bypass server, client uploads directly to storage | Unlimited |

For production systems with files >5MB: always use presigned URLs to avoid server memory/timeout issues.

## Multipart Upload Pattern

```
POST /documents
Content-Type: multipart/form-data

file: <binary>
name: "contract.pdf"
folderId: "abc123"
```

Response:
```json
{ "data": { "id": "doc_01", "name": "contract.pdf", "url": "...", "size": 204800 } }
```

## Presigned URL Pattern (Recommended for Large Files)

**Step 1 — Request upload URL:**
```
POST /uploads/presign
{ "filename": "video.mp4", "contentType": "video/mp4", "size": 52428800 }
```
Response:
```json
{
  "data": {
    "uploadId": "upl_01",
    "presignedUrl": "https://storage.example.com/...",
    "expiresAt": "2024-01-15T11:00:00Z"
  }
}
```

**Step 2 — Client uploads directly to presigned URL (PUT to storage)**

**Step 3 — Confirm upload:**
```
POST /uploads/upl_01/complete
```

## Validation Rules

Always enforce server-side (never trust client):
- File size limit (reject before reading full stream)
- Allowed MIME types via allowlist (not file extension)
- Virus/malware scanning before making file accessible
- Store files outside web root or in object storage — never in a public directory

## Response Fields for File Resources

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Stable identifier |
| `name` | string | Original filename (sanitized) |
| `size` | integer | Bytes |
| `mimeType` | string | Detected server-side |
| `url` | string | Download URL (may be presigned) |
| `expiresAt` | string | If URL is time-limited |
