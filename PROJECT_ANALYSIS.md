# Roskit API — Analisis Project

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
                  pipeline/event.Processor  ◀── InactiveAggregator (via MultiSink)
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
roskit/                               # 110 Go files, ~15.000+ baris
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
│       │   ├── definition/           # 13 file registrasi command RouterOS
│       │   ├── model/                # Domain struct (hotspot, network, system)
│       │   └── parser/               # RouterOS response parsers
│       ├── behavior/
│       │   ├── interfaces.go         # StreamSink, PollSink, QueryHandler, dll
│       │   ├── stream/               # Manager, Worker, LogWorker, InterfaceMonitor, MultiSink
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
│       │   ├── event/                # Processor + InactiveAggregator + types
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
├── tests/                            # Python integration tests
│   └── http/                         # test_auth, test_hotspot, test_routers, ...
├── testdata/                         # Testdata untuk docs dan referensi RouterOS
│   └── mikrotik/
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
    Path              string
    Type              CommandType
    SupportsFollow    bool
    SupportsInterval  bool
    SupportsCountOnly bool
    PollInterval      time.Duration
    CacheTTL          time.Duration
    IndexField        string        // secondary Redis index (e.g. "name")
    Measurement       string        // InfluxDB + Redis key segment
    Category          string        // "hotspot", "ppp", "system", dll
    WriteTimeSeries   bool          // hanya true untuk telemetri aktif
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
| `hotspot.go` | hotspot | Stream (user, profile, active, server, host, cookie, ip-binding) + Mutations + Queries |
| `system.go` | system | Poll (resource, identity, clock, health, routerboard) + Stream (scheduler, script) + Mutations |
| `network.go` | network, ip, queue | Stream (interface, address, route, arp, neighbor, pool, dhcp-lease, dns-static, nat, queue/simple) + Poll (dhcp-server, dhcp-client, dns, service, vrf) + Mutations |
| `ppp.go` | ppp | Stream (secret, active, profile) + Mutations + Queries |
| `interface.go` | interface | Poll (vlan, bridge, ethernet, wireless, wifi, wireguard, pppoe-client, l2tp-client) + Stream (bridge/host, wireless/registration-table, wifi/registration-table) + Mutations |
| `firewall.go` | firewall | Poll (filter, mangle, raw) + Stream (address-list, connection) + Mutations |
| `routing.go` | routing | Stream (route, ospf/neighbor, bgp/session) + Poll (ospf/instance, ospf/interface, bgp/connection) + Mutations |
| `queue.go` | queue | Poll (tree, interface, type) + Mutations |
| `ipsec.go` | ipsec | Poll (peer, policy) + Stream (active-peers, installed-sa) + Mutations |
| `user.go` | user | Poll (user, group) + Stream (user/active) + Mutations |
| `user_manager.go` | user-manager | Poll (profile, router) + Stream (session) + Mutations |
| `radius.go` | radius | Poll + Mutations |
| `tools.go` | tools | Stream (netwatch) + Mutations |

**Stream commands aktif (=follow, pilihan utama):**

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
| `ppp/profile/print` | `ppp_profile` | name |
| `system/scheduler/print` | `system_scheduler` | name |
| `system/script/print` | `system_script` | name |
| `interface/print` | `interface` | name |
| `interface/monitor-traffic` | `interface_traffic` | name |
| `ip/address/print` | `address` | address |
| `ip/route/print` | `route` | — |
| `ip/arp/print` | `arp` | address |
| `ip/firewall/nat/print` | `firewall_nat` | — |
| `ip/firewall/address-list/print` | `firewall_address_list` | — |
| `ip/firewall/connection/print` | `firewall_connection` | — |
| `routing/route/print` | `routing_route` | — |
| `routing/ospf/neighbor/print` | `ospf_neighbor` | — |
| `routing/bgp/session/print` | `bgp_session` | — |
| `ip/ipsec/active-peers/print` | `ipsec_active_peers` | — |
| `ip/ipsec/installed-sa/print` | `ipsec_installed_sa` | — |
| `user/active/print` | `user_active` | — |
| `user-manager/session/print` | `user_manager_session` | — |
| `tool/netwatch/print` | `tool_netwatch` | — |
| `interface/bridge/host/print` | `bridge_host` | — |
| `interface/wireless/registration-table/print` | `wireless_registration_table` | — |
| `interface/wifi/registration-table/print` | `wifi_registration_table` | — |
| `queue/simple/print` | `queue_simple` | name |
| `ip/pool/print` | `ip_pool` | name |
| `ip/pool/used/print` | `pool_used` | — |
| `ip/neighbor/print` | `neighbor` | — |
| `ip/dns/static/print` | `dns_static` | — |

**Poll commands (periodic, pilihan utama):**

| Path | Interval | Measurement |
|------|----------|-------------|
| `system/resource/print` | 60s | `system_resource` |
| `system/identity/print` | 5m | `system_identity` |
| `system/clock/print` | 60s | `system_clock` |
| `system/health/print` | 30s | `system_health` |
| `system/routerboard/print` | 10m | `system_routerboard` |
| `ip/dhcp-server/print` | 5m | — |
| `ip/dhcp-client/print` | 2m | — |
| `ip/dns/print` | 30m | — |
| `ip/service/print` | 30m | — |
| `ip/vrf/print` | 30m | — |
| `ip/firewall/filter/print` | 10m | — |
| `ip/firewall/mangle/print` | 10m | — |
| `ip/firewall/raw/print` | 10m | — |
| `interface/vlan/print` | 5m | — |
| `interface/bridge/print` | 5m | — |
| `interface/ethernet/print` | 5m | — |
| `interface/wireless/print` | 5m | — |
| `interface/wireguard/peers/print` | 2m | — |
| `routing/ospf/instance/print` | 5m | — |
| `routing/bgp/connection/print` | 5m | — |
| `queue/tree/print` | 10m | — |
| `queue/interface/print` | 1m | — |
| `radius/print` | 30m | — |
| `user/print` | 30m | — |
| `user-manager/user/print` | 2m | — |

**WriteTimeSeries = true** (hanya ditulis ke InfluxDB):
- `system/resource/print` → `system_resource`
- `interface/monitor-traffic` → `interface_traffic`

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

Engine juga menyediakan:
- `AddRouter(ctx, cfg)` — hot-add router saat runtime (connect + launch workers)
- `RemoveRouter(routerID)` — graceful stop + unregister
- `ExecuteCommand(ctx, routerID, sentence...)` — raw command one-shot
- `Stop()` — graceful shutdown seluruh engine

**Dispatcher** — unified API untuk semua operasi RouterOS:
- `Stream(ctx, routerID, path)` — start/reuse stream
- `Query(ctx, routerID, path, filters...)` — Redis-first read
- `Mutate(ctx, routerID, path, args...)` — write ke router
- `Run(ctx, routerID, sentence...)` — raw command

### 3.5 pipeline/event — Processor + InactiveAggregator

**Processor** implements `StreamSink` + `PollSink`. Menerima events dari behavior layer, menulis ke 3 storage.

**Realtime allowlist** (hanya measurement ini yang di-PUBLISH ke SSE):
```go
var realtimeMeasurements = map[string]bool{
    "hotspot_active":    true,
    "interface_traffic": true,
    "system_resource":   true,
    "dhcp_lease":        true,
}
```

**handleUpdate (stream event):**
1. `timeseries.WritePoint()` → InfluxDB (hanya jika `WriteTimeSeries == true`)
2. `cache.SetSnapshot(key, fields, TTL)` → Redis `HSET` + `EXPIRE`
3. `cache.SetIndex(indexKey, name, cacheKey)` → Redis secondary index (jika IndexField != "")
4. `pubsub.Publish()` → hanya untuk realtime measurements

**handleDead (=.dead event dari RouterOS):**
1. `cache.DeleteIndex()` → hapus secondary index entry
2. `cache.DeleteSnapshot()` → hapus snapshot
3. `pubsub.Publish()` → broadcast dead event (jika realtime)

**OnPoll (poll event):**
1. Per-row: `cache.SetSnapshot()`
2. `pubsub.Publish()` → broadcast poll update (jika realtime measurement)

**InactiveAggregator** (`pipeline/event/aggregator.go`):  
Implements `StreamSink`. Menerima events `ppp_secret`, `ppp_active`, `hotspot_user`, `hotspot_active` dan menghitung daftar user/secret yang tidak sedang aktif secara real-time. Hasilnya di-cache sebagai snapshot Redis + dipublish ke pubsub.

**MultiSink** (`behavior/stream/multi_sink.go`):  
Fan-out `StreamSink` — memungkinkan satu stream dikirim ke beberapa sink sekaligus (misal: Processor + InactiveAggregator).

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

- **Writer**: `InfluxWriter` — HTTP line protocol, buffer 500 points, flush setiap 5s. Optional: jika `INFLUXDB_URL` tidak dikonfigurasi, `NoopWriter` digunakan.
- **Reader**: `InfluxReader` — query historical data (untuk `/system/resource/history` dll). Optional: `NoopReader` jika tidak dikonfigurasi.
- Hanya measurement dengan `WriteTimeSeries == true` yang ditulis ke InfluxDB.

### 3.8 pipeline/pubsub — Redis Pub/Sub

- **Publisher** (`redis_pub.go`) — `PUBLISH` ke channel `roskit:telemetry:{routerID}`
- **Subscriber** (`redis_sub.go`) — `SUBSCRIBE`, forward ke SSE handlers
- **Subscriber interface** (`subscriber.go`) — `Subscribe(ctx, channels...) (<-chan SubscriberMessage, error)` + `NoopSubscriber`
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
| `router_service.go` | 531 | CRUD router, test koneksi, migrate config.php, SeedEngineFromDB |
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
| `system_handler.go` | 222 | Resource, Log, Clock, Dashboard, ExpireMonitor, Reboot |
| `report_handler.go` | 207 | Daily, Monthly, Resume, Summary, ExportCSV, ExportExcel |
| `auth_handler.go` | 202 | Setup, Login, Refresh, Logout, Me, ChangePassword |
| `event_handler.go` | 180 | OnLoginEvent, HealthCheck |
| `template_handler.go` | 176 | CRUD + Render |
| `router_handler.go` | 152 | List, Create, Update, Delete, Test, Migrate |
| `quick_print_handler.go` | 161 | CRUD package |
| `voucher_handler.go` | 143 | Generate, Cache, PrintData, RecordSale, Import |
| `ppp_handler.go` | 135 | ListSecrets, AddSecret, ListActive, Profiles |
| `network_handler.go` | 133 | Interfaces, Traffic, Pools, Queues, DHCP |
| `sse/log_sse_handler.go` | 146 | StreamAll, StreamHotspot, StreamPPP |
| `sse/telemetry_sse_handler.go` | 84 | Stream(measurement) — SSE dari Redis Pub/Sub |

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
3. redis.Connect()             — Redis client (pkg/redis)
4. appredis.NewCache()         — app-level Redis cache wrapper
5. roskitcache.NewRedisRepository() — roskit cache.Repository
6. roskitpubsub.NewRedisPublisher() — pubsub.Publisher
7. roskitpubsub.NewRedisSubscriber()— pubsub.Subscriber (untuk SSE)
8. roskittimeseries.NewInfluxWriter()— timeseries.Writer (optional)
9. roskittimeseries.NewInfluxReader()— timeseries.Reader (optional)
10. orchestrator.New(Config{Cache, TimeSeries, PubSub})
    └── wires: Pool + Processor + StreamManager + PollScheduler + Dispatcher
11. roskitservice.NewBridge(dispatcher, cache)
12. routerSvc.SeedEngineFromDB()   — load router dari DB → engine.AddRouter() per router
13. engine.Start(ctx)
    ├── pool.Start() → connect all routers
    └── per router: startRouterWorkers()
          ├── command.ByType(Stream) → start semua stream spec
          └── command.ByType(Poll)   → start semua poll spec
14. backgroundWorker.Start(ctx)  — background tasks
15. api.NewRouter(...)           — setup Gin routes
16. http.Server.ListenAndServe()
```

Semua dependency (Redis, InfluxDB) bersifat optional dengan Noop fallback — server dapat berjalan tanpa mereka dalam mode degraded.

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

### Stack (`docker-compose.yml`)
PostgreSQL 16 + Redis 7 + API

### Stack dengan Metrics (`docker-compose.yml` profile `metrics`)
PostgreSQL 16 + Redis 7 + InfluxDB 3 Core + API

### Dev stack (`docker-compose.dev.yml`)
Sama dengan stack utama, berbeda konfigurasi env.

### Env vars penting:
```
# Database
POSTGRES_DSN=...

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# InfluxDB (optional — noop jika tidak di-set)
INFLUXDB_URL=http://influxdb:8181
INFLUXDB_TOKEN=dev-influx-token-change-in-prod
INFLUXDB_DATABASE=mikhmon

# App
AES_ENC_KEY=...
JWT_SECRET=...
PORT=8080
```

---

## 12. Testing

- `pkg/encrypt/encrypt_test.go` — unit test encrypt package
- `tests/http/` — Python integration tests (pytest):
  - `test_auth.py`, `test_hotspot.py`, `test_routers.py`
  - `test_events.py`, `test_mikrotik_realtime.py`
- Coverage keseluruhan: **rendah** (target 80% belum tercapai)

---

## 13. Status Implementasi

### Selesai
- **core/command**: Registry, Meta, Builder, Helpers (termasuk `WriteTimeSeries` flag)
- **core/definition**: 13 file — hotspot, system, network, ppp, interface, firewall, routing, queue, ipsec, user, user_manager, radius, tools
- **core/model + parser**: Domain struct + RouterOS response parsers
- **behavior**: Stream/Poll/Query/Mutation handlers + interfaces + MultiSink
- **execution**: Pool, RouterConn, Executor
- **orchestrator**: Engine (hot-add/remove router, graceful stop), Dispatcher, RouterManager
- **pipeline/cache**: Redis HSET + secondary index + TTL
- **pipeline/event**: Processor (realtime allowlist) + InactiveAggregator + types
- **pipeline/pubsub**: Publisher + Subscriber interface + Redis implementation
- **pipeline/timeseries**: InfluxDB Writer + Reader (optional, noop fallback)
- **adapter/service**: Bridge + 9 service adapters
- **services**: Auth, Hotspot, Router, Report, Voucher, Template, System
- **api/handlers**: 11 handlers + 2 SSE handlers
- **worker**: Background tasks
- **models + repositories + migrations**: Selesai

### Belum / Perlu Perhatian
- Test coverage (saat ini rendah, tujuan: 80%+)
- `MultiSink` dan `InactiveAggregator` sudah ada tapi **belum diwire** di `engine.go` / `main.go`
- Auth middleware sudah selesai tapi perlu audit security
- Export Excel (handler ada, implementasi service perlu dicek)

---

## 14. Technology Stack

| Technology | Version | Fungsi |
|-----------|---------|--------|
| Go | 1.24.0 | Backend |
| Gin | 1.10.0 | HTTP framework |
| GORM | 1.25.12 | ORM |
| PostgreSQL | 16 | Primary DB |
| Redis | 7 | Cache + Pub/Sub |
| InfluxDB | 3 Core | Time-series metrics (optional) |
| go-routeros | 3.0.1 | RouterOS API client |
| go-redis | 9.7.0 | Redis client |
| excelize | 2.10.1 | Excel export |
| go-deepcopy | 1.7.2 | Deep copy utility |
| Docker | — | Containers |

---

## 15. Daftar Lengkap RouterOS Commands

Semua command terdaftar via `init()` di `core/definition/`. Total ~230+ path unik.

---

### 15.1 STREAM — RouterOS Push Real-time (`=follow`)

> Dijalankan sekali per router saat startup. RouterOS mengirim data secara push tiap ada perubahan.  
> `★` = `WriteTimeSeries = true` (ditulis ke InfluxDB)

#### Hotspot
| Path | Measurement | Index |
|------|-------------|-------|
| `ip/hotspot/user/print` | `hotspot_user` | name |
| `ip/hotspot/user/profile/print` | `hotspot_profile` | name |
| `ip/hotspot/active/print` ★ | `hotspot_active` | — |
| `ip/hotspot/print` | `hotspot_server` | name |
| `ip/hotspot/host/print` | `hotspot_host` | — |
| `ip/hotspot/cookie/print` | `hotspot_cookie` | — |
| `ip/hotspot/ip-binding/print` | `ip_binding` | — |
| `ip/hotspot/walled-garden/print` | `walled_garden` | — |
| `ip/hotspot/walled-garden/ip/print` | `walled_garden_ip` | — |

#### PPP
| Path | Measurement | Index |
|------|-------------|-------|
| `ppp/secret/print` | `ppp_secret` | name |
| `ppp/active/print` | `ppp_active` | name |
| `ppp/profile/print` | `ppp_profile` | name |

#### System
| Path | Measurement | Index |
|------|-------------|-------|
| `system/scheduler/print` | `system_scheduler` | name |
| `system/script/print` | `system_script` | name |

#### Network & IP
| Path | Measurement | Index |
|------|-------------|-------|
| `interface/print` | `interface` | name |
| `interface/monitor-traffic` ★ | `interface_traffic` | name |
| `ip/address/print` | `address` | address |
| `ip/route/print` | `route` | — |
| `ip/arp/print` | `arp` | address |
| `ip/neighbor/print` | `neighbor` | — |
| `ip/pool/print` | `ip_pool` | name |
| `ip/pool/used/print` | `pool_used` | — |
| `ip/dhcp-server/lease/print` | `dhcp_lease` | address |
| `ip/dns/static/print` | `dns_static` | — |
| `ip/firewall/nat/print` | `firewall_nat` | — |
| `queue/simple/print` | `queue_simple` | name |

#### Interface (detail types)
| Path | Measurement | Index |
|------|-------------|-------|
| `interface/bridge/host/print` | `bridge_host` | — |
| `interface/wireless/registration-table/print` | `wireless_registration_table` | — |
| `interface/wifi/registration-table/print` | `wifi_registration_table` | — |

#### Firewall
| Path | Measurement | Index |
|------|-------------|-------|
| `ip/firewall/address-list/print` | `firewall_address_list` | — |
| `ip/firewall/connection/print` | `firewall_connection` | — |

#### Routing
| Path | Measurement | Index |
|------|-------------|-------|
| `routing/route/print` | `routing_route` | — |
| `routing/ospf/neighbor/print` | `ospf_neighbor` | — |
| `routing/bgp/session/print` | `bgp_session` | — |

#### IPSec
| Path | Measurement | Index |
|------|-------------|-------|
| `ip/ipsec/active-peers/print` | `ipsec_active_peers` | — |
| `ip/ipsec/installed-sa/print` | `ipsec_installed_sa` | — |

#### User & Management
| Path | Measurement | Index |
|------|-------------|-------|
| `user/active/print` | `user_active` | — |
| `user-manager/session/print` | `user_manager_session` | — |
| `tool/netwatch/print` | `tool_netwatch` | — |

---

### 15.2 POLL — Periodic Snapshot (ticker-based)

> Dijalankan periodic sesuai interval. RouterOS mengembalikan snapshot satu kali per panggilan.

#### System
| Path | Interval | Measurement |
|------|----------|-------------|
| `system/resource/print` ★ | 60s | `system_resource` |
| `system/resource/cpu/print` | 30s | `system_resource_cpu` |
| `system/health/print` | 30s | `system_health` |
| `system/clock/print` | 60s | `system_clock` |
| `system/identity/print` | 5m | `system_identity` |
| `system/routerboard/print` | 10m | `system_routerboard` |
| `system/logging/print` | 30m | `system_logging` |
| `system/logging/action/print` | 30m | `system_logging_action` |
| `system/package/print` | 60m | `system_package` |
| `system/ntp/client/print` | 60m | `ntp_client` |
| `system/ntp/client/servers/print` | 60m | `ntp_servers` |

#### Hotspot
| Path | Interval | Measurement |
|------|----------|-------------|
| `ip/hotspot/profile/print` | 5m | `hotspot_server_profile` |
| `ip/hotspot/service-port/print` | 30m | `hotspot_service_port` |

#### Network & IP
| Path | Interval | Measurement |
|------|----------|-------------|
| `ip/dhcp-server/print` | 5m | `dhcp_server` |
| `ip/dhcp-server/network/print` | 10m | `dhcp_server_network` |
| `ip/dhcp-server/option/print` | 30m | `dhcp_server_option` |
| `ip/dhcp-client/print` | 2m | `dhcp_client` |
| `ip/dns/print` | 30m | `dns` |
| `ip/service/print` | 30m | `ip_service` |
| `ip/vrf/print` | 30m | `ip_vrf` |

#### Interface (detail types)
| Path | Interval | Measurement |
|------|----------|-------------|
| `interface/vlan/print` | 5m | `interface_vlan` |
| `interface/bridge/print` | 5m | `interface_bridge` |
| `interface/bridge/port/print` | 5m | `interface_bridge_port` |
| `interface/ethernet/print` | 5m | `interface_ethernet` |
| `interface/wireless/print` | 5m | `interface_wireless` |
| `interface/wifi/print` | 5m | `interface_wifi` |
| `interface/wireguard/print` | 5m | `interface_wireguard` |
| `interface/wireguard/peers/print` | 2m | `interface_wireguard_peers` |
| `interface/pppoe-client/print` | 2m | `interface_pppoe_client` |
| `interface/l2tp-client/print` | 2m | `interface_l2tp_client` |

#### Firewall
| Path | Interval | Measurement |
|------|----------|-------------|
| `ip/firewall/filter/print` | 10m | `firewall_filter` |
| `ip/firewall/mangle/print` | 10m | `firewall_mangle` |
| `ip/firewall/raw/print` | 10m | `firewall_raw` |

#### Routing
| Path | Interval | Measurement |
|------|----------|-------------|
| `routing/ospf/instance/print` | 5m | `ospf_instance` |
| `routing/ospf/interface/print` | 5m | `ospf_interface` |
| `routing/bgp/connection/print` | 5m | `bgp_connection` |

#### Queue
| Path | Interval | Measurement |
|------|----------|-------------|
| `queue/tree/print` | 10m | `queue_tree` |
| `queue/interface/print` | 1m | `queue_interface` |
| `queue/type/print` | 30m | `queue_type` |

#### IPSec
| Path | Interval | Measurement |
|------|----------|-------------|
| `ip/ipsec/peer/print` | 5m | `ipsec_peer` |
| `ip/ipsec/policy/print` | 5m | `ipsec_policy` |

#### User & Management
| Path | Interval | Measurement |
|------|----------|-------------|
| `user/print` | 30m | `user` |
| `user/group/print` | 30m | `user_group` |
| `user-manager/user/print` | 2m | `user_manager_user` |
| `user-manager/profile/print` | 10m | `user_manager_profile` |
| `user-manager/router/print` | 10m | `user_manager_router` |
| `radius/print` | 30m | `radius` |

---

### 15.3 QUERY — On-demand Read

> Dipanggil saat dibutuhkan. Cache-first via Redis, fallback ke RouterOS.

#### Hotspot
```
ip/hotspot/user/find          ip/hotspot/user/get
ip/hotspot/user/profile/find  ip/hotspot/user/profile/get
ip/hotspot/active/find        ip/hotspot/active/get
```

#### System
```
system/log/print       system/logging/find    system/package/find
```

#### Network & IP
```
interface/find               interface/get
ip/address/find              ip/address/get
ip/route/find
ip/arp/find
ip/neighbor/find
ip/pool/find                 ip/pool/used/find
ip/dhcp-server/lease/find    ip/dhcp-server/find
ip/dns/static/find
ip/service/find
ip/vrf/find
queue/simple/find            queue/simple/get
```

#### Interface (detail types)
```
interface/vlan/find          interface/bridge/find        interface/bridge/port/find
interface/ethernet/find      interface/wireless/find      interface/wifi/find
interface/wireguard/find     interface/wireguard/peers/find
interface/pppoe-client/find  interface/l2tp-client/find
```

#### Firewall
```
ip/firewall/filter/find    ip/firewall/address-list/find
ip/firewall/mangle/find    ip/firewall/connection/find
```

#### Routing
```
routing/route/find    routing/ospf/instance/find    routing/bgp/connection/find
```

#### Queue
```
queue/tree/find    queue/interface/find    queue/type/find
```

#### IPSec
```
ip/ipsec/peer/find    ip/ipsec/policy/find    ip/ipsec/active-peers/find
```

#### PPP
```
ppp/secret/find    ppp/active/find    ppp/profile/find
```

#### User & Management
```
user/find              user/active/find
user-manager/user/find user-manager/session/find
radius/find            tool/netwatch/find
```

---

### 15.4 MUTATION — Write ke RouterOS

> Tidak di-cache. Selalu langsung ke RouterOS.

#### Hotspot — user
```
ip/hotspot/user/add        ip/hotspot/user/set
ip/hotspot/user/remove     ip/hotspot/user/enable
ip/hotspot/user/disable    ip/hotspot/user/reset-counters
```

#### Hotspot — user profile
```
ip/hotspot/user/profile/add    ip/hotspot/user/profile/set
ip/hotspot/user/profile/remove ip/hotspot/user/profile/reset
```

#### Hotspot — active session
```
ip/hotspot/active/remove    ip/hotspot/active/login
```

#### Hotspot — cookie & ip-binding
```
ip/hotspot/cookie/remove
ip/hotspot/ip-binding/add    ip/hotspot/ip-binding/set
ip/hotspot/ip-binding/remove ip/hotspot/ip-binding/enable
ip/hotspot/ip-binding/disable
```

#### Hotspot — server
```
ip/hotspot/add     ip/hotspot/set      ip/hotspot/remove
ip/hotspot/enable  ip/hotspot/disable  ip/hotspot/reset-html
```

#### Hotspot — server profile & walled-garden
```
ip/hotspot/profile/add              ip/hotspot/profile/set
ip/hotspot/profile/remove
ip/hotspot/walled-garden/ip/add     ip/hotspot/walled-garden/ip/remove
```

#### PPP
```
ppp/secret/add     ppp/secret/set     ppp/secret/remove
ppp/secret/enable  ppp/secret/disable
ppp/active/remove
ppp/profile/add    ppp/profile/set    ppp/profile/remove
ppp/profile/enable ppp/profile/disable
```

#### System — scheduler & script
```
system/scheduler/add    system/scheduler/set    system/scheduler/remove
system/script/add       system/script/set       system/script/remove
system/script/run
system/reboot           system/shutdown
```

#### System — logging & NTP
```
system/logging/add      system/logging/set      system/logging/remove
system/logging/enable   system/logging/disable
system/ntp/client/set
system/ntp/client/servers/add   system/ntp/client/servers/set
system/ntp/client/servers/remove
```

#### Interface — generic
```
interface/set    interface/enable    interface/disable
```

#### Interface — VLAN
```
interface/vlan/add     interface/vlan/set     interface/vlan/remove
interface/vlan/enable  interface/vlan/disable
```

#### Interface — Bridge
```
interface/bridge/add          interface/bridge/set
interface/bridge/remove       interface/bridge/enable
interface/bridge/disable
interface/bridge/port/add     interface/bridge/port/set
interface/bridge/port/remove  interface/bridge/port/enable
interface/bridge/port/disable
```

#### Interface — Ethernet
```
interface/ethernet/set    interface/ethernet/enable
interface/ethernet/disable interface/ethernet/reset-counters
```

#### Interface — Wireless / WiFi
```
interface/wireless/set    interface/wireless/enable    interface/wireless/disable
interface/wifi/set        interface/wifi/enable        interface/wifi/disable
```

#### Interface — WireGuard
```
interface/wireguard/add         interface/wireguard/set
interface/wireguard/remove      interface/wireguard/enable
interface/wireguard/disable
interface/wireguard/peers/add   interface/wireguard/peers/set
interface/wireguard/peers/remove interface/wireguard/peers/enable
interface/wireguard/peers/disable
```

#### Interface — PPPoE & L2TP client
```
interface/pppoe-client/add     interface/pppoe-client/set
interface/pppoe-client/remove  interface/pppoe-client/enable
interface/pppoe-client/disable
interface/l2tp-client/add      interface/l2tp-client/set
interface/l2tp-client/remove   interface/l2tp-client/enable
interface/l2tp-client/disable
```

#### IP — Address, Route, ARP
```
ip/address/add     ip/address/set     ip/address/remove
ip/address/enable  ip/address/disable
ip/route/add       ip/route/set       ip/route/remove
ip/route/enable    ip/route/disable
ip/arp/remove
```

#### IP — DHCP Server
```
ip/dhcp-server/add            ip/dhcp-server/set
ip/dhcp-server/remove         ip/dhcp-server/enable
ip/dhcp-server/disable
ip/dhcp-server/lease/add      ip/dhcp-server/lease/remove
ip/dhcp-server/network/add    ip/dhcp-server/network/set
ip/dhcp-server/network/remove
ip/dhcp-server/option/add     ip/dhcp-server/option/set
ip/dhcp-server/option/remove
```

#### IP — DHCP Client
```
ip/dhcp-client/add     ip/dhcp-client/set    ip/dhcp-client/remove
ip/dhcp-client/enable  ip/dhcp-client/disable
ip/dhcp-client/renew   ip/dhcp-client/release
```

#### IP — DNS
```
ip/dns/set
ip/dns/static/add     ip/dns/static/set     ip/dns/static/remove
ip/dns/static/enable  ip/dns/static/disable
ip/dns/cache/flush
```

#### IP — Service & VRF
```
ip/service/set    ip/service/enable    ip/service/disable
ip/vrf/add        ip/vrf/set           ip/vrf/remove
ip/vrf/enable     ip/vrf/disable
```

#### Firewall — Filter
```
ip/firewall/filter/add      ip/firewall/filter/set
ip/firewall/filter/remove   ip/firewall/filter/enable
ip/firewall/filter/disable  ip/firewall/filter/move
ip/firewall/filter/reset-counters
```

#### Firewall — Address List
```
ip/firewall/address-list/add     ip/firewall/address-list/set
ip/firewall/address-list/remove  ip/firewall/address-list/enable
ip/firewall/address-list/disable
```

#### Firewall — Mangle
```
ip/firewall/mangle/add     ip/firewall/mangle/set
ip/firewall/mangle/remove  ip/firewall/mangle/enable
ip/firewall/mangle/disable ip/firewall/mangle/move
```

#### Firewall — Raw
```
ip/firewall/raw/add     ip/firewall/raw/set
ip/firewall/raw/remove  ip/firewall/raw/enable
ip/firewall/raw/disable ip/firewall/raw/move
```

#### Firewall — NAT & Connection
```
ip/firewall/nat/add     ip/firewall/nat/set    ip/firewall/nat/remove
ip/firewall/nat/enable  ip/firewall/nat/disable ip/firewall/nat/move
ip/firewall/connection/remove
```

#### Routing — OSPF & BGP
```
routing/ospf/instance/add    routing/ospf/instance/set
routing/ospf/instance/remove routing/ospf/instance/enable
routing/ospf/instance/disable
routing/ospf/area/add        routing/ospf/area/set
routing/ospf/area/remove
routing/bgp/connection/add   routing/bgp/connection/set
routing/bgp/connection/remove routing/bgp/connection/enable
routing/bgp/connection/disable
```

#### Queue
```
queue/tree/add     queue/tree/set     queue/tree/remove
queue/tree/enable  queue/tree/disable queue/tree/reset-counters
queue/type/add     queue/type/set     queue/type/remove
queue/simple/add   queue/simple/set   queue/simple/remove
```

#### IPSec
```
ip/ipsec/peer/add     ip/ipsec/peer/set    ip/ipsec/peer/remove
ip/ipsec/peer/enable  ip/ipsec/peer/disable
ip/ipsec/policy/add   ip/ipsec/policy/set  ip/ipsec/policy/remove
ip/ipsec/policy/enable ip/ipsec/policy/disable
```

#### User (RouterOS local user)
```
user/add    user/set    user/remove    user/enable    user/disable
```

#### User Manager
```
user-manager/user/add      user-manager/user/set      user-manager/user/remove
user-manager/profile/add   user-manager/profile/set   user-manager/profile/remove
```

#### RADIUS & Tools
```
radius/add    radius/set    radius/remove    radius/enable    radius/disable
tool/netwatch/add    tool/netwatch/set    tool/netwatch/remove
tool/netwatch/enable tool/netwatch/disable
```
