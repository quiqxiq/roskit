# Roskit Architecture & Developer Guide

This document provides a deep dive into the architecture, design patterns, and internal workings of the Roskit backend.

---

## Architecture Overview

Roskit is a **multi-tenant SaaS** with two layered concerns:

1. **Application layer** — tenant-aware business logic for hotspot/voucher management.
2. **Roskit subsystem** (`internal/roskit/`) — behavior-classified RouterOS engine, isolated from application code.

Services NEVER touch the TCP connection pool or execution layer directly. They communicate exclusively through the `Bridge` adapter.

### Top-level Layout

```text
internal/
├── api/             # HTTP layer
│   ├── handlers/    # Gin handlers, one per resource
│   └── middleware/  # Auth, Tenant, Casbin, RouterTenant, Audit, RateLimit
├── casbin/          # RBAC enforcer + role policies (model.conf, policies.go)
├── config/          # Env loader
├── models/          # GORM models (tenant-aware)
├── repository/      # Tenant-scoped queries
├── services/        # Business logic, depends on Bridge
├── roskit/          # See subsystem layout below
└── worker/          # Background jobs (cross-tenant, platform-level)
```

### Subsystem Structure (`internal/roskit/`)

```text
internal/roskit/
├── core/           # PURE — no Redis, no Influx, no network
│   ├── command/    # Global registry (Register/Lookup/ByType)
│   ├── definition/ # Registers all commands via init()
│   ├── parser/     # Parses RouterOS reply maps into structured Go types
│   └── model/      # Domain models for parsed data
├── behavior/       # HOW commands behave
│   ├── stream/     # Persistent listen/follow workers with reconnection
│   ├── poll/       # Periodic polling workers
│   ├── query/      # Redis-first reads, fallback to live RouterOS
│   ├── mutation/   # Write operations (add/set/remove)
│   └── interfaces.go # StreamHandler, QueryHandler, etc.
├── execution/      # RouterOS TCP I/O only
│   ├── pool.go     # Pool (Register/Unregister/Borrow/Return)
│   └── executor.go # Low-level Run/Add/Set/Remove sentence helpers
├── pipeline/       # Redis + InfluxDB integrations
│   ├── cache/      # RedisRepository (key prefix: roskit:{routerID}:{measurement}:{id})
│   ├── event/      # Processor (update → cache + timeseries + pubsub)
│   ├── pubsub/     # Publisher (broadcast) + Subscriber (SSE feed)
│   └── timeseries/ # Writer + Reader interfaces for InfluxDB
├── orchestrator/   # engine.go (lifecycle), dispatcher.go (command routing)
└── adapter/service/ # Bridge — the single service-layer entry point
```

---

## Multi-Tenancy

### Schema relations

```
tenants (1)
├── tenant_settings (1:1, CASCADE)
├── users (1:N, CASCADE)            ← tenant_id nullable: NULL = superadmin
├── routers (1:N, CASCADE)
│   └── profile_price_mappings (1:N, CASCADE from router)
├── voucher_sales (1:N, CASCADE)    ← router_id is *uint, SET NULL on router delete
├── print_templates (1:N, CASCADE)  ← tenant_id nullable: NULL = global default
└── audit_logs (1:N, CASCADE)
```

### Tenant identification

- Each tenant has a unique `slug` (lowercase, alphanumeric + dashes). The slug is the **Casbin domain** for that tenant.
- Reserved slug `__platform__` (constant `models.PlatformTenantSlug`) represents the platform-level domain used by superadmin for cross-tenant operations.
- JWT claims carry `TenantID *uint` and `TenantSlug string`. For superadmin, `TenantID` is nil.

### Tenant deletion

Soft delete (`status = suspended`) preserves data and access can be restored. Hard delete uses `db.Unscoped().Delete(&tenant)` to bypass GORM soft-delete and trigger FK CASCADE on every child table. The service exposes both via `TenantService.Suspend` / `TenantService.HardDelete`.

### Two-table cascade nuance

- `voucher_sales.router_id` is `*uint` with `OnDelete:SET NULL`. Deleting a router preserves financial records but unlinks them from the (now-gone) router.
- `audit_logs.user_id` is `*uint` and **does not** cascade. Logs survive user deletion.
- Everything else cascades fully when a tenant is hard-deleted.

---

## Authentication & Casbin RBAC

### Auth flow

1. `POST /auth/login` — body `{tenant?, username, password}`. The `tenant` field is omitted only for superadmin (whose `users.tenant_id` is `NULL`).
2. `AuthService.Login`:
   - Resolves the tenant via `TenantRepo.GetBySlug` (rejects suspended tenants).
   - Looks up the user via `UserRepo.GetByTenantUsername(tenantID, username)`.
   - Issues access (15 min) + refresh (7 days) JWTs with claims: `uid`, `sub`, `role`, `tid`, `tslug`, `jti`.
3. `POST /auth/setup` is allowed only when `users` table is empty. It creates a `Tenant`, `TenantSettings`, an `owner` user, and copies all global default templates into the tenant — atomically inside one DB transaction.

### Middleware chain

For every protected route the chain runs in this order:

```
AuthMiddleware       → validates JWT, sets userID/role/tokenID + jwtTenantID/jwtTenantSlug
TenantMiddleware     → resolves active tenant (overridable for superadmin via X-Tenant-Slug)
CasbinMiddleware     → enforces RBAC using (userID, tenantSlug, route, method)
RouterTenantMiddleware → on /routers/:routerId/* groups, validates router belongs to tenant (returns 404 if not)
```

### Casbin model

`internal/casbin/model.conf` is a domain-based RBAC with `keyMatch2` for route patterns:

```ini
[matchers]
m = g(r.sub, p.sub, r.dom) && (p.dom == "*" || r.dom == p.dom)
    && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

Policies in `internal/casbin/policies.go` are seeded idempotently at startup by `casbinx.SeedPolicies`. Roles use domain `*` so a single policy row applies to every tenant. Per-user role grants live in the `g` table with the tenant slug as domain (e.g. `g, 42, owner, my-hotspot`).

| Role | Coverage |
|---|---|
| `superadmin` | `p, superadmin, *, /api/v1/*, *` (everything) |
| `owner` | `/tenant/*`, `/users/*`, `/routers/*`, `/templates/*` (full) |
| `admin` | Same as owner except `/tenant/*` |
| `staff` | Read-only on routers/templates; full on hotspot users + vouchers + reports (read) |

### Role grants

When `AuthService.CreateUser` succeeds, it calls `casbinx.AssignRole(enforcer, userID, tenantSlug, role)` to insert the `g` row. `UserHandler.Update` revokes-and-reassigns when the role changes; `UserHandler.Delete` revokes all roles.

---

## Data Flow & TCP Connections

### TCP Connection Multiplexing

Every registered router maintains exactly **two** persistent TCP connections to RouterOS:
1. `sync` (Client Connection) — sequential queries, polling, mutations.
2. `async` (Stream Connection) — used exclusively by stream workers, tag-multiplexed via `go-routeros` so dozens of `=follow` listeners share a single TCP socket.

This keeps the RouterOS connection count well under the default limit of 20 even for tenants with many routers.

### Read/Write Paths

- **Write Path (Telemetry):** RouterOS TCP → `execution.Pool` → `stream/poll worker` → `parser` → `pipeline/event.Processor` → (Redis HSET + InfluxDB + Redis PUBLISH).
- **Read Path (API Queries):** `service.Bridge.Query()` → `orchestrator.Dispatcher` → `behavior.query.Handler` → Redis first, fallback to live RouterOS.

### Engine seeding across tenants

`RouterService.SeedEngineFromDB` calls `routerRepo.ListAll(ctx)` (cross-tenant) at startup, registering every router in the engine pool. Routers are indexed by ID alone — tenant scoping happens at the repository/handler layer, not at the engine.

---

## Dependency Injection (DI)

The application wires dependencies in `cmd/api/main.go`:

```go
config.Load()
  → database.Connect() + redis.Connect()
  → casbinx.NewEnforcer(db) + casbinx.SeedPolicies(enforcer)   // RBAC
  → repository.NewTenantRepo / TenantSettingsRepo / UserRepo / TemplateRepo / SaleRepo
  → services.NewTenantService(db, tenantRepo, settingsRepo, templateRepo)
  → services.NewAuthService(userRepo, tenantRepo, enforcer, cache, jwtSecret, refreshSecret)
  → roskitcache.NewRedisRepository()      // Telemetry cache
  → roskitpubsub.NewRedisSubscriber()     // SSE subscriber
  → orchestrator.New(...)                 // engine
  → service.NewBridge(engine.Dispatcher(), cacheRepo)
  → services.NewRouterService(routerRepo, engine, cache, aesKey)
  → routerSvc.SeedEngineFromDB() → engine.Start()
  → api.NewRouter(cfg, db, cache, bridge, routerSvc, tsReader, subscriber,
                  profileRepo, tenantRepo, tenantSettingsRepo, tenantSvc, authSvc, enforcer)
```

**CRITICAL RULES:**
1. Business services must inject `*service.Bridge` only — never `*execution.Pool` or `*routeros.Client`.
2. Services that touch tenant-scoped data must accept `tenantID uint` (or `*uint` for global/optional) as a parameter — never read it from package-level state.
3. Handlers extract `tenantID` from `gin.Context` via the helpers in `internal/api/handlers/context.go` (`tenantIDFromCtx` / `optionalTenantIDFromCtx`).

---

## Two Cache Systems

| | Cache A (RouterOS Telemetry) | Cache B (App Cache) |
|---|---|---|
| **Package** | `internal/roskit/pipeline/cache` | `pkg/redis/cache.go` |
| **Key Pattern** | `roskit:{routerID}:{measurement}:{id}` | `mikhmon:{resource}:{id}` |
| **Managed By** | Streaming pipeline (auto, 5min TTL) | Service layer (manual) |
| **Invalidation** | NEVER manually — TTL handles it | After writes via `cache.Delete()` |

Tenant scoping is implicit in App Cache keys because reports use composite keys like `t{tenantID}:{scope}:day:{date}`. Cache A keys remain router-indexed because telemetry is router-local.

---

## Core Development Rules

1. **`core/` is Pure:** No Redis, no Influx, no network imports allowed in `internal/roskit/core/`.
2. **Type Safety:** Router IDs are `string` in the Bridge/Pool (e.g. `"42"`), `uint` in DB and API. Convert with `fmt.Sprintf("%d", router.ID)`.
3. **Command Registry:** Global. New RouterOS commands must call `command.Register()` in `internal/roskit/core/definition/` via `init()`.
4. **Auto-Discovery:** Calling `engine.AddRouter()` starts all registered stream/poll workers automatically — no manual spec lists.
5. **Database Models:** PostgreSQL stores ONLY application configuration: tenants, tenant settings, users, routers, profile mappings, voucher sales, audit logs, templates. **NEVER create GORM models for transient RouterOS data** like hotspot users, queues, interfaces, or active sessions.
6. **Tenant isolation:** Every repository method that touches a tenant-owned table must filter by `tenant_id`. The only exceptions are `RouterRepo.ListAll` and `RouterRepo.GetByIDAny` — used exclusively by the engine seeder and worker for cross-tenant operations.
7. **Casbin sync:** Any change to a user's role MUST also revoke and re-assign the Casbin grant. This is centralized in `UserHandler.Update`.

---

## Command Definition Domains

| File | Domain | Worker Type |
|---|---|---|
| `hotspot.go` | `ip/hotspot/*` | Stream + Mutation |
| `network.go` | `ip/address`, `route`, `arp`, `dhcp-*`, `dns`, `pool` | Stream + Poll |
| `ppp.go` | `ppp/secret`, `ppp/active`, `ppp/profile` | Stream + Mutation |
| `firewall.go` | `ip/firewall/filter`, `address-list`, `connection` | Stream + Poll |
| `interface.go` | `interface/vlan`, `bridge`, `wireless`, `wireguard` | Stream + Poll |
| `system.go` | `system/resource`, `health`, `script`, `ntp` | Poll + Stream |
| `user_manager.go` | `user-manager/user`, `session`, `profile` | Stream + Poll |

**Stream vs Poll Decision Matrix:**
- **Stream:** Frequently changing data (e.g., `hotspot/active`, `firewall/connection`). Uses `=follow`.
- **Poll (with follow):** Admin configs that rarely change (e.g., `firewall/filter`, `interface/vlan`).
- **Poll (no follow):** Metrics that don't support `=follow` (e.g., `system/resource`).

---

## Domain Knowledge & Quirks

### RouterOS Scripts & Syntax
- **Script Variables:** Use `$"mac-address"` (not `$mac`) and `$user` (not `$username`).
- **Sales Records:** Recorded in `/system/script` with format
  `name = "{date}-|-{time}-|-{user}-|-{price}-|-{ip}-|-{mac}-|-{validity}-|-{profile}-|-{comment}"`,
  `owner = "{Month}{Year}"`, `comment = "mikhmon"`.
- **Profile Pricing:** Injected via on-login `:put`:
  `:put (",{expmode},{price},{validity},{sprice},,{lockuser},{lockserver},")`

### Voucher Types & Comments
- **Types:** `vc` (username equals password), `up` (separate username/password).
- **User Comment Formats** (parsed by frontend, do NOT change):
  - Expiry: `mmm/dd/yyyy hh:mm:ss N` or `mmm/dd/yyyy hh:mm:ss X`
  - Voucher: `vc-{code}-{MM.DD.YY}-{text}` or `up-{code}-{MM.DD.YY}-{text}`
- **Expiration Modes:** `ntf` (notify), `ntfc` (notify+record), `rem` (remove), `remc` (remove+record), `0`+price (none).

### On-Login Webhook & Tenant Resolution

RouterOS calls `/api/v1/events/on-login` via `/tool/fetch` (form-encoded, not JSON). Tenant resolution flow:
1. Reads `X-Router-Token` header (or `token` form field).
2. `tenantSettingsRepo.GetByWebhookToken(token)` returns the matching `TenantSettings`.
3. `routerRepo.GetByName(tenantID, router_name)` resolves the router under that tenant.
4. Idempotency key prevents double-recording.

This means **every tenant has one webhook token** shared across its routers. Rotate via `PUT /tenant/settings`.

### Hotspot User Count
Returns `count - 1` to exclude the admin user, mirroring legacy PHP behavior.

### Legacy Migrations
The legacy XOR `"128"` cycled base64 encryption is decoded by `cmd/migrate import`. It auto-creates one tenant per session in the legacy `config.php` (slug = sanitized session name) and tags routers + sales with that tenant.

---

## Migrations

The four embedded SQL files in `migrations/*.sql` are no longer used directly. Schema is managed by GORM `AutoMigrate()` invoked from `cmd/migrate up`. The migration order respects FK dependencies:

```go
db.AutoMigrate(
    &models.Tenant{},
    &models.TenantSettings{},
    &models.User{},
    &models.Router{},
    &models.ProfilePriceMapping{},
    &models.VoucherSale{},
    &models.PrintTemplate{},
    &models.AuditLog{},
)
```

The Casbin policy table (`casbin_rule`) is auto-created by `gorm-adapter` on first enforcer instantiation. Policy rows are seeded idempotently by `casbinx.SeedPolicies`.
