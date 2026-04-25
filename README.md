# mikhmon-api

Go backend untuk manajemen hotspot MikroTik. Menyediakan REST API dan SSE real-time untuk mengelola user hotspot, PPP secret, voucher, laporan penjualan, dan monitoring RouterOS.

**Stack**: Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · InfluxDB 3  
**Module**: `github.com/quiqxiq/roskit`

---

## Prasyarat

| Kebutuhan | Versi |
|---|---|
| Go | 1.24+ |
| PostgreSQL | 16 |
| Redis | 7 |
| InfluxDB 3 Core | opsional (time-series metrics) |
| Docker & Compose | opsional |

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

Edit `.env` sesuai environment:

| Variable | Keterangan |
|---|---|
| `JWT_SECRET` | Secret JWT access token (min 32 byte) |
| `JWT_REFRESH_SECRET` | Secret JWT refresh token (min 32 byte) |
| `AES_ENCRYPTION_KEY` | Kunci AES-256 GCM untuk enkripsi password router (base64, 32 byte) |
| `DB_PASSWORD` | Password PostgreSQL |
| `REDIS_PASSWORD` | Password Redis (kosongkan jika tanpa auth) |
| `INFLUXDB_URL` | URL InfluxDB (kosongkan untuk disable time-series) |
| `INFLUXDB_TOKEN` | Token InfluxDB |
| `INFLUXDB_DATABASE` | Nama database InfluxDB (default: `mikhmon`) |

### 3. Jalankan migration

```bash
make migrate-up
```

### 4. Jalankan API server

```bash
make run
# atau langsung: go run ./cmd/api
```

Server berjalan di `http://localhost:8080`.

### 5. Buat admin pertama

```bash
make curl-setup
# atau manual:
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'
```

---

## Docker (Development)

```bash
make docker-up            # Build & jalankan seluruh stack
make migrate-up-docker    # Jalankan migration di dalam container
make docker-logs          # Tail log API
make docker-down          # Stop stack
make docker-clean         # Stop + hapus volumes
```

Stack dev menggunakan `docker/docker-compose.dev.yml` yang menjalankan:
- `api` — aplikasi ini (port 8080)
- `postgres` — PostgreSQL 16 (port 5432)
- `redis` — Redis 7 (port 6379)
- `influxdb` — InfluxDB 3 Core (port 8181)

---

## Struktur Proyek

```
mikhmon-api/
├── cmd/
│   ├── api/          — entry point HTTP server
│   ├── migrate/      — CLI migration & import config.php
│   └── worker/       — background worker (opsional)
│
├── internal/
│   ├── api/
│   │   └── handlers/ — Gin route handlers (auth, hotspot, ppp, network, system, dll)
│   │       └── sse/  — SSE handlers (telemetry, log streaming)
│   │
│   ├── roskit/       — inti engine RouterOS
│   │   ├── core/
│   │   │   ├── command/    — CommandMeta, registry, builder
│   │   │   ├── definition/ — registrasi semua command RouterOS (per domain)
│   │   │   ├── model/      — domain model (HotspotUser, Interface, dll)
│   │   │   └── parser/     — parsing reply RouterOS → domain model
│   │   │
│   │   ├── execution/      — TCP connection pool ke router
│   │   ├── orchestrator/   — engine startup, dispatcher, router manager
│   │   ├── behavior/
│   │   │   ├── stream/     — stream workers (=follow real-time push)
│   │   │   ├── poll/       — poll workers (ticker periodic)
│   │   │   ├── query/      — on-demand get/find
│   │   │   └── mutation/   — write commands (add/set/remove/dll)
│   │   ├── pipeline/
│   │   │   ├── event/      — event processor (sink utama)
│   │   │   ├── cache/      — Redis snapshot & secondary index
│   │   │   ├── pubsub/     — Redis pub-sub untuk SSE
│   │   │   └── timeseries/ — InfluxDB writer & reader
│   │   └── adapter/
│   │       └── service/    — Bridge: menghubungkan services ke orchestrator
│   │
│   └── services/           — business logic (hotspot, router, voucher, report, dll)
│
├── migrations/             — SQL migration files
├── docs/
│   ├── mikrotik/           — routeros_7.20.8.json (API spec)
│   └── routeros-commands.md — referensi command registry
├── tests/http/             — Python integration tests (pytest)
├── docker/                 — docker-compose files & Dockerfile
└── Makefile
```

---

## Arsitektur

### Gambaran Besar

```
RouterOS (TCP 8728)
        │
        ▼
┌─────────────────────────────────────────────────────┐
│  execution.Pool                                     │
│   ├── sync conn  — poll / query / mutation          │
│   └── async conn — semua stream workers (tag-mux)   │
└─────────────────────────────────────────────────────┘
        │
        ├── behavior/stream   — =follow, push per event
        ├── behavior/poll     — ticker periodic snapshot
        ├── behavior/query    — on-demand get/find
        └── behavior/mutation — add/set/remove/enable/disable
        │
        ▼
┌─────────────────────────────────────────────────────┐
│  pipeline/event.Processor  (StreamSink & PollSink)  │
│   ├── pipeline/cache    → Redis HSET snapshot       │
│   ├── pipeline/timeseries → InfluxDB (telemetry)    │
│   └── pipeline/pubsub   → Redis PUBLISH → SSE       │
└─────────────────────────────────────────────────────┘
        │
        ▼
roskit/adapter/service.Bridge
        │
        ▼
internal/services  +  internal/api/handlers
        │
        ▼
REST API + SSE  (Gin, port 8080)
```

### Koneksi TCP per Router

Setiap router menggunakan **tepat 2 koneksi TCP**:

| Koneksi | Digunakan untuk |
|---|---|
| `sync` | poll, query, mutation — dieksekusi satu per satu |
| `async` | semua stream workers — tag-multiplexed via `client.Async()` |

Stream workers tidak membuka koneksi sendiri-sendiri; semuanya berbagi satu `async` connection. Ini menjaga jumlah koneksi jauh di bawah limit RouterOS (default 20).

### Command Registry

Semua command RouterOS didefinisikan di `internal/roskit/core/definition/` menggunakan helper constructors. Engine membaca registry ini saat router ditambahkan dan otomatis memulai workers yang sesuai.

```go
// Stream: RouterOS push setiap ada perubahan
command.Register(command.StreamDef("ip/hotspot/active/print", "", "hotspot_active"))

// Poll: dieksekusi periodik via ticker
command.Register(command.PollDef("system/resource/print", 60*time.Second))

// Mutation: write ke router
command.Register(command.MutationDef("ip/hotspot/user/add"))

// Query: on-demand, short-cached
command.Register(command.QueryDef("ip/hotspot/user/find"))
```

**Aturan Stream vs Poll**:

| Verdict | Kondisi | Contoh |
|---|---|---|
| **Stream** | Data berubah sering/intens, follow=true | `hotspot/active`, `ip/route`, `firewall/connection` |
| **Poll** | follow=true tapi admin-only config, jarang berubah | `firewall/filter`, `interface/vlan`, `system/ntp` |
| **Poll** | Tidak punya follow, hanya interval | `system/resource`, `system/health` |

Lihat `docs/routeros-commands.md` untuk referensi lengkap semua command yang terdaftar.

### Pipeline Event

`event.Processor` menerima setiap event dari stream/poll dan meneruskannya ke:

1. **Redis cache** — snapshot disimpan sebagai `HSET router:{id}:{measurement}:{entryID}`. Secondary index untuk O(1) lookup (misalnya `hotspot_user` diindex by `name`).
2. **InfluxDB** — hanya untuk 3 measurement telemetry: `system_resource`, `interface_traffic`, `hotspot_active`.
3. **Redis pub-sub** — broadcast realtime untuk measurement yang punya frontend subscriber aktif: `hotspot_active`, `interface_traffic`, `system_resource`, `dhcp_lease`.

Cache di-invalidate eksplisit setelah setiap mutation berhasil, sehingga read berikutnya langsung mendapat data terbaru dari router.

---

## Command Domains

| File | Domain | Tipe |
|---|---|---|
| `definition/hotspot.go` | `ip/hotspot/*` | Stream + Mutation |
| `definition/network.go` | `ip/address`, `ip/route`, `ip/arp`, `ip/dhcp-*`, `ip/dns`, `ip/pool`, `queue/simple` | Stream + Poll |
| `definition/ppp.go` | `ppp/secret`, `ppp/active`, `ppp/profile` | Stream + Mutation |
| `definition/firewall.go` | `ip/firewall/filter`, `address-list`, `mangle`, `raw`, `connection` | Stream + Poll |
| `definition/interface.go` | `interface/vlan`, `bridge`, `ethernet`, `wireless`, `wifi`, `wireguard`, `pppoe-client`, `l2tp-client` | Stream + Poll |
| `definition/queue.go` | `queue/tree`, `queue/interface`, `queue/type` | Poll |
| `definition/routing.go` | `routing/route`, `routing/ospf/*`, `routing/bgp/*` | Stream + Poll |
| `definition/system.go` | `system/resource`, `system/health`, `system/scheduler`, `system/script`, `system/logging`, `system/ntp` | Poll + Stream |
| `definition/user.go` | `user`, `user/active`, `user/group` | Stream + Poll |
| `definition/radius.go` | `radius` | Poll |
| `definition/ipsec.go` | `ip/ipsec/peer`, `ip/ipsec/active-peers`, `ip/ipsec/policy`, `ip/ipsec/installed-sa` | Stream + Poll |
| `definition/user_manager.go` | `user-manager/user`, `user-manager/session`, `user-manager/profile`, `user-manager/router` | Stream + Poll |
| `definition/tools.go` | `tool/netwatch` | Stream |

---

## API Endpoints

Semua endpoint di bawah `/api/v1/`. Endpoint terproteksi memerlukan header `Authorization: Bearer <token>`.

### Auth

| Method | Path | Deskripsi |
|---|---|---|
| POST | `/auth/setup` | Buat admin pertama (sekali pakai) |
| POST | `/auth/login` | Login, dapat access + refresh token |
| POST | `/auth/refresh` | Perbarui access token |
| POST | `/auth/logout` | Logout (invalidate refresh token) |
| GET | `/auth/me` | Profil user login |
| PUT | `/auth/password` | Ganti password |

### Router

| Method | Path | Deskripsi |
|---|---|---|
| GET | `/routers` | Daftar semua router |
| POST | `/routers` | Tambah router baru |
| GET | `/routers/:id` | Detail router |
| PUT | `/routers/:id` | Update config router |
| DELETE | `/routers/:id` | Hapus router |
| POST | `/routers/:id/test` | Test koneksi ke router |
| POST | `/routers/migrate` | Import dari `config.php` mikhmon lama |

### Hotspot

| Method | Path | Deskripsi |
|---|---|---|
| GET/POST | `/routers/:id/hotspot/users` | List & tambah user |
| GET/PUT/DELETE | `/routers/:id/hotspot/users/:uid` | Detail, update, hapus user |
| GET/POST | `/routers/:id/hotspot/profiles` | List & tambah profil |
| GET/PUT/DELETE | `/routers/:id/hotspot/profiles/:pid` | Detail, update, hapus profil |
| GET | `/routers/:id/hotspot/active` | Sesi aktif saat ini |
| DELETE | `/routers/:id/hotspot/active/:aid` | Disconnect sesi |
| GET | `/routers/:id/hotspot/hosts` | Hotspot hosts |
| GET | `/routers/:id/hotspot/servers` | Hotspot servers |
| GET/DELETE | `/routers/:id/hotspot/cookies` | Cookies |
| GET/POST/PUT/DELETE | `/routers/:id/hotspot/bindings` | IP Binding CRUD |

### PPP

| Method | Path | Deskripsi |
|---|---|---|
| GET/POST | `/routers/:id/ppp/secrets` | List & tambah PPP secret |
| GET/PUT/DELETE | `/routers/:id/ppp/secrets/:sid` | Detail, update, hapus |
| GET | `/routers/:id/ppp/active` | Sesi PPP aktif |
| GET | `/routers/:id/ppp/profiles` | PPP profiles |

### Network

| Method | Path | Deskripsi |
|---|---|---|
| GET | `/routers/:id/network/interfaces` | Daftar interface |
| GET | `/routers/:id/network/traffic/:iface` | Monitor traffic interface |
| GET | `/routers/:id/network/dhcp/leases` | DHCP leases |
| DELETE | `/routers/:id/network/dhcp/:lid/release` | Release lease |

### System

| Method | Path | Deskripsi |
|---|---|---|
| GET | `/routers/:id/system/resource` | CPU, RAM, uptime |
| GET | `/routers/:id/system/resource/history` | History dari InfluxDB |
| GET | `/routers/:id/system/log` | System log |
| GET | `/routers/:id/system/clock` | Jam router |
| GET | `/routers/:id/system/identity` | Identity router |
| GET | `/routers/:id/system/routerboard` | Info hardware |
| GET | `/routers/:id/system/dashboard` | Data ringkasan dashboard |
| POST | `/routers/:id/system/reboot` | Reboot router |
| GET | `/routers/:id/system/expire-monitor` | Status expire monitor |
| POST | `/routers/:id/system/expire-monitor/deploy` | Deploy expire monitor script |

### Voucher

| Method | Path | Deskripsi |
|---|---|---|
| POST | `/routers/:id/vouchers/generate` | Generate batch voucher |
| POST | `/routers/:id/vouchers/sales` | Catat penjualan |
| GET | `/routers/:id/vouchers/print-data` | Data untuk cetak voucher |

### Report

| Method | Path | Deskripsi |
|---|---|---|
| GET | `/routers/:id/reports/daily` | Laporan harian |
| GET | `/routers/:id/reports/monthly` | Laporan bulanan |
| GET | `/routers/:id/reports/resume` | Resume |
| GET | `/routers/:id/reports/summary` | Dashboard summary |
| GET | `/routers/:id/reports/export/csv` | Export CSV |
| GET | `/routers/:id/reports/export/excel` | Export Excel |

### Quick Print & Template

| Method | Path | Deskripsi |
|---|---|---|
| GET/POST/PUT/DELETE | `/routers/:id/quick-print` | Quick print packages |
| GET/POST/PUT/DELETE | `/routers/:id/templates` | Template voucher |
| POST | `/routers/:id/templates/render` | Render template |

### SSE (Server-Sent Events)

| Path | Data |
|---|---|
| `/routers/:id/sse/hotspot/users` | Perubahan user hotspot real-time |
| `/routers/:id/sse/hotspot/active` | Sesi aktif real-time |
| `/routers/:id/sse/system/resource` | CPU/RAM/uptime real-time |
| `/routers/:id/sse/network/traffic/:iface` | Bandwidth interface real-time |
| `/routers/:id/logs/stream/all` | System log stream |
| `/routers/:id/logs/stream/hotspot` | Log hotspot stream |
| `/routers/:id/logs/stream/ppp` | Log PPP stream |

### Events (Webhook RouterOS)

| Method | Path | Deskripsi |
|---|---|---|
| POST | `/events/on-login` | Hook on-login dari RouterOS |
| GET | `/events/health` | Health check |

---

## Makefile

```bash
make build              # Build semua binary (api, migrate, worker) ke bin/
make run                # Build & jalankan API server
make test               # go test -race ./...
make migrate-up         # Jalankan schema migration (binary lokal)
make migrate-import     # Import config.php: CONFIG_FILE=/path/to/config.php
make sync-profiles      # Sinkronisasi profil dari router ke database
make docker-up          # Build & jalankan dev stack
make docker-down        # Stop stack
make docker-logs        # Tail log container api
make docker-clean       # Stop + hapus volumes
make migrate-up-docker  # Migration di dalam container
make curl-health        # Cek health endpoint
make curl-setup         # Buat admin pertama via curl
make curl-login         # Login via curl
```

---

## Testing

```bash
# Unit & integration tests Go
make test

# Integration tests HTTP (butuh router aktif)
TEST_PASSWORD=admin1234 \
ROUTER_ID=1 \
ROUTER_IP=192.168.1.1 \
ROUTER_PASSWORD=your_password \
  python -m pytest tests/http/ -v --tb=short
```

---

## License

Private repository.
