# Roskit

A modern Go backend for MikroTik hotspot management. Provides REST APIs and real-time SSE for managing hotspot users, PPP secrets, vouchers, sales reports, and RouterOS monitoring.

**Stack**: Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · InfluxDB 3
**Module**: `github.com/quiqxiq/roskit`

For detailed technical architecture and developer guidelines, see [docs/README.md](docs/README.md).

---

## Roles

Two roles control access:

| Role | Permissions |
|---|---|
| `admin` | Full access: settings, users, routers, templates, vouchers, reports. |
| `staff` | Operational only — hotspot users, vouchers, reports (read), templates (read). |

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

This runs `AutoMigrate` for all 7 tables: `settings`, `users`, `routers`, `profile_price_mappings`, `voucher_sales`, `print_templates`, `audit_logs`.

### 4. Run the API Server

```bash
make run
# or directly: go run ./cmd/api
```

The server is available at `http://localhost:8080`.

### 5. Bootstrap Initial Admin User

The `auth/setup` endpoint runs **once** when the database has zero users. It creates an admin user and returns a JWT.

```bash
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin1234"
  }'
```

### 6. Login (subsequent sessions)

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'
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

All endpoints prefixed with `/api/v1/`. Protected endpoints require `Authorization: Bearer <token>` header.

### Auth (public)

| Method | Path | Description |
|---|---|---|
| POST | `/auth/setup` | Bootstrap first admin user (one-time, only when DB has zero users) |
| POST | `/auth/login` | Login with `{username, password}` |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout (revoke refresh token + blacklist current access token) |

### Auth (authenticated)

| Method | Path | Description |
|---|---|---|
| GET | `/auth/me` | Current logged-in user profile |
| PUT | `/auth/password` | Change password |

### Settings (admin only)

| Method | Path | Description |
|---|---|---|
| GET | `/settings` | Application settings |
| PUT | `/settings` | Update settings (hotspot_name, currency, dns_name, idle_timeout, …) |
| POST | `/settings/logo` | Upload logo (PNG/JPEG/WebP, ≤1 MB) |
| GET | `/settings/logo` | Fetch logo binary |

### User management (admin only)

| Method | Path | Description |
|---|---|---|
| GET | `/users` | List all users |
| POST | `/users` | Create user |
| GET | `/users/:id` | User details |
| PUT | `/users/:id` | Update user |
| DELETE | `/users/:id` | Soft delete user |

### Templates (all authenticated)

| Method | Path | Description |
|---|---|---|
| GET | `/templates` | List templates |
| POST | `/templates` | Create template |
| GET | `/templates/:templateId` | Template details |
| PUT | `/templates/:templateId` | Update template |
| DELETE | `/templates/:templateId` | Delete template |
| POST | `/templates/render` | Render cached vouchers as HTML |
| POST | `/templates/seed-defaults` | Seed default templates |

### Routers (all authenticated)

| Method | Path | Description |
|---|---|---|
| GET | `/routers` | List routers |
| POST | `/routers` | Add router |
| GET | `/routers/:routerId` | Router details |
| PUT | `/routers/:routerId` | Update router config |
| DELETE | `/routers/:routerId` | Delete router |
| POST | `/routers/:routerId/test` | Test connection |
| POST | `/routers/migrate` | Import from legacy `config.php` |

### Hotspot, PPP, Network, System, Vouchers, Reports

All under `/routers/:routerId/*`. Ownership of `:routerId` is validated by `RouterOwnershipMiddleware` before any handler runs (returns 404 if router not found). See [docs/README.md](docs/README.md) for the full list.

### SSE (Server-Sent Events)

Paths under `/routers/:routerId/sse/*` and `/routers/:routerId/logs/stream/*`. EventSource passes the JWT via `?token=` query param (cannot set `Authorization` header).

### Events (RouterOS Webhooks, public)

| Method | Path | Description |
|---|---|---|
| POST | `/events/on-login` | RouterOS on-login form callback |
| GET | `/events/health` | Webhook health check |

### Public status check

| Method | Path | Description |
|---|---|---|
| GET | `/status` | User session check (used by login page) |

---

## Makefile

```bash
make build              # Build all binaries (api, migrate, worker) to bin/
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
make curl-setup         # Bootstrap first admin via cURL
make curl-login         # Login via cURL
```

---

## Testing

```bash
# Go unit & integration tests
make test

# HTTP Integration tests (requires active router)
TEST_USERNAME=admin \
TEST_PASSWORD=admin1234 \
ROUTER_ID=1 \
ROUTER_IP=192.168.1.1 \
ROUTER_PASSWORD=your_password \
  python -m pytest tests/http/ -v --tb=short
```

See [tests/http/README.md](tests/http/README.md) for the full test guide.

---

## License

Private repository.
