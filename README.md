# Roskit

A modern Go backend for MikroTik hotspot management, designed as a **multi-tenant SaaS platform** with **Casbin RBAC**. Provides REST APIs and real-time SSE for managing hotspot users, PPP secrets, vouchers, sales reports, and RouterOS monitoring.

**Stack**: Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · InfluxDB 3 · Casbin v2
**Module**: `github.com/quiqxiq/roskit`

For detailed technical architecture and developer guidelines, see [docs/README.md](docs/README.md).

---

## Multi-Tenancy & RBAC

Roskit isolates data per **tenant** (organization). Every resource (router, voucher sale, user, template, audit log) carries a `tenant_id`. Access is enforced by **Casbin** with four roles:

| Role | Scope | Permissions |
|---|---|---|
| `superadmin` | Platform-wide | Full access to every tenant + `/admin/*` endpoints. Selects active tenant via `X-Tenant-Slug` header. |
| `owner` | Single tenant | Full access within tenant: settings, users, routers, templates, vouchers, reports. |
| `admin` | Single tenant | Same as owner except cannot edit tenant settings or billing. |
| `staff` | Single tenant | Operational only — hotspot users, vouchers, reports (read), templates (read). |

**Key terms:**
- **Tenant**: organization unit. Has a unique `slug` (lowercase, kebab-case) used at login and in Casbin policies.
- **Platform tenant**: the reserved slug `__platform__` used by superadmin for cross-tenant operations.

---

## Prerequisites

| Requirement | Version |
|---|---|
| Go | 1.24+ |
| PostgreSQL | 16 |
| Redis | 7 |
| InfluxDB 3 Core | optional (time-series metrics) |
| Docker & Compose | optional |

---

## Setup

### 1. Clone & Install Dependencies

```bash
git clone https://github.com/quiqxiq/roskit.git
cd roskit
go mod download
```

### 2. Environment Configuration

```bash
cp .env.example .env
```

| Variable | Description |
|---|---|
| `JWT_SECRET` | Secret for JWT access tokens (min 32 bytes) |
| `JWT_REFRESH_SECRET` | Secret for JWT refresh tokens (min 32 bytes) |
| `AES_ENCRYPTION_KEY` | AES-256 GCM key to encrypt router passwords (base64, 32 bytes) |
| `DB_PASSWORD` | PostgreSQL password |
| `REDIS_PASSWORD` | Redis password (leave empty if no auth) |
| `INFLUXDB_URL` | InfluxDB URL (leave empty to disable time-series metrics) |
| `INFLUXDB_TOKEN` | InfluxDB Token |
| `INFLUXDB_DATABASE` | InfluxDB database name |

### 3. Run Database Migrations

```bash
make migrate-up
```

This runs `AutoMigrate` for all 8 tables (in FK order): `tenants`, `tenant_settings`, `users`, `routers`, `profile_price_mappings`, `voucher_sales`, `print_templates`, `audit_logs`. Casbin policy table (`casbin_rule`) is auto-created on first API start.

### 4. Run the API Server

```bash
make run
# or directly: go run ./cmd/api
```

The server is available at `http://localhost:8080`. On first startup, Casbin role policies are seeded (idempotent).

### 5. Bootstrap First Tenant + Owner

The `auth/setup` endpoint runs **once** when the database has zero users. It creates a tenant, an `owner` user, copies the global default print templates to the tenant, and returns a JWT.

```bash
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_name": "My Hotspot",
    "tenant_slug": "my-hotspot",
    "username":    "owner",
    "password":    "owner1234"
  }'
```

### 6. Login (subsequent sessions)

Login requires the tenant slug. Superadmin omits the `tenant` field.

```bash
# Tenant user
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"tenant":"my-hotspot","username":"owner","password":"owner1234"}'

# Superadmin (created via cmd/seed or manual SQL)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"root","password":"root1234"}'
```

---

## Docker (Development Stack)

```bash
make docker-up            # Build & run the entire stack
make migrate-up-docker    # Run database migrations inside the container
make docker-logs          # Tail API logs
make docker-down          # Stop the stack
make docker-clean         # Stop and remove volumes
```

---

## API Endpoints

All endpoints prefixed with `/api/v1/`. Protected endpoints require:
- `Authorization: Bearer <token>` header
- For **superadmin** acting on a specific tenant: also `X-Tenant-Slug: <tenant-slug>` header

### Auth (public)

| Method | Path | Description |
|---|---|---|
| POST | `/auth/setup` | Bootstrap first tenant + owner (one-time, only when DB has zero users) |
| POST | `/auth/login` | Login with `{tenant, username, password}` (omit `tenant` for superadmin) |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout (revoke refresh token + blacklist current access token) |

### Auth (authenticated, no tenant context required)

| Method | Path | Description |
|---|---|---|
| GET | `/auth/me` | Current logged-in user profile (includes tenant info) |
| PUT | `/auth/password` | Change password |

### Tenant self-management (owner via Casbin)

| Method | Path | Description |
|---|---|---|
| GET | `/tenant` | Tenant info + settings |
| PUT | `/tenant` | Update tenant name |
| GET | `/tenant/settings` | Hotspot settings |
| PUT | `/tenant/settings` | Update hotspot settings (currency, dns_name, idle_timeout, …) |
| POST | `/tenant/logo` | Upload tenant logo (PNG/JPEG/WebP, ≤1 MB) |
| GET | `/tenant/logo` | Fetch tenant logo binary |

### User management (owner + admin via Casbin)

| Method | Path | Description |
|---|---|---|
| GET | `/users` | List users in current tenant |
| POST | `/users` | Create user + assign Casbin role |
| GET | `/users/:id` | User details |
| PUT | `/users/:id` | Update user (re-assigns Casbin role if `role` changed) |
| DELETE | `/users/:id` | Soft delete user + revoke all Casbin roles |

### Templates (tenant-scoped, was `/routers/:id/templates`)

| Method | Path | Description |
|---|---|---|
| GET | `/templates` | List tenant templates |
| POST | `/templates` | Create template |
| GET | `/templates/:templateId` | Template details |
| PUT | `/templates/:templateId` | Update template |
| DELETE | `/templates/:templateId` | Delete template |
| POST | `/templates/render` | Render cached vouchers as HTML |
| POST | `/templates/seed-defaults` | Reset to copy of global defaults |

### Routers (tenant-scoped)

| Method | Path | Description |
|---|---|---|
| GET | `/routers` | List routers in current tenant |
| POST | `/routers` | Add router |
| GET | `/routers/:id` | Router details |
| PUT | `/routers/:id` | Update router config |
| DELETE | `/routers/:id` | Delete router |
| POST | `/routers/:id/test` | Test connection |
| POST | `/routers/migrate` | Import from legacy `config.php` (creates a tenant per session) |

### Hotspot, PPP, Network, System, Vouchers, Reports

All under `/routers/:id/*`. Tenant ownership of `:id` is validated by `RouterTenantMiddleware` before any handler runs (returns 404 if router belongs to another tenant). Endpoints are unchanged from previous version — see [docs/README.md](docs/README.md) for the full list and [planing.md](planing.md) for policy mappings per role.

### SSE (Server-Sent Events)

Same paths as before, all under `/routers/:id/sse/*` and `/routers/:id/logs/stream/*`. EventSource passes the JWT via `?token=` query param (cannot set `Authorization` header).

### Events (RouterOS Webhooks, public)

| Method | Path | Description |
|---|---|---|
| POST | `/events/on-login` | RouterOS on-login form callback. Tenant resolved by `X-Router-Token` (or `token` form field), which must match `tenant_settings.webhook_token`. |
| GET | `/events/health` | Webhook health check |

### Public status check

| Method | Path | Description |
|---|---|---|
| GET | `/status` | User session check (used by login page; requires tenant context via auth) |

### Platform admin (superadmin only via Casbin policy `superadmin, *, /api/v1/*, *`)

| Method | Path | Description |
|---|---|---|
| GET | `/admin/tenants` | List all tenants |
| POST | `/admin/tenants` | Create tenant + settings + copy global templates |
| GET | `/admin/tenants/:id` | Tenant details |
| PUT | `/admin/tenants/:id` | Update tenant name |
| DELETE | `/admin/tenants/:id` | **Hard delete** — cascades to all child rows |
| POST | `/admin/tenants/:id/suspend` | Set status to suspended |
| POST | `/admin/tenants/:id/activate` | Restore active status |
| GET | `/admin/templates` | List global default templates (`tenant_id IS NULL`) |
| POST | `/admin/templates` | Create global default |
| PUT | `/admin/templates/:templateId` | Update global default |
| DELETE | `/admin/templates/:templateId` | Delete global default |

> Superadmin acting **inside** a specific tenant (e.g. `GET /routers`) sets `X-Tenant-Slug: <slug>`. Without the header, requests are scoped to the platform (`__platform__`).

---

## Makefile

```bash
make build              # Build all binaries (api, migrate, worker, seed) to bin/
make run                # Build & run API server
make test               # Run Go tests: go test -race ./...
make migrate-up         # Run schema migration (local binary)
make migrate-import     # Import legacy config: CONFIG_FILE=/path/to/config.php
make sync-profiles      # Sync profiles from router to DB
make docker-up          # Build & run Docker stack
make docker-down        # Stop Docker stack
make docker-logs        # Tail API container logs
make docker-clean       # Stop stack & remove volumes
make migrate-up-docker  # Run schema migration inside container
make curl-health        # Check health endpoint
make curl-setup         # Bootstrap first tenant via cURL
make curl-login         # Login via cURL
```

---

## Testing

```bash
# Go unit & integration tests
make test

# HTTP Integration tests (requires active router + active tenant)
TEST_TENANT=my-hotspot \
TEST_USERNAME=owner \
TEST_PASSWORD=owner1234 \
ROUTER_ID=1 \
ROUTER_IP=192.168.1.1 \
ROUTER_PASSWORD=your_password \
  python -m pytest tests/http/ -v --tb=short
```

See [tests/http/README.md](tests/http/README.md) for the full test guide.

---

## Breaking Changes from Single-Tenant Version

- **JWT tokens are no longer compatible** — all users must re-login (claims now include `tid` + `tslug`).
- **Login payload requires `tenant` field** for non-superadmin users.
- **Setup payload requires `tenant_name` + `tenant_slug`** (in addition to `username` + `password`).
- **Templates moved** from `/routers/:id/templates` to `/templates` (tenant-scoped, not router-scoped).
- **Logo moved** from `/routers/:id/logo` to `/tenant/logo` (one logo per tenant, not per router).
- **`HotspotConfig` table removed** — fields merged into `tenant_settings`.
- **`SystemUser` renamed** to `users` with composite uniqueness `(tenant_id, username)`.

---

## License

Private repository.
