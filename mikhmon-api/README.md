# mikhmon-api

Go backend untuk manajemen hotspot MikroTik. Menyediakan REST API untuk mengelola user hotspot, profil, voucher, laporan penjualan, dan monitoring perangkat RouterOS secara real-time.

**Stack**: Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · InfluxDB 3  
**Module**: `github.com/quiqxiq/roskit`

---

## Prasyarat

- Go 1.24+
- PostgreSQL 16
- Redis 7
- InfluxDB 3 Core (opsional — untuk metrik time-series)
- Docker & Docker Compose (opsional)

---

## Setup

### 1. Clone & install dependencies

```bash
git clone <repo-url>
cd mikhmon-api
go mod download
```

### 2. Buat file `.env`

```bash
cp .env.example .env
```

| Variable | Deskripsi |
|---|---|
| `JWT_SECRET` | Secret JWT access token (min 32 byte) |
| `JWT_REFRESH_SECRET` | Secret JWT refresh token (min 32 byte) |
| `AES_ENCRYPTION_KEY` | Kunci AES-256 GCM (base64, 32 byte) |
| `DB_PASSWORD` | Password PostgreSQL |
| `REDIS_PASSWORD` | Password Redis (kosongkan jika tidak pakai) |
| `INFLUXDB_URL` | URL InfluxDB (kosongkan untuk disable) |
| `INFLUXDB_TOKEN` | Token InfluxDB |
| `INFLUXDB_DATABASE` | Nama database InfluxDB (default: `mikhmon`) |

### 3. Jalankan migration

```bash
make migrate-up
```

### 4. Jalankan API server

```bash
make run
# atau: go run ./cmd/api
```

Server berjalan di `http://localhost:8080`.

### 5. Buat admin pertama

```bash
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'
```

---

## Docker

```bash
make docker-up         # Jalankan seluruh stack (PostgreSQL + Redis + InfluxDB + API)
make migrate-up-docker # Jalankan migration di dalam container
make docker-logs       # Lihat log API
make docker-down       # Stop semua container
make docker-clean      # Stop + hapus volumes
```

---

## Arsitektur

```
RouterOS (TCP 8728)
    │
    ▼
execution.Pool          ← koneksi TCP persistent, health check, reconnect
    │
    ├── behavior/stream  ← =follow streaming (real-time push dari RouterOS)
    └── behavior/poll    ← periodic snapshot (ticker-based)
    │
    ▼
pipeline/event.Processor
    ├── pipeline/cache    → Redis HSET snapshot + secondary index + TTL 5m
    ├── pipeline/timeseries → InfluxDB batch write (500 points / 5s flush)
    └── pipeline/pubsub   → Redis PUBLISH → SSE ke frontend
    │
    ▼
roskit/adapter/service.Bridge
    │
    ▼
internal/services + internal/api/handlers
    │
    ▼
REST API + SSE (Gin, port 8080)
```

### Command Registry

Semua perintah RouterOS terdaftar di `internal/roskit/core/definition/` via `init()`:

| Tipe | Mekanisme | Contoh |
|------|-----------|--------|
| `Stream` | `=follow` — RouterOS push real-time | hotspot/user, dhcp-server/lease, ppp/active |
| `Poll` | Ticker periodic | system/resource (60s), system/identity (5m) |
| `Query` | On-demand | hotspot/user/get, hotspot/user/find |
| `Mutation` | Write ke router | hotspot/user/add, hotspot/user/remove |
| `Action` | One-shot | system/reboot, system/shutdown |

Engine otomatis memulai semua stream dan poll yang terdaftar saat router ditambahkan.

---

## API Endpoints

Semua endpoint di bawah `/api/v1/`. Route protected memerlukan `Authorization: Bearer <token>`.

| Group | Endpoint | Deskripsi |
|---|---|---|
| Auth | `POST /auth/setup` | Buat admin pertama |
| Auth | `POST /auth/login` | Login |
| Auth | `POST /auth/refresh` | Refresh token |
| Auth | `POST /auth/logout` | Logout |
| Auth | `GET /auth/me` | Profil user |
| Auth | `PUT /auth/password` | Ganti password |
| Routers | `GET/POST /routers` | List & tambah router |
| Routers | `GET/PUT/DELETE /routers/:id` | Detail, update, hapus |
| Routers | `POST /routers/:id/test` | Test koneksi |
| Routers | `POST /routers/migrate` | Import dari config.php |
| Hotspot | `GET/POST /routers/:id/hotspot/users` | CRUD user |
| Hotspot | `GET/POST /routers/:id/hotspot/profiles` | CRUD profil |
| Hotspot | `GET/DELETE /routers/:id/hotspot/active` | Sesi aktif |
| Hotspot | `POST /routers/:id/hotspot/active/:id/disconnect` | Disconnect user |
| Hotspot | `GET/DELETE /routers/:id/hotspot/hosts` | Hosts |
| Hotspot | `GET /routers/:id/hotspot/servers` | Servers |
| Hotspot | `GET/DELETE /routers/:id/hotspot/cookies` | Cookies |
| Hotspot | `GET/POST/PUT/DELETE /routers/:id/hotspot/bindings` | IP Binding CRUD |
| Vouchers | `POST /routers/:id/vouchers/generate` | Generate batch voucher |
| Vouchers | `POST /routers/:id/vouchers/sales` | Catat penjualan |
| Vouchers | `GET /routers/:id/vouchers/print-data` | Data cetak voucher |
| Reports | `GET /routers/:id/reports/daily` | Laporan harian |
| Reports | `GET /routers/:id/reports/monthly` | Laporan bulanan |
| Reports | `GET /routers/:id/reports/resume` | Resume report |
| Reports | `GET /routers/:id/reports/summary` | Dashboard summary |
| Reports | `GET /routers/:id/reports/export/{csv,excel}` | Export laporan |
| System | `GET /routers/:id/system/resource` | CPU, RAM, uptime |
| System | `GET /routers/:id/system/resource/history` | History dari InfluxDB |
| System | `GET /routers/:id/system/log` | System log |
| System | `GET /routers/:id/system/clock` | Jam router |
| System | `GET /routers/:id/system/identity` | Identity router |
| System | `GET /routers/:id/system/routerboard` | Info hardware |
| System | `GET /routers/:id/system/dashboard` | Dashboard info |
| System | `GET /routers/:id/system/expire-monitor` | Status expire monitor |
| System | `POST /routers/:id/system/expire-monitor/deploy` | Deploy expire monitor |
| System | `POST /routers/:id/system/reboot` | Reboot router |
| PPP | `GET/POST /routers/:id/ppp/secrets` | PPP Secret CRUD |
| PPP | `GET /routers/:id/ppp/active` | PPP active sessions |
| PPP | `GET /routers/:id/ppp/profiles` | PPP profiles |
| Network | `GET /routers/:id/network/interfaces` | Interface list |
| Network | `GET /routers/:id/network/traffic/:iface` | Monitor traffic |
| Network | `GET /routers/:id/network/dhcp/leases` | DHCP leases |
| Network | `DELETE /routers/:id/network/dhcp/:id/release` | Release DHCP lease |
| Quick Print | `GET/POST/PUT/DELETE /routers/:id/quick-print` | Quick print CRUD |
| Templates | `GET/POST/PUT/DELETE /routers/:id/templates` | Template voucher CRUD |
| Templates | `POST /routers/:id/templates/render` | Render template |
| SSE Logs | `GET /routers/:id/logs/stream/all` | Real-time log stream |
| SSE Logs | `GET /routers/:id/logs/stream/hotspot` | Log hotspot |
| SSE Logs | `GET /routers/:id/logs/stream/ppp` | Log PPP |
| SSE Telemetry | `GET /routers/:id/sse/hotspot/users` | Real-time user updates |
| SSE Telemetry | `GET /routers/:id/sse/hotspot/active` | Active sessions |
| SSE Telemetry | `GET /routers/:id/sse/system/resource` | System resource |
| SSE Telemetry | `GET /routers/:id/sse/network/traffic/:iface` | Interface traffic |
| Events | `POST /events/on-login` | RouterOS on-login webhook |
| Events | `GET /events/health` | Health check |

---

## Makefile

```bash
make build        # Build semua binary
make run          # Build & jalankan API
make test         # go test -race ./...
make docker-up    # Jalankan dev stack
make docker-logs  # Lihat log API
make migrate-up   # Schema migration
```

---

## License

Private repository.
