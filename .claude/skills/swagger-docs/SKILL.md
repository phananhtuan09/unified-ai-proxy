---
name: swagger-docs
description: |
  Generate and review OpenAPI 3.0 (Swagger) documentation following OpenAPI Initiative international standards — rules, naming conventions, and quality checklist focused on developer clarity, not YAML syntax.
  Use when: writing or updating Swagger/OpenAPI docs, documenting new endpoints, reviewing existing API docs for completeness, or generating OpenAPI spec from code.
  Skip when: the API is internal-only with no external consumers and no doc requirement. Skip when: making API design decisions (endpoint naming, HTTP conventions, versioning strategy) — use api-design instead.
---

# OpenAPI 3.0 Documentation

**Team standard**: [OpenAPI Specification 3.0.x](https://spec.openapis.org/oas/v3.0.3) — used here as the project baseline.
OAS 3.1 and 3.2 exist as newer versions with full JSON Schema alignment; migrate when tooling support matures.
Rules below are either **spec** (required by OAS 3.0.x — violations break tooling) or **convention** (industry best practice from OpenAPI Initiative — violations hurt developer experience).

---

## Core Principle

Documentation is complete when a developer who has never seen the codebase can:
1. Understand what each endpoint does from `summary` + `description` alone
2. Know exactly what to send from the request schema + examples
3. Know all possible responses including errors — without asking the implementer

---

## Naming Rules *(convention)*

| Element | Convention | Rule |
|---------|-----------|------|
| `operationId` | `camelCase` verb+noun | Unique across entire spec. Name the capability: `createOrder` not `postV1Orders` |
| Schema names | `PascalCase` | `CreateUserRequest`, `OrderResponse` — in `components/schemas`, not inline |
| Tag names | `Title Case` | Match resource noun: `Users`, `Orders`, `Payment Methods` |
| Parameter names | `camelCase` | `userId`, `pageSize`, `sortOrder` |
| Max length | 50 chars | Names over 50 chars break SDK generators and portals |

---

## Every Operation Must Have *(spec + convention)*

| Field | Rule |
|-------|------|
| `summary` | One sentence, verb-first: "Create a user account" — under 50 chars |
| `description` | Business context: when to call this, side effects, rate limits, preconditions |
| `operationId` | Unique camelCase verb+noun |
| `tags` | At least one tag matching a resource group |
| `security` | Declare the auth scheme, or `[]` for explicitly public endpoints |
| `responses` | At least one 2xx + all expected 4xx errors |

---

## Schema Rules

- **No inline schemas** *(convention)* — define all schemas in `components/schemas`, reference with `$ref`
- **Every property has `description`** *(convention)* — explain business meaning, not just the type
- **Every property has `example`** *(convention)* — use realistic data, never `"string"`, `1`, or `true`
- **Examples must be valid** *(spec)* — they must conform to the schema (tools use them for test generation)
- **Required parameters before optional** *(convention)* — in `required` array and parameter lists
- **Use `format`** *(convention)* where applicable: `email`, `date-time`, `uuid`, `uri`, `int64`
- **Document every `enum` value** *(convention)* — explain what each value means in `description`

---

## Response Rules

- Document **all** response codes the endpoint can realistically return
- Every error response includes: machine-readable `code` + human-readable `message`
- Reuse error response schema via `$ref` — consistent shape across all endpoints
- 4xx responses always have an `example` showing real error payload
- `GET` endpoints must have at least one 2xx response with content (not just 204)

---

## Reusability Rules (DRY)

- Common responses (`401 Unauthorized`, `500 Internal Server Error`) → `components/responses`
- Common schemas (error body, pagination, timestamps) → `components/schemas`
- Common parameters (`page`, `limit`, `Authorization` header) → `components/parameters`
- Security schemes → `components/securitySchemes`, applied globally, overridden per-endpoint
- Never repeat the same schema inline in multiple places

---

## File & Structure Rules

- Spec file committed to source control as a first-class artifact *(convention — OpenAPI Initiative)*
- `servers` array not empty — include at least production + staging URLs *(convention)*
- `info.contact` includes team name or support email *(convention)*
- `info.description` explains what the service does and who consumes it *(convention)*
- Tags defined at root level with descriptions *(convention)*
- Large APIs (>20 endpoints): split into multiple files using `$ref` with URL hierarchy *(convention)*

---

## Quality Checklist

Before finalizing OpenAPI docs:

**Structure**
- [ ] `servers` is not empty
- [ ] `info.contact` is filled
- [ ] All tags defined at root with descriptions
- [ ] Spec committed to source control

**Operations**
- [ ] Every operation has `summary`, `description`, `operationId`, `tags`, `security`
- [ ] `operationId` is unique, camelCase, under 50 chars
- [ ] Required parameters listed before optional

**Schemas**
- [ ] No inline schemas — all in `components/schemas`
- [ ] Every property has `description` (business meaning, not type restatement)
- [ ] Every property has a realistic `example`
- [ ] All examples validated against their schema
- [ ] Enums have per-value explanations in `description`

**Responses**
- [ ] All expected 4xx/5xx documented
- [ ] Error responses use consistent shared schema via `$ref`
- [ ] Every error response has a realistic `example`

**Reusability**
- [ ] Common responses in `components/responses`
- [ ] No duplicated schema shapes — use `$ref`

---

## Validation Tools

Use these to catch spec errors automatically:
- **Spectral** (by Stoplight) — linting rules engine, supports custom rulesets
- **Swagger Editor** — browser-based, real-time validation
- **openapi-validator** (IBM) — CLI validation
- Integrate into CI: fail PR if spec fails linting rules
