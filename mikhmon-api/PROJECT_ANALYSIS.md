# Mikhmon API — Analisis Project

> Backend Go untuk manajemen hotspot MikroTik RouterOS. Migrasi dari PHP monolith ke Go (Gin) + PostgreSQL + Redis + InfluxDB.  
> Module: `github.com/quiqxiq/roskit`

---

## 1. Gambaran Umum

**Prinsip utama:** RouterOS adalah source of truth untuk semua data operasional jaringan. Backend hanya menyimpan: transaksi penjualan, konfigurasi router, user sistem, audit log, dan template cetak.

**Data flow:**

```
RouterOS ──TCP 8728──▶ execution.Pool
                            │
                   ┌────────┴────────┐
                   ▼                 ▼
           behavior/stream    behavior/poll
           (=follow push)     (ticker-based)
                   │                 │
                   └────────┬────────┘
                            ▼
                  pipeline/event.Processor
                  ┌──────────┬──────────────┐
                  ▼          ▼              ▼
            cache.Redis   timeseries   pubsub.Redis
           (HSET + TTL)   (InfluxDB)   (PUBLISH)
                                           │
                                           ▼
                                    SSE → Frontend
```

**Read path (Redis-first):**
```
Handler → service.Bridge → behavior/query.Handler
              ├── Redis cache (HGETALL/SCAN) — cache-first
              └── RouterOS API fallback (cold start)
```

---

## 2. Struktur Project

```
mikhmon-api/                          # 101 Go files, ~14.100+ baris
├── cmd/
│   ├── api/main.go                   # HTTP API server entry point
│   ├── migrate/main.go               # Database migration CLI
│   └── worker/main.go                # Background worker
│
├── internal/
│   ├── config/config.go              # Environment-based config
│   ├── models/                       # 6 GORM models
│   ├── repository/                   # Data access via GORM
│   ├── services/                     # Business logic (selesai)
│   ├── api/                          # HTTP layer (Gin)
│   ├── worker/worker.go              # Background tasks
│   │
│   └── roskit/                       # RouterOS subsystem (inti)
│       ├── core/
│       │   ├── command/              # CommandMeta, Registry, Builder
│       │   ├── definition/           # Registrasi semua command RouterOS
│       │   ├── model/                # Domain struct (hotspot, network, system)
│       │   └── parser/               # RouterOS response parsers
│       ├── behavior/
│       │   ├── interfaces.go         # StreamSink, PollSink, QueryHandler, dll
│       │   ├── stream/               # Stream manager, worker, log_worker, interface_monitor
│       │   ├── poll/                 # Poll scheduler + worker
│       │   ├── query/                # Cache-first query handler
│       │   └── mutation/             # Write command handler
│       ├── execution/
│       │   ├── pool.go               # TCP connection pool
│       │   ├── router.go             # Per-router connection lifecycle
│       │   └── executor.go           # Raw command executor
│       ├── orchestrator/
│       │   ├── engine.go             # Top-level engine (wires semua komponen)
│       │   ├── dispatcher.go         # Unified API: Stream/Query/Mutate/Run
│       │   └── router_manager.go     # AddRouter/RemoveRouter facade
│       ├── pipeline/
│       │   ├── cache/                # Redis HSET + index + TTL
│       │   ├── event/                # Processor (StreamSink + PollSink)
│       │   ├── pubsub/               # Redis Pub/Sub publisher + subscriber
│       │   └── timeseries/           # InfluxDB writer + reader
│       └── adapter/
│           ├── service/              # Bridge + hotspot/voucher/system/ppp/network/...
│           └── http/handler.go       # On-login webhook handler
│
├── migrations/migrations.go
├── pkg/
│   ├── database/postgres.go
│   ├── encrypt/encrypt.go
│   ├── errors/errors.go
│   └── redis/                        # Redis client + cache wrapper
└── docker/
    ├── Dockerfile
    ├── docker-compose.yml
    └── docker-compose.dev.yml
```

---

## 3. Lapisan Roskit (Inti)

### 3.1 core/command — Command Registry

Semua perintah RouterOS terdaftar sekali lewat `init()` di `core/definition/`.  
Engine memakai `command.ByType()` untuk memulai worker secara otomatis — tidak perlu daftar spec manual.

```go
// CommandType menentukan mekanisme eksekusi:
CommandTypeStream   // =follow: RouterOS push real-time
CommandTypePoll     // print tanpa follow: ticker periodic
CommandTypeQuery    // get/find: on-demand, short-cached
CommandTypeMutation // add/set/remove/enable/disable: write
CommandTypeAction   // reboot, shutdown, dll: one-shot
```

**CommandMeta** — single source of truth per path:
```go
type CommandMeta struct {
    Path              string        // e.g. "ip/hotspot/user/print"
    Type              CommandType
    SupportsFollow    bool
    SupportsInterval  bool
    PollInterval      time.Duration // untuk Poll
    CacheTTL          time.Duration // Redis TTL
    IndexField        string        // secondary index (e.g. "name")
    Measurement       string        // InfluxDB + Redis key segment
    Category          string        // "hotspot", "ppp", "system", dll
}
```

**Helper constructors:**
- `StreamDef(path, indexField, measurement)` — untuk =follow commands
- `PollDef(path, interval)` — untuk periodic snapshot
- `MutationDef(path)` — untuk write commands
- `QueryDef(path)` — untuk on-demand read
- `MonitorDef(path, measurement)` — untuk monitor-traffic

### 3.2 core/definition — Command Registrations

| File | Kategori | Isi |
|------|----------|-----|
| `hotspot.go` | hotspot | 7 Stream + 5 Mutation groups + 6 Query |
| `system.go` | system, interface | 4 Poll + 2 Stream + Mutations (scheduler, script, reboot) |
| `network.go` | network, dhcp, ppp | Stream (dhcp-lease, ppp) + Poll + Mutations |
| `ppp.go` | ppp | Stream + Mutations |

**Stream commands yang aktif (=follow):**

| Path | Measurement | Index |
|------|-------------|-------|
| `ip/hotspot/user/print` | `hotspot_user` | name |
| `ip/hotspot/user/profile/print` | `hotspot_profile` | name |
| `ip/hotspot/active/print` | `hotspot_active` | — |
| `ip/hotspot/print` | `hotspot_server` | name |
| `ip/hotspot/host/print` | `hotspot_host` | — |
| `ip/hotspot/cookie/print` | `hotspot_cookie` | — |
| `ip/hotspot/ip-binding/print` | `ip_binding` | — |
| `ip/dhcp-server/lease/print` | `dhcp_lease` | address |
| `ppp/secret/print` | `ppp_secret` | name |
| `ppp/active/print` | `ppp_active` | name |
| `system/scheduler/print` | `system_scheduler` | name |
| `system/script/print` | `system_script` | name |
| `interface/monitor-traffic` | `interface_traffic` | name |

**Poll commands (periodic):**

| Path | Interval | Measurement |
|------|----------|-------------|
| `system/resource/print` | 60s | `system_resource` |
| `system/identity/print` | 5m | `system_identity` |
| `system/clock/print` | 60s | `system_clock` |
| `system/health/print` | 30s | `system_health` |
| `system/routerboard/print` | 10m | `system_routerboard` |

### 3.3 execution — TCP Connection Pool

- `Pool` — manages `map[routerID]*RouterConn`
- `RouterConn` — states: `Disconnected → Connecting → Connected`
- Auto-reconnect dengan exponential backoff
- Health check setiap 30s via `/system/identity/print`
- `Borrow/Return` pattern untuk concurrent safe access

### 3.4 orchestrator — Engine + Dispatcher

**Engine** (`orchestrator/engine.go`):
```
Engine.Start(ctx)
  └── pool.Start(ctx) → connects all routers
  └── per router: startRouterWorkers()
        ├── command.ByType(Stream) → streams.Start(ctx, routerID, meta)
        └── command.ByType(Poll)   → polls.Start(ctx, routerID, meta)
```

**Dispatcher** — unified API untuk semua operasi RouterOS:
- `Stream(ctx, routerID, path)` — start/reuse stream
- `Query(ctx, routerID, path, filters...)` — Redis-first read
- `Mutate(ctx, routerID, path, args...)` — write ke router
- `Run(ctx, routerID, sentence...)` — raw command

### 3.5 pipeline/event — Processor

Implements `StreamSink` + `PollSink`. Menerima events dari behavior layer, menulis ke 3 storage:

**handleUpdate (stream event):**
1. `timeseries.WritePoint()` → InfluxDB batch buffer
2. `cache.SetSnapshot(key, fields, TTL)` → Redis `HSET` + `EXPIRE`
3. `cache.SetIndex(indexKey, name, cacheKey)` → Redis secondary index (jika IndexField != "")
4. `pubsub.Publish(channel, msg)` → Redis `PUBLISH`

**handleDead (=.dead event dari RouterOS):**
1. `cache.DeleteIndex()` → hapus secondary index entry
2. `cache.DeleteSnapshot()` → hapus snapshot
3. `pubsub.Publish()` → broadcast dead event

**OnPoll (poll event):**
1. Per-row: `timeseries.WritePoint()` + `cache.SetSnapshot()`
2. `pubsub.Publish()` → broadcast poll update

### 3.6 pipeline/cache — Redis Repository

**Key patterns:**

| Pattern | Format | TTL |
|---------|--------|-----|
| Entity snapshot | `roskit:{routerID}:{measurement}:{id}` | 5m (auto-refresh oleh stream) |
| Secondary index | `roskit:{routerID}:idx:{measurement}` | No TTL |
| Inactive list | `roskit:{routerID}:{hotspot\|ppp}_inactive` | No TTL |
| Pub/Sub channel | `roskit:telemetry:{routerID}` | — |

**Methods:** `SetSnapshot`, `GetSnapshot`, `DeleteSnapshot`, `ScanByMeasurement`, `CountByMeasurement`, `SetIndex`, `GetByIndex`, `DeleteIndex`

### 3.7 pipeline/timeseries — InfluxDB

- **Writer**: `InfluxWriter` — HTTP line protocol, buffer 500 points, flush setiap 5s
- **Reader**: `InfluxReader` — query historical data (untuk `/system/resource/history` dll)
- Noop implementations tersedia untuk dev tanpa InfluxDB

### 3.8 pipeline/pubsub — Redis Pub/Sub

- **Publisher** (`redis_pub.go`) — `PUBLISH` ke channel `roskit:telemetry:{routerID}`
- **Subscriber** (`redis_sub.go`) — `SUBSCRIBE`, forward ke SSE handlers
- **Message struct**: `{Type, RouterID, Measurement, EntityID, Fields, Timestamp}`

### 3.9 adapter/service — Bridge + Services

**Bridge** adalah satu-satunya entry point dari application layer ke RouterOS:

```go
type Bridge struct {
    dispatcher *orchestrator.Dispatcher
    cache      cache.Repository
}
// Methods: Query, QueryOne, Mutate, Lookup, Run, PoolStatus
```

**Service adapters** (`adapter/service/`):
- `hotspot.go` — AddUser, UpdateUser, RemoveUser, DisconnectUser, AddProfile, ...
- `voucher.go` — GenerateVoucher, CacheVouchers, RecordSale
- `system.go` — GetSystemResource, GetSystemLog, Reboot, Shutdown, GetDashboard
- `scheduler.go` — DeployExpireMonitor, CheckExpireMonitor
- `script.go` — GenerateOnLoginScript, ParseOnLoginPut
- `network.go` — ListInterfaces, ListDHCPLeases, ReleaseLease
- `ppp.go` — ListSecrets, AddSecret, ListActive, DisconnectActive
- `quick_print.go` — CRUD untuk package quick print
- `report.go` — FetchDailySales, FetchMonthlySales, ImportSales

---

## 4. Internal — Services (Selesai)

| Service | Baris | Fungsi Utama |
|---------|-------|-------------|
| `auth_service.go` | 365 | Login/logout JWT + refresh token, bcrypt |
| `hotspot_service.go` | 383 | CRUD user/profil, active session, host, binding |
| `router_service.go` | 529 | CRUD router, test koneksi, migrate config.php |
| `report_service.go` | 344 | Daily/monthly/summary report, export CSV/Excel |
| `voucher_service.go` | 307 | Generate voucher (10 charset), record sale, import |
| `template_service.go` | 232 | CRUD print template, render |
| `system_service.go` | 157 | Resource, log, clock, dashboard, expire monitor |

---

## 5. Internal — API Layer

### Handlers

| File | Baris | Handler |
|------|-------|---------|
| `hotspot_handler.go` | 522 | ListUsers, AddUser, ListProfiles, Active, Hosts, Bindings |
| `router_handler.go` | 152 | List, Create, Update, Delete, Test, Migrate |
| `system_handler.go` | 222 | Resource, Log, Clock, Dashboard, ExpireMonitor, Reboot |
| `report_handler.go` | 207 | Daily, Monthly, Resume, Summary, ExportCSV, ExportExcel |
| `auth_handler.go` | 202 | Setup, Login, Refresh, Logout, Me, ChangePassword |
| `template_handler.go` | 176 | CRUD + Render |
| `event_handler.go` | 180 | OnLoginEvent, HealthCheck |
| `quick_print_handler.go` | 161 | CRUD package |
| `voucher_handler.go` | 143 | Generate, Cache, PrintData, RecordSale, Import |
| `ppp_handler.go` | 135 | ListSecrets, AddSecret, ListActive, Profiles |
| `network_handler.go` | 133 | Interfaces, Traffic, Pools, Queues, DHCP |
| `sse/log_sse_handler.go` | — | StreamAll, StreamHotspot, StreamPPP |
| `sse/telemetry_sse_handler.go` | — | Stream(measurement) — SSE dari Redis Pub/Sub |

### Middleware

| Middleware | Status | Deskripsi |
|-----------|--------|-----------|
| `AuthMiddleware` | Selesai | JWT validation |
| `CORSMiddleware` | Selesai | Dynamic origin, max-age 24h |
| `LoggerMiddleware` | Selesai | Request logging |

---

## 6. Internal — Models (Database)

6 GORM models. Data operasional RouterOS (users, profiles, dll) **tidak** disimpan di DB — diambil langsung dari router via Redis cache.

| Model | Tabel | Keterangan |
|-------|-------|-----------|
| `Router` | `routers` | Konfigurasi router, soft delete, AES encrypted password |
| `SystemUser` | `system_users` | Admin user, bcrypt, soft delete |
| `VoucherSale` | `voucher_sales` | Transaksi penjualan, FK ke Router |
| `ProfilePriceMapping` | `profile_price_mappings` | Harga profil per router |
| `AuditLog` | `audit_logs` | Audit trail |
| `PrintTemplate` | `print_templates` | Template HTML voucher |

---

## 7. Startup Flow (cmd/api/main.go)

```
1. config.Load()               — baca env
2. database.Connect()          — PostgreSQL
3. redis.Connect()             — Redis client
4. rrepository.NewRedisCache() — cache.Repository
5. timeseries.NewInfluxWriter()— timeseries.Writer (optional)
6. pubsub.NewRedisPublisher()  — pubsub.Publisher
7. pubsub.NewRedisSubscriber() — pubsub.Subscriber (untuk SSE)
8. timeseries.NewInfluxReader()— timeseries.Reader (untuk history)
9. orchestrator.New(Config{Cache, TimeSeries, PubSub})
   └── wires: Pool + Processor + StreamManager + PollScheduler + Dispatcher
10. registerStartupRouters()   — load dari DB, engine.AddRouter() per router
11. engine.Start(ctx)
    ├── pool.Start() → connect all routers
    └── per router: startRouterWorkers()
          ├── command.ByType(Stream) → start semua stream spec
          └── command.ByType(Poll)   → start semua poll spec
12. worker.Start(ctx)          — background tasks
13. api.NewRouter(...)         — setup Gin routes
14. http.Server.ListenAndServe()
```

---

## 8. Background Worker

| Task | Interval | Deskripsi |
|------|----------|-----------|
| Sales cache warmup | Sekali saat startup | Warm-up Redis cache penjualan hari ini & bulan ini |
| Profile price sync | 5 menit | Sync profil hotspot dari RouterOS ke `profile_price_mappings` |
| Voucher session cleanup | 1 jam | Log jumlah sesi aktif per router |

---

## 9. On-Login System

RouterOS memanggil `POST /api/v1/events/on-login` saat user login hotspot.

**Expire modes:**

| Mode | Aksi | Record Sale |
|------|------|-------------|
| `ntf` | `limit-uptime=1s` | Tidak |
| `ntfc` | `limit-uptime=1s` | Ya |
| `rem` | Delete user | Tidak |
| `remc` | Delete user | Ya |
| `0` | Tidak ada | Tidak |

**Script metadata (`:put`):**
```
:put (",{EXPMODE},{PRICE},{VALIDITY},{SELLINGPRICE},,{LOCKUSER},{LOCKSERVER},")
```

---

## 10. Voucher Generation

- Tipe: `vc` (username=password) | `up` (username≠password)
- 10 charset mode: lower, upper, upplow, mix, mix1, mix2, num, lower1, upper1, upplow1
- Batch 50 per request dengan deduplication check
- Comment format: `{vc|up}-{gencode}-{MM.DD.YY}-{gcomment}`

---

## 11. Docker & Deployment

### Dev stack (`docker-compose.dev.yml`)
PostgreSQL 16 + Redis 7 + InfluxDB 3 Core + API

### Env vars penting untuk dev:
```
INFLUXDB_URL=http://influxdb:8181
INFLUXDB_TOKEN=dev-influx-token-change-in-prod
INFLUXDB_DATABASE=mikhmon
REDIS_HOST=redis
REDIS_PORT=6379
```

---

## 12. Status Implementasi

### Selesai
- **core/command**: Registry, Meta, Builder, Helpers
- **core/definition**: Semua command terdaftar (hotspot, system, network, ppp)
- **core/model + parser**: Domain struct + RouterOS response parsers
- **behavior**: Stream/Poll/Query/Mutation handlers + interfaces
- **execution**: Pool, RouterConn, Executor
- **orchestrator**: Engine, Dispatcher, RouterManager
- **pipeline/cache**: Redis HSET + secondary index + TTL
- **pipeline/event**: Processor (StreamSink + PollSink)
- **pipeline/pubsub**: Publisher + Subscriber (Redis Pub/Sub)
- **pipeline/timeseries**: InfluxDB Writer + Reader
- **adapter/service**: Bridge + 9 service adapters
- **services**: Auth, Hotspot, Router, Report, Voucher, Template, System
- **api/handlers**: 11 handlers + 2 SSE handlers
- **worker**: Background tasks
- **models + repositories + migrations**: Selesai

### Belum / Perlu Perhatian
- Test coverage (tujuan: 80%+)
- Auth middleware sudah selesai tapi perlu audit security
- Export Excel (handler ada, implementasi service perlu dicek)

---

## 13. Technology Stack

| Technology | Version | Fungsi |
|-----------|---------|--------|
| Go | 1.24.0 | Backend |
| Gin | 1.10.0 | HTTP framework |
| GORM | 1.25.12 | ORM |
| PostgreSQL | 16 | Primary DB |
| Redis | 7 | Cache + Pub/Sub |
| InfluxDB | 3 Core | Time-series metrics |
| go-routeros | 3.0.1 | RouterOS API client |
| go-redis | 9.7.0 | Redis client |
| Docker | — | Containers |
