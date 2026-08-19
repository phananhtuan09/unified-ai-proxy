# Unified AI Proxy

## Project

- Go 1.22 local proxy exposing OpenAI-compatible and Anthropic-compatible APIs.
- Supported upstreams: OpenAI Codex OAuth and Google Gemini API key.
- Read `spec.md` for the approved MVP contract; runtime behavior and tests are the source of truth when they differ from other documentation.

## Layout

- `main.go`: CLI entry point.
- `internal/cli`: commands and process lifecycle.
- `internal/server`: HTTP handlers and API translation.
- `internal/proxy`: routing and provider orchestration.
- `internal/provider`: upstream clients and SSE handling.
- `internal/config`, `internal/accounts`, `internal/tokenstore`, `internal/backup`: supporting services.

## Architecture source of truth

- Read `docs/dev/architecture.md` before adding or moving production code.
- Follow its dependency direction, package ownership, placement rules, and extension checklists.
- Treat `docs/ai/project/*` as workflow-building documentation, not application architecture guidance.

## Commands

```sh
go test ./...
go vet ./...
./scripts/check.sh
go run . start --config path/to/config.yaml
go run . help
```

Run focused tests during development, then `./scripts/check.sh` before handoff.

## Engineering rules

- Keep changes small and within the MVP scope; do not introduce mock providers.
- Preserve OpenAI and Anthropic response/error compatibility, including streaming SSE behavior.
- Never log API keys, OAuth tokens, authorization headers, passwords, or decrypted backups.
- Keep token files at restrictive permissions and retain authenticated local API endpoints.
- Add or update tests for changed behavior.
- Format Go code with `gofmt`; avoid unrelated refactors.
- Do not overwrite or revert existing user changes.
