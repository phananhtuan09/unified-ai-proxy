# Unified AI Proxy - Project Specification

## Overview

**Name:** unified-ai-proxy
**Description:** A local proxy server that exposes OpenAI-compatible and Anthropic-compatible API endpoints while authenticating upstream providers through browser OAuth flows (OpenAI Codex) and API keys (Google Gemini).
The MVP focuses on OpenAI Codex OAuth and Google Gemini API key only.
It must not use mock providers in the MVP implementation.
**License:** MIT
**Language:** Go
**Status:** MVP Phase

## MVP Scope Decisions

The MVP implementation must satisfy these confirmed decisions:

- Provider support is limited to **OpenAI Codex OAuth** and **Google Gemini API key**.
- OAuth browser login is required in MVP for OpenAI Codex.
- Gemini authentication uses an API key configured directly in the YAML config.
- Both `POST /v1/chat/completions` and `POST /v1/messages` are required in MVP.
- Streaming is required in MVP for both API protocols.
- Mock providers are not allowed as MVP behavior.
- SQLite usage tracking, config hot reload, and advanced observability are post-MVP.

## Tech Stack

| Component | Choice | Reason |
|---|---|---|
| Language | Go 1.22+ | Goroutine concurrency, single binary, fast compile |
| HTTP Framework | Gin | Fast, mature, middleware ecosystem |
| Config | YAML | Human-readable local configuration |
| Streaming | SSE via `net/http` `Flusher` | Native OpenAI and Anthropic streaming compatibility |
| Auth | OAuth 2.0 browser flow with PKCE (Codex); API key header (Gemini) | Codex avoids storing upstream keys; Gemini uses a user-supplied key |
| Local API Auth | Bearer API keys from config | Protect local endpoint from unauthorized clients |
| Token Storage | Local JSON files with `0600` permissions (OAuth providers) | Simple and portable MVP token persistence |
| Encryption | AES-256-GCM with Argon2id password-derived key | Config export/import encryption |

## API Endpoints

### `GET /health`

Returns server health.
This endpoint does not require local API key authentication.

Response:

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

### `GET /v1/models`

Returns all configured upstream models from enabled providers.
This endpoint requires local API key authentication.

OpenAI-compatible response shape:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-2.5-flash",
      "object": "model",
      "owned_by": "gemini"
    },
    {
      "id": "gpt-5-codex",
      "object": "model",
      "owned_by": "openai-codex"
    }
  ]
}
```

### `POST /v1/chat/completions`

Accepts OpenAI-compatible chat completion requests.
This endpoint requires local API key authentication.
It must support both streaming and non-streaming requests.

MVP-supported request fields:

- `model`
- `messages`
- `stream`
- `temperature`
- `top_p`
- `max_tokens`
- `stop`
- `metadata`

MVP unsupported request fields must return HTTP `400` with `unsupported_field` unless the field can be safely ignored.
Unsupported fields include:

- `tools`
- `tool_choice`
- `functions`
- `function_call`
- `response_format`
- `logprobs`
- `top_logprobs`
- `n`
- `audio`
- `modalities`

### `POST /v1/messages`

Accepts Anthropic-compatible Messages API requests.
This endpoint requires local API key authentication.
It must support both streaming and non-streaming requests.

MVP-supported request fields:

- `model`
- `messages`
- `system`
- `stream`
- `temperature`
- `top_p`
- `max_tokens`
- `stop_sequences`
- `metadata`

MVP unsupported request fields must return HTTP `400` with `unsupported_field` unless the field can be safely ignored.
Unsupported fields include:

- `tools`
- `tool_choice`
- `thinking`
- `container`
- `mcp_servers`
- binary/multimodal content blocks other than text

## Provider Support

### Provider Interface

Each provider must implement this interface:

```go
type Provider interface {
    Name() string
    Models() []Model
    ValidateAccount(ctx context.Context, account Account) error
    ChatCompletion(ctx context.Context, account Account, req *ChatRequest) (*ChatResponse, error)
    StreamChatCompletion(ctx context.Context, account Account, req *ChatRequest) (<-chan StreamEvent, error)
    RefreshToken(ctx context.Context, account Account) (*TokenSet, error)
}
```

Internal `ChatRequest`, `ChatResponse`, and `StreamEvent` types are normalized proxy types.
HTTP handlers translate OpenAI-compatible or Anthropic-compatible input into the normalized types before routing to a provider.

The `RefreshToken` method is meaningful only for OAuth providers (OpenAI Codex).
API-key providers (Gemini) return a "refresh not supported" error.
`Account` carries either an OAuth `token_file` (Codex) or an `api_key` (Gemini).

### OpenAI Codex Provider

The OpenAI Codex provider represents Codex access authenticated through browser OAuth.
The MVP must implement a real OAuth flow and must not return mock responses.

Required provider configuration:

```yaml
providers:
  openai_codex:
    enabled: true
    auth:
      method: oauth
      client_id: "TBD"
      authorization_url: "TBD"
      token_url: "TBD"
      scopes: []
      redirect_host: "127.0.0.1"
      redirect_port: 14552
      pkce: true
    api:
      base_url: "TBD"
    models:
      - id: "gpt-5-codex"
        upstream: "TBD"
    accounts:
      - name: "codex-main"
        token_file: "~/.config/unified-ai-proxy/tokens/codex-main.json"
```

Implementation blocker:

- `client_id`, `authorization_url`, `token_url`, OAuth scopes, and upstream API `base_url` must be resolved before the OpenAI Codex provider can be implemented.
- If the intended auth source is Codex CLI's existing login/session, the spec must define whether the proxy reads Codex token files or performs its own OAuth app flow.
- If reading Codex token files, the source path, token schema, refresh behavior, and permission expectations must be documented before implementation.

### Gemini Provider

The Gemini provider represents Google Gemini access authenticated with a static API key.
The MVP must implement a real API and must not return mock responses.
Gemini does not use OAuth: the API key is configured directly in the YAML config.

Required provider configuration:

```yaml
providers:
  gemini:
    enabled: true
    auth:
      method: api_key
    api:
      base_url: "https://generativelanguage.googleapis.com"
    models:
      - id: "gemini-2.5-flash"
        upstream: "gemini-2.5-flash"
      - id: "gemini-2.5-pro"
        upstream: "gemini-2.5-pro"
    accounts:
      - name: "gemini-main"
        api_key: "AIza..."
      - name: "gemini-backup"
        api_key: "AIza..."
```

Gemini API:

- Non-streaming: `POST {base_url}/v1beta/models/{model}:generateContent`
- Streaming: `POST {base_url}/v1beta/models/{model}:streamGenerateContent`
- Auth header: `x-goog-api-key: <api_key>`

`auth.method` must be `api_key`.
An account with an empty `api_key` fails startup validation.
The Gemini native API is publicly documented, so no OAuth blocker applies; the only required input is a real Google API key.

## OAuth Browser Flow (OpenAI Codex)

The `unified-ai-proxy auth openai-codex --account <name>` command starts an OAuth browser login.
The command must start a temporary local callback server and open the user's system browser.
The callback server must listen only on `127.0.0.1`.
PKCE must be used when supported by the upstream OAuth server.
The callback must validate the OAuth `state` parameter.

Command example:

```bash
unified-ai-proxy auth openai-codex --account codex-main
```

Gemini accounts use a configured API key and do not run the browser OAuth flow.
There is no `auth` subcommand for Gemini.

Token files must use this JSON schema:

```json
{
  "provider": "openai_codex",
  "account": "codex-main",
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_at": "2026-08-13T12:00:00Z",
  "scope": "...",
  "created_at": "2026-08-13T11:00:00Z",
  "updated_at": "2026-08-13T11:00:00Z"
}
```

Token file requirements:

- Token files must be written with `0600` permissions.
- Parent token directories must be created with `0700` permissions.
- Expired access tokens must be refreshed using `refresh_token` before sending an upstream request.
- If refresh fails, the account must be marked `reauth_required` and skipped until the user runs `auth` again.

## Routing Rules

Routing is model-driven.
The proxy must inspect the request `model` field and resolve it through configured provider model aliases.

Rules:

- If `model` matches an OpenAI Codex configured model alias, route to the OpenAI Codex provider.
- If `model` matches a Gemini configured model alias, route to the Gemini provider.
- If the model is unknown, return HTTP `404` with `model_not_found`.
- If multiple providers configure the same model alias, startup must fail with a config validation error.
- Endpoint type must not force provider choice.
- Both API endpoints can route to either provider through translation.

## Request and Response Translation

### Normalized Internal Request

Handlers must convert API-specific input into a normalized internal request before routing.

Required normalized fields:

- `Provider`
- `Model`
- `Messages`
- `System`
- `Stream`
- `Temperature`
- `TopP`
- `MaxTokens`
- `StopSequences`
- `Metadata`

### OpenAI to OpenAI Codex

When `/v1/chat/completions` routes to OpenAI Codex:

- OpenAI `system` and `developer` messages must be converted into a Codex `developer` input item.
- OpenAI `user` messages map to Codex `user` input items.
- OpenAI `assistant` messages map to Codex `assistant` input items.
- Only text content is supported in MVP.
- `max_tokens` maps to `max_output_tokens`.
- `stop` maps to Codex `stop`.
- `metadata` maps to Responses `metadata`.

### Anthropic to OpenAI Codex

When `/v1/messages` routes to OpenAI Codex:

- Anthropic `system` maps to a Codex `developer` input item.
- Anthropic `user` messages map to Codex `user` input items.
- Anthropic `assistant` messages map to Codex `assistant` input items.
- Only text content blocks are supported in MVP.
- `stop_sequences` maps to Codex `stop`.

Implementation blocker:

- The exact OpenAI Codex upstream API shape must be resolved before implementation.
- If Codex uses OpenAI Responses API instead of Chat Completions API, this section must be updated with the concrete Responses API mapping.

### Gemini Native Mapping

When either API endpoint routes to Gemini, the normalized request is rendered to the Gemini `generateContent` / `streamGenerateContent` request:

- `System` maps to Gemini `systemInstruction`.
- Normalized `user` messages map to Gemini `contents` entries with role `user`.
- Normalized `assistant` messages map to Gemini `contents` entries with role `model`.
- Only text content is supported in MVP.
- `temperature` and `top_p` map to `generationConfig.temperature` and `generationConfig.topP`.
- `max_tokens` maps to `generationConfig.maxOutputTokens`.
- `stop_sequences` (and OpenAI `stop`) map to `generationConfig.stopSequences`.

Note: Gemini requires the final `contents` entry to be a `user` turn.
Trailing `assistant` history must be handled by the proxy before sending (for example, by dropping or rejecting the request with `invalid_request`).

## Streaming SSE

Streaming must be supported for both external API protocols.
The proxy must flush every event through `http.Flusher`.
The response must stop when the provider stream completes or when the client disconnects.

### OpenAI-compatible Streaming Output

For `/v1/chat/completions?stream=true`, output must use OpenAI-style SSE frames:

```text
data: {"id":"...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}

data: [DONE]
```

### Anthropic-compatible Streaming Output

For `/v1/messages?stream=true`, output must use Anthropic-style SSE events:

```text
event: message_start
data: {"type":"message_start","message":{"id":"...","type":"message","role":"assistant","content":[],"model":"gemini-2.5-flash"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}

event: message_stop
data: {"type":"message_stop"}
```

Streaming failover rules:

- If an upstream request fails before any downstream bytes are sent, failover may retry with the next healthy account.
- If downstream streaming has already started, the proxy must not retry with another account.
- If an upstream stream fails after downstream streaming has started, emit a protocol-compatible error event where possible and close the stream.

## Multi-Account Round-Robin

Each provider can configure multiple accounts.
Account selection is per provider and uses round-robin across healthy accounts.

Rules:

- Disabled accounts are skipped.
- Accounts marked `reauth_required` (OAuth providers only) are skipped.
- Accounts marked temporarily unhealthy are skipped until their cooldown expires.
- Account selection must be concurrency-safe.
- The selected account name should be included in debug logs, but tokens and API keys must never be logged.

## Auto Failover

Failover applies only before response streaming starts.

Retryable failures:

- HTTP `429`
- HTTP `500`
- HTTP `502`
- HTTP `503`
- HTTP `504`
- Network timeout
- Temporary connection failure

Non-retryable failures:

- HTTP `400`
- HTTP `401`
- HTTP `403`
- Unknown model
- Unsupported request field
- Invalid request body

Config:

```yaml
routing:
  strategy: "round-robin"
  failover:
    enabled: true
    max_retries: 3
    unhealthy_cooldown: "5m"
    request_timeout: "2m"
```

A Gemini `403` (invalid API key) maps to `provider_auth_failed`.

## Local API Key Authentication

All `/v1/*` endpoints require:

```http
Authorization: Bearer <local-api-key>
```

Missing or invalid keys return HTTP `401`.
API keys are configured locally and are not related to upstream OAuth tokens or API keys.

## YAML Config

Default config path:

```text
~/.config/unified-ai-proxy/config.yaml
```

Example:

```yaml
server:
  host: "127.0.0.1"
  port: 8080
  api_keys:
    - "sk-local-key-1"
  default_max_tokens: 4096

providers:
  openai_codex:
    enabled: true
    auth:
      method: oauth
      client_id: "TBD"
      authorization_url: "TBD"
      token_url: "TBD"
      scopes: []
      redirect_host: "127.0.0.1"
      redirect_port: 14552
      pkce: true
    api:
      base_url: "TBD"
    models:
      - id: "gpt-5-codex"
        upstream: "TBD"
    accounts:
      - name: "codex-main"
        token_file: "~/.config/unified-ai-proxy/tokens/codex-main.json"

  gemini:
    enabled: true
    auth:
      method: api_key
    api:
      base_url: "https://generativelanguage.googleapis.com"
    models:
      - id: "gemini-2.5-flash"
        upstream: "gemini-2.5-flash"
      - id: "gemini-2.5-pro"
        upstream: "gemini-2.5-pro"
    accounts:
      - name: "gemini-main"
        api_key: "AIza..."
      - name: "gemini-backup"
        api_key: "AIza..."

routing:
  strategy: "round-robin"
  failover:
    enabled: true
    max_retries: 3
    unhealthy_cooldown: "5m"
    request_timeout: "2m"
```

Startup validation must fail when:

- No local API keys are configured.
- No provider is enabled.
- An enabled provider has no accounts.
- A configured OAuth account (Codex) has no token file.
- A configured API-key account (Gemini) has no `api_key`.
- Two providers expose the same model alias.
- Any required Codex OAuth/API field remains `TBD`.

## Import and Export Config

Transfer complete setup between local machines without manual reconfiguration.

Export:

```bash
unified-ai-proxy export --output backup.enc --password "my-secret"
```

Import:

```bash
unified-ai-proxy import --input backup.enc --password "my-secret"
```

Backup payload includes:

- YAML config (including Gemini API keys).
- OAuth token files referenced by config (Codex).
- Local API keys.

Encryption requirements:

- Use AES-256-GCM.
- Derive key with Argon2id.
- Include backup format version, salt, nonce, and KDF parameters in the encrypted file envelope.
- Imported token files must be restored with `0600` permissions.
- Imported directories must be restored with `0700` permissions.

Post-import behavior:

- Validate config.
- Validate token files exist and are parseable.
- Attempt token refresh for expired OAuth accounts (Codex).
- Report accounts requiring reauthentication.

## CLI Commands

```bash
unified-ai-proxy start
unified-ai-proxy start --config path.yaml
unified-ai-proxy auth openai-codex --account <name>
unified-ai-proxy accounts
unified-ai-proxy export --output <file> --password <pass>
unified-ai-proxy import --input <file> --password <pass>
unified-ai-proxy version
```

`auth` applies only to OAuth providers (OpenAI Codex).
Gemini API keys are managed by editing the config directly.

## Error Contract

Errors must use JSON responses.

OpenAI-compatible endpoint error example:

```json
{
  "error": {
    "message": "unknown model: foo",
    "type": "invalid_request_error",
    "code": "model_not_found"
  }
}
```

Anthropic-compatible endpoint error example:

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "unknown model: foo"
  }
}
```

Required error codes:

| Code | HTTP Status | Meaning |
|---|---|---:|
| `unauthorized` | 401 | Missing or invalid local API key |
| `invalid_request` | 400 | Request body is malformed |
| `unsupported_field` | 400 | Request uses an unsupported MVP field |
| `model_not_found` | 404 | Model does not match configured routing |
| `provider_auth_failed` | 401 | Upstream provider credentials are invalid |
| `reauth_required` | 401 | OAuth account requires browser login again |
| `rate_limited` | 429 | All available accounts are rate-limited |
| `provider_unavailable` | 503 | No healthy provider account is available |
| `upstream_timeout` | 504 | Upstream request timed out |

## Implementation Readiness

The project is ready to implement shared infrastructure from this spec.
Shared infrastructure includes config loading, CLI command wiring, API key middleware, routing validation, request type definitions, response writers, account state management, and provider integration seams.

The project is not ready to implement real OpenAI Codex OAuth until these values are resolved:

- OpenAI Codex OAuth authorization URL.
- OpenAI Codex OAuth token URL.
- OpenAI Codex OAuth client ID.
- OpenAI Codex OAuth scopes.
- OpenAI Codex upstream API base URL and request schema.
- Whether the existing Codex CLI token store may be read directly.

The Gemini provider has no unresolved OAuth values (the native API and API-key auth are publicly documented).
It requires only a real Google API key to be supplied.

## Expected Output After MVP Implementation

After all blocker values are resolved and the MVP is implemented:

1. Running `unified-ai-proxy auth openai-codex --account codex-main` opens a browser OAuth flow and stores a usable Codex token.
2. Configuring a Gemini API key in `config.yaml` enables Gemini access (no browser flow required).
3. Running `unified-ai-proxy start` starts the server on `http://127.0.0.1:8080`.
4. OpenAI-compatible clients can call `POST http://127.0.0.1:8080/v1/chat/completions` with streaming or non-streaming requests.
5. Anthropic-compatible clients can call `POST http://127.0.0.1:8080/v1/messages` with streaming or non-streaming requests.
6. Requests route by configured model alias to OpenAI Codex or Google Gemini.
7. Multiple accounts per provider are selected by round-robin with pre-stream failover.
