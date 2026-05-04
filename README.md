# Roskit

A modern Go backend for MikroTik hotspot management. Provides REST APIs and real-time SSE (Server-Sent Events) for managing hotspot users, PPP secrets, vouchers, sales reports, and RouterOS monitoring.

**Stack**: Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · InfluxDB 3  
**Module**: `github.com/quiqxiq/roskit`

For detailed technical architecture and developer guidelines, see [docs/README.md](docs/README.md).

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

Create the `.env` file:
```bash
cp .env.example .env
```

Configure `.env` according to your environment:

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

> Note: The Docker Dev stack defaults use `mikhmon` for the database names, user, and password for backward compatibility with older setups.

### 3. Run Database Migrations

```bash
make migrate-up
```

### 4. Run the API Server

```bash
make run
# or directly: go run ./cmd/api
```

The server will be available at `http://localhost:8080`.

### 5. Create Initial Admin User

```bash
make curl-setup
# or manually:
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'
```

---

## Docker (Development Stack)

The development stack runs via `docker/docker-compose.dev.yml` and includes the API, PostgreSQL, Redis, and InfluxDB.

```bash
make docker-up            # Build & run the entire stack
make migrate-up-docker    # Run database migrations inside the container
make docker-logs          # Tail API logs
make docker-down          # Stop the stack
make docker-clean         # Stop and remove volumes
```

---

## API Endpoints

All endpoints are prefixed with `/api/v1/`. Protected endpoints require the `Authorization: Bearer <token>` header.

### Auth

| Method | Path | Description |
|---|---|---|
| POST | `/auth/setup` | Create the initial admin (one-time use) |
| POST | `/auth/login` | Login, retrieve access + refresh tokens |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout (invalidates refresh token) |
| GET | `/auth/me` | Current logged-in user profile |
| PUT | `/auth/password` | Change password |

### Router

| Method | Path | Description |
|---|---|---|
| GET | `/routers` | List all routers |
| POST | `/routers` | Add a new router |
| GET | `/routers/:id` | Router details |
| PUT | `/routers/:id` | Update router config |
| DELETE | `/routers/:id` | Delete router |
| POST | `/routers/:id/test` | Test connection to router |
| POST | `/routers/migrate` | Import from legacy `config.php` |

### Hotspot

| Method | Path | Description |
|---|---|---|
| GET/POST | `/routers/:id/hotspot/users` | List & add hotspot users |
| GET/PUT/DELETE | `/routers/:id/hotspot/users/:uid` | Details, update, delete user |
| GET/POST | `/routers/:id/hotspot/profiles` | List & add profiles |
| GET/PUT/DELETE | `/routers/:id/hotspot/profiles/:pid` | Details, update, delete profile |
| GET | `/routers/:id/hotspot/active` | Current active sessions |
| DELETE | `/routers/:id/hotspot/active/:aid` | Disconnect active session |
| GET | `/routers/:id/hotspot/hosts` | Connected hosts |
| GET | `/routers/:id/hotspot/servers` | Hotspot servers |
| GET/DELETE | `/routers/:id/hotspot/cookies` | Hotspot cookies |
| GET/POST/PUT/DELETE | `/routers/:id/hotspot/bindings` | IP Binding CRUD |

### PPP

| Method | Path | Description |
|---|---|---|
| GET/POST | `/routers/:id/ppp/secrets` | List & add PPP secret |
| GET/PUT/DELETE | `/routers/:id/ppp/secrets/:sid` | Details, update, delete |
| GET | `/routers/:id/ppp/active` | Active PPP sessions |
| GET | `/routers/:id/ppp/profiles` | PPP profiles |

### Network

| Method | Path | Description |
|---|---|---|
| GET | `/routers/:id/network/interfaces` | List interfaces |
| GET | `/routers/:id/network/traffic/:iface` | Monitor interface traffic |
| GET | `/routers/:id/network/dhcp/leases` | DHCP leases |
| DELETE | `/routers/:id/network/dhcp/:lid/release` | Release lease |

### System

| Method | Path | Description |
|---|---|---|
| GET | `/routers/:id/system/resource` | CPU, RAM, uptime |
| GET | `/routers/:id/system/resource/history` | Historical metrics from InfluxDB |
| GET | `/routers/:id/system/log` | System logs |
| GET | `/routers/:id/system/clock` | Router clock |
| GET | `/routers/:id/system/identity` | Router identity |
| GET | `/routers/:id/system/routerboard` | Hardware info |
| GET | `/routers/:id/system/dashboard` | Dashboard summary data |
| POST | `/routers/:id/system/reboot` | Reboot router |
| GET | `/routers/:id/system/expire-monitor` | Expire monitor status |
| POST | `/routers/:id/system/expire-monitor/deploy` | Deploy expire monitor script |

### Vouchers

| Method | Path | Description |
|---|---|---|
| POST | `/routers/:id/vouchers/generate` | Generate batch vouchers |
| POST | `/routers/:id/vouchers/sales` | Record sale |
| GET | `/routers/:id/vouchers/print-data` | Data for printing vouchers |

### Reports

| Method | Path | Description |
|---|---|---|
| GET | `/routers/:id/reports/daily` | Daily report |
| GET | `/routers/:id/reports/monthly` | Monthly report |
| GET | `/routers/:id/reports/resume` | Sales resume |
| GET | `/routers/:id/reports/summary` | Dashboard summary |
| GET | `/routers/:id/reports/export/csv` | Export as CSV |
| GET | `/routers/:id/reports/export/excel` | Export as Excel |

### Quick Print & Templates

| Method | Path | Description |
|---|---|---|
| GET/POST/PUT/DELETE | `/routers/:id/quick-print` | Quick print packages |
| GET/POST/PUT/DELETE | `/routers/:id/templates` | Voucher templates |
| POST | `/routers/:id/templates/render` | Render template |

### SSE (Server-Sent Events)

| Path | Data |
|---|---|
| `/routers/:id/sse/hotspot/users` | Real-time hotspot user changes |
| `/routers/:id/sse/hotspot/active` | Real-time active sessions |
| `/routers/:id/sse/system/resource` | Real-time CPU/RAM/uptime |
| `/routers/:id/sse/network/traffic/:iface` | Real-time interface bandwidth |
| `/routers/:id/logs/stream/all` | System log stream |
| `/routers/:id/logs/stream/hotspot` | Hotspot log stream |
| `/routers/:id/logs/stream/ppp` | PPP log stream |

### Events (RouterOS Webhooks)

| Method | Path | Description |
|---|---|---|
| POST | `/events/on-login` | RouterOS on-login webhook |
| GET | `/events/health` | Webhook health check |

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
make curl-setup         # Create initial admin via cURL
make curl-login         # Login via cURL
```

---

## Testing

```bash
# Go unit & integration tests
make test

# HTTP Integration tests (requires active router)
TEST_PASSWORD=admin1234 \
ROUTER_ID=1 \
ROUTER_IP=192.168.1.1 \
ROUTER_PASSWORD=your_password \
  python -m pytest tests/http/ -v --tb=short
```

---

## License

Private repository.
