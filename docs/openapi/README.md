# Roskit OpenAPI Spec

Hand-written OpenAPI 3.1 description of the Roskit HTTP API.

## Layout

```
docs/openapi/
├── openapi.yaml              # root: info, servers, tags, security, components, paths
├── components/
│   ├── schemas/              # all object schemas, split per domain
│   │   ├── common.yaml       # Envelope, APIError, PaginationMeta, EmptyOK
│   │   ├── auth.yaml
│   │   ├── tenant.yaml
│   │   ├── user.yaml
│   │   ├── router.yaml
│   │   ├── hotspot.yaml
│   │   ├── ppp.yaml
│   │   ├── network.yaml
│   │   ├── system.yaml
│   │   ├── voucher.yaml
│   │   ├── template.yaml
│   │   ├── report.yaml
│   │   ├── event.yaml
│   │   └── sse.yaml
│   ├── parameters.yaml       # routerId, userId, paging, X-Tenant-Slug, etc.
│   ├── responses.yaml        # 400 / 401 / 403 / 404 / 409 / 429 / 500
│   └── securitySchemes.yaml  # bearerAuth, webhookToken, tenantSlugHeader
└── paths/
    ├── health.yaml
    ├── status.yaml
    ├── auth.yaml
    ├── tenant.yaml
    ├── users.yaml
    ├── routers.yaml
    ├── hotspot.yaml
    ├── ppp.yaml
    ├── network.yaml
    ├── system.yaml
    ├── voucher.yaml
    ├── template.yaml
    ├── report.yaml
    ├── quickprint.yaml
    ├── profile_mapping.yaml
    ├── event.yaml
    └── sse.yaml
```

The root `openapi.yaml` references each operation via `$ref` into the
matching `paths/<domain>.yaml` file. Schemas, parameters, responses, and
security schemes are likewise split for review-friendly diffs.

## Render locally

```bash
# Lint (zero error required)
npx @redocly/cli lint docs/openapi/openapi.yaml

# Live preview at http://localhost:8081
npx @redocly/cli preview-docs docs/openapi/openapi.yaml --port 8081

# Static bundle (single file, useful for CI artifacts)
npx @redocly/cli bundle docs/openapi/openapi.yaml -o docs/openapi/_bundled.yaml

# Swagger UI via Docker (alternative)
docker run --rm -p 8082:8080 \
  -e SWAGGER_JSON=/spec/openapi.yaml \
  -v $(pwd)/docs/openapi:/spec swaggerapi/swagger-ui
```

## Adding a new endpoint

1. **Identify the domain** — pick the matching `paths/<domain>.yaml`. If
   the new endpoint introduces a fresh schema, create or extend
   `components/schemas/<domain>.yaml` first.
2. **Write the operation** with all of: `operationId` (camelCase, unique),
   `summary`, `description`, `tags`, `security`, `parameters`,
   `requestBody`, `responses`. Always cover at least 200/201/204, 400, 401,
   403, 404, 500 where applicable.
3. **Reference the operation** from `openapi.yaml` under `paths`:
   `/your/path: { get: { $ref: 'paths/<domain>.yaml#/yourOp' } }`.
4. **Add an example** — at least one realistic request and response. Use
   the same example data the handler tests use when possible.
5. **Run** `npx @redocly/cli lint docs/openapi/openapi.yaml` until clean.

## Contract → code mapping

| Schema                                   | Source of truth                                   |
|------------------------------------------|---------------------------------------------------|
| `Envelope`, `APIError`                   | `pkg/httpresp/respond.go`                         |
| `Tenant`, `TenantSettings`               | `internal/models/tenant.go`, `tenant_settings.go` |
| `User`, `UserView`                       | `internal/models/user.go`, `services/auth_service.go` |
| `Router`, `RouterPublicView`             | `internal/models/router.go`, `services/router_service.go` |
| `HotspotUser`, `HotspotProfile`, etc.    | `internal/roskit/core/model/hotspot.go`           |
| `PPPSecret`, `PPPProfile`, `PPPActive`   | `internal/roskit/core/model/ppp.go`               |
| `Interface`, `Pool`, `Queue`, `DHCPLease`| `internal/roskit/core/model/network.go`           |
| `SystemResource`, `Scheduler`, `Script`  | `internal/roskit/core/model/system.go`            |
| `VoucherSale`, `GeneratedVoucher`        | `internal/services/voucher_service.go`, `internal/models/voucher_sale.go` |
| `PrintTemplate`                          | `internal/models/print_template.go`               |
| `OnLoginPayload`                         | `internal/api/handlers/event_handler.go`          |

When code drifts from the spec, fix the spec in the same PR. CI runs
`redocly lint` on every push to keep the spec syntactically valid.

## Conventions

- **operationId** is camelCase and globally unique (`listHotspotUsers`,
  not `list-hotspot-users`).
- **Error responses** always use `$ref: '#/components/responses/<Name>'`.
  Don't inline error envelopes — they drift.
- **Path params** always reference `components/parameters.yaml`. Don't
  inline `routerId: { type: integer }` in operation files.
- **Security**: omit `security:` to inherit the global `bearerAuth`.
  Public endpoints set `security: []` explicitly. Webhook endpoints set
  `security: [{ webhookToken: [] }]`.
- **`x-sse: true`** marks SSE endpoints; the response content type is
  always `text/event-stream`.
