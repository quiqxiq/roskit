# Mikhmon Integration Plan — Roskit (OpenCode)

> Generated for review. No code changes until approved.

---

## 1. Project Architecture Summary

### Tech Stack
Go 1.24 · Gin (HTTP) · GORM (PostgreSQL) · Redis 7 · InfluxDB 3 Core

### Layering

```
HTTP (Gin handler)
  → Service (orchestration via Bridge + repository + Cache B)
    → Bridge (roskit adapter — single entry point to RouterOS)
      → Dispatcher → behavior handlers (stream/poll/query/mutation)
        → execution.Pool → RouterOS TCP (go-routeros)
    → Repository (pure GORM → PostgreSQL)
```

### Roskit Subsystem (`internal/roskit/`)

| Package | Responsibility |
|---------|---------------|
| `core/command` | Global command registry. Commands auto-registered via `init()` |
| `core/definition` | Registers all RouterOS paths (stream/poll/query/mutation) |
| `core/parser` | Parses RouterOS reply maps into structured types |
| `core/model` | Domain models for parsed telemetry data |
| `behavior/stream` | Persistent listen/follow workers with reconnection |
| `behavior/poll` | Periodic polling workers (30s global cycle) |
| `behavior/query` | Redis-first reads, fallback to live RouterOS |
| `behavior/mutation` | Write operations + cache invalidation |
| `execution` | RouterOS TCP I/O only (pool + sentence helpers) |
| `pipeline/cache` | Redis telemetry cache (`roskit:{routerID}:{measurement}:{id}`) |
| `pipeline/event` | Processor (update→cache+timeseries+pubsub) |
| `pipeline/pubsub` | Redis pub/sub for SSE telemetry |
| `pipeline/timeseries` | InfluxDB writer/reader |
| `orchestrator` | Engine lifecycle, dispatcher, router manager |
| `adapter/service` | **Bridge** — the single service-layer entry point |

### Two Cache Systems

| | Cache A (Telemetry) | Cache B (App) |
|---|---|---|
| Package | `pipeline/cache` | `pkg/redis` |
| Key | `roskit:{routerID}:{measurement}:{id}` | `mikhmon:{resource}:{id}` |
| Managed by | Streaming pipeline (auto, 5min TTL) | Service layer (manual) |
| Rule | NEVER manually invalidate | Invalidate after writes |

### Key Conventions
- Services inject `*service.Bridge` — never `*execution.Pool` or raw `*routeros.Client`
- Router IDs are `string` in Bridge/Pool, `uint` in API URL params
- `core/` is pure — no Redis/Influx/network imports
- `execution/` is the only layer importing `go-routeros`
- `pipeline/` is the only layer importing Redis/InfluxDB clients
- DB stores ONLY: routers, users, sales, templates, audit logs, profile price mappings
- Response envelope: `{"data": ..., "error": null}` or `{"data": null, "error": "msg"}`
- All errors via `AppError` from `pkg/errors`

---

## 2. Mikhmon Feature Inventory

### Legend
- ✅ Done — fully implemented with handler, service, and Bridge layers
- ⚠️ Partial — Bridge/service layer exists but API handler missing, or implementation incomplete
- ❌ Missing — not implemented at all

| # | Fitur | Deskripsi | Status | Prioritas |
|---|-------|-----------|--------|-----------|
| **Router Management** |
| 1 | Router CRUD | Create/read/update/delete routers | ✅ Done | — |
| 2 | Connection test | Standalone `routeros.DialContext()`, not through engine | ✅ Done | — |
| 3 | Config migration | Import from legacy PHP config.php | ✅ Done | — |
| 4 | Logo upload | Upload PNG logo per router session | ❌ Missing | Medium |
| **Hotspot Users** |
| 5 | User CRUD | Add/set/remove/enable/disable hotspot users | ✅ Done | — |
| 6 | User count | Count minus 1 (exclude admin) | ✅ Done | — |
| 7 | Reset counters | Reset user byte counters | ⚠️ Partial | Medium |
| 8 | Export users | Export as RouterOS script or CSV | ✅ Done | — |
| 9 | User comment parsing | Parse vc/up codes and expiry dates | ✅ Done | — |
| 10 | Remove user with cleanup | Remove user + associated script + scheduler | ✅ Done | — |
| **Hotspot Profiles** |
| 11 | Profile CRUD | Add/set/remove profiles with full field support | ✅ Done | — |
| 12 | On-login script generation | Generate on-login for ntf/ntfc/rem/remc/0 modes | ✅ Done | — |
| 13 | Profile metadata parsing | Parse `:put(...)` into structured metadata | ✅ Done | — |
| 14 | Profile price mapping | Worker syncs profile pricing from RouterOS to DB | ✅ Done | — |
| 15 | Profile fields | address-pool, rate-limit, shared-users, parent-queue | ✅ Done | — |
| **Active Sessions** |
| 16 | List/remove active | List and remove active hotspot sessions | ✅ Done | — |
| 17 | Disconnect user | Remove cookie + active session | ✅ Done | — |
| **Hosts / Cookies / Bindings** |
| 18 | Hosts list/remove | ✅ Done | — |
| 19 | Cookies list/remove | ✅ Done | — |
| 20 | IP Bindings full CRUD | Add/set/remove/enable/disable + cleanup (queue+scheduler+ARP+DHCP) | ✅ Done | — |
| **Voucher System** |
| 21 | Voucher generation | vc/up modes, all charsets, batch create (50/group) | ✅ Done | — |
| 22 | Voucher caching | Redis session cache (2h TTL) | ✅ Done | — |
| 23 | Quick print packages | CRUD stored as RouterOS scripts | ✅ Done | — |
| 24 | Voucher comment format | `vc/up-{code}-{date}-{text}` | ✅ Done | — |
| 25 | Data limit / time limit | `limit-bytes-total` and `limit-uptime` on vouchers | ✅ Done | — |
| **Print Templates** |
| 26 | Template CRUD | Create/read/update/delete print templates | ✅ Done | — |
| 27 | Template rendering | Go `html/template` rendering with variables | ⚠️ Partial | Medium |
| 28 | Default templates | Seed default/small/thermal templates | ❌ Missing | High |
| 29 | QR code generation | QR for voucher `username:password` URL | ⚠️ Partial | Low |
| 30 | Full print page | Complete HTML document wrapper with print trigger | ❌ Missing | Medium |
| **Sales Recording** |
| 31 | On-login event recording | Receive RouterOS fetch webhook → record sale | ⚠️ Partial | **Critical** |
| 32 | Manual sale recording | POST endpoint for manual sale recording | ✅ Done | — |
| 33 | Sales import | Import from RouterOS system/script entries | ✅ Done | — |
| **Reports** |
| 34 | Daily report | Sales by date with filters (profile/server/search) | ✅ Done | — |
| 35 | Monthly report | Daily breakdown for a month | ✅ Done | — |
| 36 | Resume report | Yearly 12-month breakdown | ✅ Done | — |
| 37 | Dashboard summary | Today + month count/sum | ✅ Done | — |
| 38 | CSV/Excel export | File download with all sale fields | ✅ Done | — |
| **System** |
| 39 | System resource/identity/clock/health/routerboard | ✅ Done | — |
| 40 | System resource history | InfluxDB time-range query with auto-step | ✅ Done | — |
| 41 | System log with SSE | Log streaming via Redis pubsub | ✅ Done | — |
| 42 | Reboot / shutdown | ✅ Done | — |
| 43 | Expire monitor | Deploy/remove/check scheduler | ✅ Done | — |
| 44 | Scheduler CRUD | Generic add/update/remove/enable/disable | ❌ Missing | Medium |
| 45 | Script management | List/run/remove scripts | ❌ Missing | Low |
| 46 | Logging setup | Configure RouterOS logging prefix `->` for hotspot | ⚠️ Partial | Low |
| 47 | Dashboard | Resource + identity + user counts | ⚠️ Partial | Medium |
| **Network** |
| 48 | Interfaces / traffic / pools / queues / NAT | ✅ Done | — |
| 49 | DHCP leases + release | ✅ Done | — |
| 50 | ARP entries | Streamed, queryable | ✅ Done | — |
| **PPP** |
| 51 | PPP secret/active/profile CRUD | Full CRUD with inactive computation | ✅ Done | — |
| **Walled Garden** |
| 52 | List walled garden | Bridge method exists | ✅ Done | — |
| 53 | Walled garden CRUD | Add/remove/update entries | ❌ Missing | Medium |
| **SSE / Realtime** |
| 54 | Telemetry SSE | All major measurements streamed via Redis pubsub | ✅ Done | — |
| 55 | Log SSE | Hotspot/PPP/all log streams | ✅ Done | — |
| **Auth & Security** |
| 56 | JWT auth (login/refresh/logout) | ✅ Done | — |
| 57 | Multi-role users | owner/admin/operator with bcrypt | ✅ Done | — |
| 58 | Password encryption | AES-GCM for router passwords | ✅ Done | — |
| 59 | Audit logging | Action/entity/details tracking | ✅ Done | — |
| **Frontend** |
| 60 | Basic admin SPA | Login + router CRUD + hotspot management | ⚠️ Partial | Low |
| 61 | Voucher printing UI | Print preview with template selection | ❌ Missing | Low |
| 62 | Theme system | Dark/light/blue/green/pink | ❌ Missing | Low |
| 63 | Multi-language | en/id/es/tl i18n | ❌ Missing | Low |
| 64 | User status/island page | Public page for users to check connection status | ❌ Missing | Low |

---

## 3. Gap Analysis

### 3.1 CRITICAL — Profile Price Lookup in On-Login Event

**What's wrong:** `EventHandler.fetchProfilePrice()` at `internal/api/handlers/event_handler.go:178` is a **stub** that always returns `Price: 0`. Every sale recorded via RouterOS on-login webhook has `price = 0`, making reports financially useless.

**What exists:**
- `ProfilePriceMapping` model in DB with `Price`, `SellingPrice`, `Validity`, `ExpMode`, etc.
- `ProfilePriceMappingRepository` with `Upsert()` method
- Worker syncs profile prices from RouterOS every 5 minutes (`internal/worker/worker.go:79`)
- `EventHandler` has access to `routerRepo` and `saleRepo` but NOT `profilePriceMappingRepo`

**What's needed:**
- Inject `ProfilePriceMappingRepository` into `EventHandler`
- Replace stub with actual lookup: find mapping by `(routerID, profileName)`
- Include `SellingPrice`, `Validity` from the mapping in the sale record

**Dependencies:** None — this is a self-contained fix.

### 3.2 HIGH — Default Print Templates Not Seeded

**What exists:**
- `PrintTemplate` model with `name`, `type`, `part` (header/row/footer), `content`, `router_id`
- Template CRUD endpoints
- Template rendering via Go `html/template`
- Reference templates in `mikhmon/voucher/` (default, small, thermal)

**What's needed:**
- Port default template content from mikhmon PHP (`header.default.txt`, `row.default.txt`, `footer.default.txt` etc.)
- Create seed function that inserts templates for each router on first connection or via migration
- Add `POST /api/v1/routers/:id/templates/seed-defaults` endpoint

**Dependencies:** None.

### 3.3 HIGH — Walled Garden CRUD API Handlers

**What exists:**
- Stream command registered: `ip/hotspot/walled-garden/print` → `walled_garden` measurement
- IP walled garden commands registered: `ip/hotspot/walled-garden/ip/print`, `ip/hotspot/walled-garden/ip/add`, `ip/hotspot/walled-garden/ip/remove`
- Core model: `HotspotWalledGarden` with full field set
- Parser for walled garden entries
- Bridge method: `ListWalledGarden()`

**What's needed:**
- Bridge methods: `AddWalledGarden`, `RemoveWalledGarden`, `SetWalledGarden`
- Bridge methods: `AddWalledGardenIP`, `RemoveWalledGardenIP`, `ListWalledGardenIP`
- API handlers: CRUD for both walled-garden and walled-garden/ip
- Service layer methods (thin orchestration)

**Dependencies:** Command definitions already registered — just need Bridge + service + handler.

### 3.4 MEDIUM — Reset User Counters API Endpoint

**What exists:**
- `Bridge.ResetUserCounters()` at `internal/roskit/adapter/service/hotspot.go:64`
- Mutation command registered: `ip/hotspot/user/reset-counters`
- Called internally by `RemoveHotspotUserWithCleanup`

**What's needed:**
- Add `POST /api/v1/routers/:id/hotspot/users/:id/reset-counters` handler
- Add `ResetUserCounters(routerID, userID)` method to `HotspotService`
- Wire in router.go

**Dependencies:** None — Bridge method already works.

### 3.5 MEDIUM — Scheduler CRUD API Endpoints

**What exists:**
- `Bridge.ListSchedulers()` — lists all schedulers
- `Bridge.EnableScheduler()` / `Bridge.DisableScheduler()` — used internally by expire monitor
- Mutation commands registered: `system/scheduler/add`, `system/scheduler/set`, `system/scheduler/remove`, `system/scheduler/enable`, `system/scheduler/disable`
- Stream command: `system/scheduler/print` → `system_scheduler` measurement

**What's needed:**
- Bridge methods: `AddScheduler`, `SetScheduler`, `RemoveScheduler`
- Service layer: `SchedulerService` or extend `SystemService`
- API handlers: `GET/POST/PUT/DELETE /routers/:id/system/schedulers` + `POST .../enable|disable`
- Scheduler fields: name, start-time, interval, on-event, disabled, comment

**Dependencies:** None — command defs already registered.

### 3.6 MEDIUM — Full Voucher Print Page

**What exists:**
- `TemplateService.Render()` renders header/row/footer fragments
- `VoucherHandler.PrintData()` returns vouchers + router info for a gencode
- `TemplateHandler.Render()` accepts template type + voucher data + renders

**What's needed:**
- New endpoint: `POST /api/v1/routers/:id/vouchers/print` that combines:
  1. Fetch cached vouchers by gencode
  2. Fetch router info (hotspot_name, dns_name, currency, etc.)
  3. Fetch profile metadata (validity, price)
  4. Render template (header + N×row + footer)
  5. Wrap in full HTML document with `<style>`, print trigger, page-break CSS
- Support `%qrCode%` placeholder with server-generated QR image data (base64)
- Default to appropriate template based on `template_type` parameter

**Dependencies:** Task 2 (default template seeding) should be done first.

### 3.7 MEDIUM — Logo Upload Endpoint

**What exists:**
- `Router` model has no logo field (mikhmon stores `logo-{session}.png` in filesystem)
- `RenderParams.Logo` in template service accepts a logo URL/path
- `VoucherTemplateVars.Logo` available in template rendering

**What's needed:**
- Add `LogoPath string` field to `Router` model + migration
- `POST /api/v1/routers/:id/logo` handler accepting `multipart/form-data` (PNG, max 1MB)
- Store in configurable directory (e.g., `./uploads/logos/`) or as base64 in DB
- Serve logo via `GET /api/v1/routers/:id/logo`
- Pass logo URL to template rendering

**Dependencies:** None — but needed for full print template parity.

### 3.8 MEDIUM — Enhanced Dashboard

**What exists:**
- `SystemService.GetDashboard()` returns: resource, identity, hotspot_users (count), active_sessions (count)
- `ReportService.GetDashboardSummary()` returns: today_count, today_sum, month_count, month_sum

**What mikhmon has:**
- System health (voltage, cpu-temperature)
- Traffic chart with interface selector
- Hotspot log (last N events)
- Income display (today + month)

**What's needed:**
- Extend `GetDashboard()` to include system health, traffic summary, and log tail
- Or create a single aggregated `GET /routers/:id/system/dashboard-full` that combines:
  - System resource ✅ (already included)
  - System health (voltage, temperature)
  - Hotspot active count ✅ (already included)
  - Dashboard summary (today/month income)
  - Last 10 hotspot log entries
  - Interface traffic summary (top 5 interfaces by current TX/RX)

**Dependencies:** None — all Bridge methods already exist.

### 3.9 LOW — Offline QR Code Generation

**What exists:**
- `TemplateService` generates QR via external API: `https://api.qrserver.com/v1/create-qr-code/`
- mikhmon uses client-side `qrious.min.js` (offline)

**What's needed:**
- Add Go QR library (e.g., `github.com/skip2/go-qrcode`)
- Replace external API call with local QR generation
- Return as base64-encoded PNG for embedding in templates

**Dependencies:** None.

### 3.10 LOW — Script Management API

**What exists:**
- `Bridge.ListScripts()` — list all scripts
- Mutation commands registered: `system/script/add`, `system/script/set`, `system/script/remove`, `system/script/run`
- Scripts used internally for: sales records (`comment=mikhmon`), quick print (`comment=QuickPrintMikhmon`)

**What's needed:**
- Bridge methods: `AddScript`, `SetScript`, `RemoveScript`, `RunScript`
- API handlers: `GET/POST/PUT/DELETE /routers/:id/system/scripts` + `POST .../run`
- Note: should filter out system-managed scripts (mikhmon/QuickPrint) from generic listing

**Dependencies:** None.

### 3.11 LOW — Logging Setup API Endpoint

**What exists:**
- `Bridge.SetupLogging()` at `internal/roskit/adapter/service/system.go:60` — creates logging action with prefix `->` and topics `hotspot,debug,info`
- **Never called** anywhere in the codebase (dead code)

**What's needed:**
- `POST /api/v1/routers/:id/system/setup-logging` handler
- Optionally call during router connection setup or expire monitor deployment

**Dependencies:** None.

### 3.12 LOW — Frontend Gaps

These are frontend-only concerns. The Go backend already exposes all necessary APIs.

| Feature | Status | Notes |
|---------|--------|-------|
| Theme switching | ❌ Missing | Frontend CSS concern. Backend needs no changes. |
| Multi-language (i18n) | ❌ Missing | Frontend JS concern. Backend responses are language-neutral. |
| Bluetooth printing | ❌ Missing | Frontend-only (Web Bluetooth API). |
| Voucher printing UI | ❌ Missing | Backend APIs exist (template CRUD, render, print-data). Frontend needs build. |
| User status/island page | ❌ Missing | Would need a new public endpoint (no auth) returning user connection info. |

---

## 4. Implementation Plan

Tasks ordered by dependency and priority. Each task follows roskit conventions: handler → service → Bridge → RouterOS.

---

### Task 1 — Fix Profile Price Lookup in On-Login Event [CRITICAL]

**Problem:** All sales recorded via on-login webhook have `price = 0`.

**Scope:**
- Replace `fetchProfilePrice` stub with actual DB lookup using `ProfilePriceMappingRepository`
- Update `VoucherSale` creation to include `SellingPrice`, `Validity` from mapping

**Files to create/modify:**
- `internal/api/handlers/event_handler.go` — inject `ProfilePriceMappingRepository`, replace stub
- `cmd/api/main.go` — pass repo to `NewEventHandler()`

**DB schema changes:** None — `profile_price_mappings` table already exists and is populated by worker.

**MikroTik commands via roskit:** None — reads from DB only.

**Redis usage:** None.

**Estimated complexity:** Low

---

### Task 2 — Seed Default Print Templates [HIGH]

**Problem:** After fresh install, no print templates exist. Users must create them manually.

**Scope:**
- Port 9 template parts from mikhmon (3 sizes × 3 parts: header/row/footer)
- Create seed function callable from migration or on-demand
- Add API endpoint to seed defaults for a router

**Files to create/modify:**
- `internal/services/template_service.go` — add `SeedDefaults(routerID)` method
- `internal/api/handlers/template_handler.go` — add `SeedDefaults` handler
- `internal/api/router.go` — register `POST /routers/:id/templates/seed-defaults`
- New file: `internal/services/template_defaults.go` — default template content constants

**DB schema changes:** None — uses existing `print_templates` table.

**MikroTik commands via roskit:** None.

**Redis usage:** Invalidate template cache after seeding.

**Estimated complexity:** Medium (content porting from PHP templates)

**Template content to port:**
- From `mikhmon/voucher/` or `mikhmon/template/`:
  - `header.default.txt` → header for "default" (230px, QR code slot)
  - `row.default.txt` → row for "default"
  - `footer.default.txt` → footer for "default"
  - `header.small.txt` → header for "small" (140px, compact)
  - `row.small.txt` → row for "small"
  - `footer.small.txt` → footer for "small"
  - `header.thermal.txt` → header for "thermal" (180px, QR + logo)
  - `row.thermal.txt` → row for "thermal"
  - `footer.thermal.txt` → footer for "thermal"

---

### Task 3 — Walled Garden CRUD API [HIGH]

**Problem:** Only read access to walled garden. No add/remove/update handlers.

**Scope:**
- Bridge methods for walled garden and walled garden IP CRUD
- Service layer (extend HotspotService)
- API handlers with full CRUD endpoints

**Files to create/modify:**
- `internal/roskit/adapter/service/hotspot.go` — add Bridge methods:
  - `AddWalledGarden(ctx, routerID, params) (string, error)`
  - `RemoveWalledGarden(ctx, routerID, id) error`
  - `SetWalledGarden(ctx, routerID, id, params) error`
  - `AddWalledGardenIP(ctx, routerID, params) (string, error)`
  - `RemoveWalledGardenIP(ctx, routerID, id) error`
  - `ListWalledGardenIP(ctx, routerID) ([]map[string]string, error)`
- `internal/services/hotspot_service.go` — add service methods
- `internal/api/handlers/hotspot_handler.go` — add handlers
- `internal/api/router.go` — register routes:
  ```
  GET    /routers/:id/hotspot/walled-garden
  POST   /routers/:id/hotspot/walled-garden
  DELETE /routers/:id/hotspot/walled-garden/:wid
  GET    /routers/:id/hotspot/walled-garden-ip
  POST   /routers/:id/hotspot/walled-garden-ip
  DELETE /routers/:id/hotspot/walled-garden-ip/:wid
  ```

**DB schema changes:** None — walled garden data lives on RouterOS.

**MikroTik commands via roskit:**
- `ip/hotspot/walled-garden/add|set|remove` (mutation defs already registered)
- `ip/hotspot/walled-garden/ip/add|remove` (mutation defs already registered)
- `ip/hotspot/walled-garden/ip/print` (query def already registered)

**Redis usage:** Cache A automatically updated by stream pipeline after mutations.

**Estimated complexity:** Medium

---

### Task 4 — Reset User Counters Endpoint [MEDIUM]

**Problem:** Bridge method exists but no HTTP endpoint exposed.

**Scope:**
- Add handler + service method + route for resetting user counters

**Files to create/modify:**
- `internal/services/hotspot_service.go` — add `ResetUserCounters(ctx, routerID, userID) error`
- `internal/api/handlers/hotspot_handler.go` — add `ResetUserCounters` handler
- `internal/api/router.go` — register `POST /routers/:id/hotspot/users/:id/reset-counters`

**DB schema changes:** None.

**MikroTik commands via roskit:** `ip/hotspot/user/reset-counters` (mutation def already registered).

**Redis usage:** None — counter reset doesn't affect cache.

**Estimated complexity:** Low

---

### Task 5 — Scheduler CRUD API Endpoints [MEDIUM]

**Problem:** Only list + expire monitor available. No generic scheduler management.

**Scope:**
- Bridge methods for generic scheduler CRUD
- Service methods (extend SystemService)
- API handlers with full CRUD

**Files to create/modify:**
- `internal/roskit/adapter/service/system.go` — add Bridge methods:
  - `AddScheduler(ctx, routerID, params) (string, error)`
  - `SetScheduler(ctx, routerID, id, params) error`
  - `RemoveScheduler(ctx, routerID, id) error`
  - `RunScheduler(ctx, routerID, id) error` (if applicable)
- `internal/services/system_service.go` — add service methods
- `internal/api/handlers/system_handler.go` — add handlers
- `internal/api/router.go` — register routes:
  ```
  POST   /routers/:id/system/schedulers
  PUT    /routers/:id/system/schedulers/:schedulerId
  DELETE /routers/:id/system/schedulers/:schedulerId
  POST   /routers/:id/system/schedulers/:schedulerId/enable
  POST   /routers/:id/system/schedulers/:schedulerId/disable
  ```

**DB schema changes:** None — schedulers live on RouterOS.

**MikroTik commands via roskit:**
- `system/scheduler/add|set|remove|enable|disable` (mutation defs already registered)

**Redis usage:** Cache A updated by stream pipeline.

**Estimated complexity:** Medium

---

### Task 6 — Full Voucher Print Page Endpoint [MEDIUM]

**Problem:** Template rendering produces fragments, not a complete printable HTML document.

**Scope:**
- New endpoint that combines voucher data + template rendering + HTML document wrapper
- Generate QR codes locally (base64 PNG)
- Page-break CSS for multi-voucher printing

**Files to create/modify:**
- `internal/api/handlers/voucher_handler.go` — add `PrintVouchers` handler
- `internal/services/voucher_service.go` — add `PrintVouchers(ctx, routerID, req)` method
- `internal/api/router.go` — register `POST /routers/:id/vouchers/print`

**New file:**
- `internal/services/print_page.go` — HTML document wrapper with CSS, print trigger, QR generation

**DB schema changes:** None.

**MikroTik commands via roskit:** None — uses cached vouchers + DB templates.

**Redis usage:** Reads cached vouchers (Cache B `VoucherSessionKey`).

**Estimated complexity:** Medium

**Dependencies:** Task 2 (default templates) should be done first.

---

### Task 7 — Logo Upload Endpoint [MEDIUM]

**Problem:** No mechanism to upload or store logo images per router.

**Scope:**
- Add logo path field to Router model
- Multipart upload handler
- Static file serving for logos

**Files to create/modify:**
- `internal/models/router.go` — add `LogoPath string` field
- `internal/repository/router_repo.go` — no changes needed (GORM auto-migrates)
- `internal/services/router_service.go` — add `UploadLogo(ctx, routerID, file) error`
- `internal/api/handlers/router_handler.go` — add `UploadLogo` and `GetLogo` handlers
- `internal/api/router.go` — register routes:
  ```
  POST /routers/:id/logo
  GET  /routers/:id/logo
  ```
- `migrations/006_router_logo.up.sql` — `ALTER TABLE routers ADD COLUMN logo_path TEXT DEFAULT ''`

**DB schema changes:** Add `logo_path` column to `routers` table.

**MikroTik commands via roskit:** None.

**Redis usage:** None.

**Estimated complexity:** Low

---

### Task 8 — Enhanced Dashboard [MEDIUM]

**Problem:** Current dashboard only has resource + identity + user counts. Missing health, income, traffic, log.

**Scope:**
- Extend dashboard endpoint to include all mikhmon dashboard data

**Files to create/modify:**
- `internal/services/system_service.go` — extend `GetDashboard()` to include:
  - `system_health` (voltage, temperature) via `bridge.GetSystemHealth()`
  - `income` (today_count, today_sum, month_count, month_sum) via `reportService.GetDashboardSummary()`
  - `recent_logs` (last 5 hotspot log entries) via `bridge.GetSystemLog(ctx, routerID, "hotspot")`
  - `top_interfaces` (top 5 by traffic from cache) via `bridge.ListInterfaces()` + cache
- `internal/api/handlers/system_handler.go` — inject `ReportService` dependency
- `internal/api/router.go` — update DI wiring

**DB schema changes:** None.

**MikroTik commands via roskit:** Uses existing Bridge methods only.

**Redis usage:** Reads from Cache A (telemetry) for traffic data.

**Estimated complexity:** Medium

---

### Task 9 — Offline QR Code Generation [LOW]

**Problem:** QR codes generated via external API (`api.qrserver.com`). Not offline-capable.

**Scope:**
- Replace external API with local Go QR library

**Files to create/modify:**
- `internal/services/template_service.go` — replace QR generation with local library
- `go.mod` / `go.sum` — add `github.com/skip2/go-qrcode` (or equivalent)

**DB schema changes:** None.

**MikroTik commands via roskit:** None.

**Redis usage:** None.

**Estimated complexity:** Low

---

### Task 10 — Script Management API [LOW]

**Problem:** No generic script management endpoints. Scripts are only managed implicitly (sales records, quick print).

**Scope:**
- Bridge methods for script CRUD + run
- API handlers with full CRUD
- Filter system-managed scripts from generic listing

**Files to create/modify:**
- `internal/roskit/adapter/service/system.go` — add Bridge methods:
  - `AddScript(ctx, routerID, params) (string, error)`
  - `SetScript(ctx, routerID, id, params) error`
  - `RemoveScript(ctx, routerID, id) error`
  - `RunScript(ctx, routerID, id) error`
- `internal/services/system_service.go` — add service methods
- `internal/api/handlers/system_handler.go` — add handlers
- `internal/api/router.go` — register routes:
  ```
  GET    /routers/:id/system/scripts
  POST   /routers/:id/system/scripts
  PUT    /routers/:id/system/scripts/:scriptId
  DELETE /routers/:id/system/scripts/:scriptId
  POST   /routers/:id/system/scripts/:scriptId/run
  ```

**DB schema changes:** None.

**MikroTik commands via roskit:**
- `system/script/add|set|remove|run` (mutation defs already registered)

**Redis usage:** None.

**Estimated complexity:** Medium

---

### Task 11 — Logging Setup Endpoint [LOW]

**Problem:** `Bridge.SetupLogging()` exists but is never called (dead code).

**Scope:**
- Expose as API endpoint
- Optionally auto-call during expire monitor deployment

**Files to create/modify:**
- `internal/api/handlers/system_handler.go` — add `SetupLogging` handler
- `internal/services/system_service.go` — add `SetupLogging` method
- `internal/api/router.go` — register `POST /routers/:id/system/setup-logging`
- Optionally: call `SetupLogging()` inside `DeployExpireMonitor()` automatically

**DB schema changes:** None.

**MikroTik commands via roskit:** `system/logging/print` + `system/logging/add` (defs already registered).

**Redis usage:** None.

**Estimated complexity:** Low

---

### Task 12 — User Status / Island Page [LOW]

**Problem:** No public endpoint for hotspot users to check their connection status. Mikhmon has a status page showing user info, time remaining, data usage.

**Scope:**
- New public (no auth) endpoint that takes MAC address or username
- Returns: user profile, time left, bytes used/total, expiry date

**Files to create/modify:**
- `internal/api/handlers/status_handler.go` — new handler
- `internal/services/status_service.go` — new service
- `internal/api/router.go` — register public route:
  ```
  GET /api/v1/status?mac={mac}&router={sessionName}  (public, no auth)
  ```
- Logic: find router by session name → query hotspot active by MAC → query hotspot user → return combined status

**DB schema changes:** None.

**MikroTik commands via roskit:**
- `ip/hotspot/active/print ?mac-address={mac}` (query def registered)
- `ip/hotspot/user/print ?name={username}` (query def registered)

**Redis usage:** Reads from Cache A (hotspot_active, hotspot_user).

**Estimated complexity:** Medium

---

## 5. Assumptions & Open Questions

### Assumptions

1. **Backend-only scope.** Frontend gaps (theme, i18n, Bluetooth printing, voucher printing UI) are deferred to a separate frontend implementation phase. This plan covers the Go backend API layer only.

2. **ProfilePriceMapping is the source of truth for on-login price lookup.** The worker syncs pricing every 5 minutes. There is a potential window where a new profile's pricing hasn't synced yet when a user logs in. We accept this trade-off.

3. **Walled Garden mutations only need domain-based and IP-based entries.** If additional walled garden types are needed, they can be added later.

4. **Logo storage uses local filesystem** (e.g., `./uploads/logos/`). If object storage (S3/MinIO) is needed, that's a separate change.

5. **Print templates use Go `html/template` syntax** (current approach). No plan to switch to a different templating engine.

6. **QR code URL format:** `http://{dnsName}/login?username={username}&password={password}` — same as mikhmon.

### Open Questions

1. **Should `SetupLogging` be auto-called during router connection?** Currently dead code. Auto-calling it when the engine starts streaming for a router would ensure hotspot logs are always captured, but it modifies router config without explicit user action.

2. **Should default templates be router-specific or global?** Current model has `router_id` on templates. Options:
   - (A) Seed defaults per router (each router gets its own copy, editable independently)
   - (B) Global defaults with `router_id = NULL` (shared, not editable per router)
   - Recommendation: (A) for consistency with current model

3. **User status endpoint — rate limiting?** The public `/status` endpoint could be abused. Should it be rate-limited? Should it require a token or captcha?

4. **Scheduler/Script management — should system-managed entries be protected?** The expire monitor scheduler (`Mikhmon-Expire-Monitor`) and quick print scripts (`QuickPrintMikhmon`) should not be editable/deletable through generic scheduler/script endpoints. Should we filter them out, or return them with a `system_managed: true` flag?

5. **Logo upload — max size and format?** Mikhmon accepts PNG only, max 1MB. Should we support SVG/WebP? Store in DB as base64 or filesystem?

6. **Print page endpoint — should it return HTML or PDF?** HTML with print CSS is simpler and matches mikhmon. PDF would require a library like `wkhtmltopdf` or Go-based PDF generation.

7. **Dashboard enhancement — how much data to aggregate?** Including traffic stats for all interfaces could be expensive. Should we limit to top N interfaces? Cache the aggregated dashboard response?

8. **On-login event — should we record to RouterOS scripts too?** Currently only records to PostgreSQL. Mikhmon records to both RouterOS scripts AND the database (via fetch). Should we mirror back to RouterOS for compatibility with the legacy report viewer?

9. **Voucher print — Bluetooth/thermal printer support?** Mikhmon v3 has Bluetooth printing integration. This is frontend-only but the backend may need to provide formatted data (raw text/ESC-P commands) instead of HTML.

10. **Report mode — what does `ReportMode` actually do?** The Router model has `ReportMode` (default "disable") matching mikhmon's per-router setting, but it's never used in the backend logic. Should it control whether sales are recorded via on-login webhook?
