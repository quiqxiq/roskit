# Roskit Architecture & Developer Guide

This document provides a deep dive into the architecture, design patterns, and internal workings of the Roskit backend.

---

## Architecture Overview

Roskit is designed with a **behavior-classified architecture** rather than a traditional MVC structure. The core engine (`internal/roskit/`) is isolated from the application's business logic (`internal/services/`).

Services NEVER touch the TCP connection pool or execution layer directly. They communicate exclusively through the `Bridge` adapter.

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
│   └── interfaces.go # Key interfaces: StreamHandler, QueryHandler, etc.
├── execution/      # RouterOS TCP I/O only
│   ├── pool.go     # Pool (Register/Unregister/Borrow/Return)
│   └── executor.go # Low-level Run/Add/Set/Remove sentence helpers
├── pipeline/       # Redis + InfluxDB Integrations
│   ├── cache/      # RedisRepository (key prefix: roskit:{routerID}:{measurement}:{id})
│   ├── event/      # Processor (update → cache + timeseries + pubsub)
│   ├── pubsub/     # Publisher (broadcast) + Subscriber (SSE feed)
│   └── timeseries/ # Writer + Reader interfaces for InfluxDB
├── orchestrator/   # engine.go (lifecycle), dispatcher.go (command routing)
└── adapter/service/ # Bridge — the single service-layer entry point
```

---

## Data Flow & TCP Connections

### TCP Connection Multiplexing

Every registered router maintains exactly **two** persistent TCP connections to RouterOS:
1. `sync` (Client Connection): Used for sequential queries, polling, and mutations.
2. `async` (Stream Connection): Used exclusively by stream workers. This connection is tag-multiplexed using `go-routeros`, allowing dozens of real-time `=follow` listeners to share a single TCP socket safely.

This architecture prevents connection exhaustion, keeping the RouterOS connection count far below the default limit of 20.

### Read/Write Paths

- **Write Path (Telemetry):** RouterOS TCP → `execution.Pool` → `stream/poll worker` → `parser` → `pipeline/event.Processor` → (Redis HSET + InfluxDB + Redis PUBLISH).
- **Read Path (API Queries):** `service.Bridge.Query()` → `orchestrator.Dispatcher` → `behavior.query.Handler` → Checks Redis first, falls back to live RouterOS if missed.

---

## Dependency Injection (DI)

The application wires dependencies together at startup in `cmd/api/main.go`.

```go
config.Load()
  → database.Connect() + redis.Connect()
  → roskitcache.NewRedisRepository()     // Telemetry cache
  → roskitpubsub.NewRedisSubscriber()    // SSE subscriber
  → orchestrator.New(...)
  → service.NewBridge(engine.Dispatcher(), cacheRepo)  // Inject THIS into services
  → services.NewRouterService(repo, engine, cache, aesKey)
  → routerSvc.SeedEngineFromDB()  → engine.Start()
  → api.NewRouter(...)
```

**CRITICAL RULE:** Business services in `internal/services/` must ONLY inject `*service.Bridge`. They must never inject `*execution.Pool` or `*routeros.Client`.

---

## Two Cache Systems

Roskit utilizes two distinct caching systems that must never be confused:

| | Cache A (RouterOS Telemetry) | Cache B (App Cache) |
|---|---|---|
| **Package** | `internal/roskit/pipeline/cache` | `pkg/redis/cache.go` |
| **Key Pattern** | `roskit:{routerID}:{measurement}:{id}` | `mikhmon:{resource}:{id}` |
| **Managed By** | Streaming pipeline (auto, 5min TTL) | Service layer (manual) |
| **Invalidation** | NEVER manually — TTL handles it | After writes via `cache.Delete()` |

---

## Core Development Rules

1. **`core/` is Pure:** No Redis, no Influx, no network imports allowed in `internal/roskit/core/`.
2. **Type Safety:** Router IDs are strictly `string` in the Bridge/Pool (e.g., `"42"`), but `uint` in API URL params. Convert using `fmt.Sprintf("%d", router.ID)`.
3. **Command Registry:** The command registry is global. New RouterOS commands must be registered by calling `command.Register()` in `internal/roskit/core/definition/` via an `init()` block.
4. **Auto-Discovery:** The engine auto-discovers commands. Calling `engine.AddRouter()` automatically starts all registered stream/poll workers. No manual spec lists are needed.
5. **Database Models:** The PostgreSQL database stores ONLY application configuration: voucher sales, router configs, system users, audit logs, and templates. **NEVER create GORM models for transient data** like hotspot_users, queues, interfaces, or active_sessions.

---

## Command Definition Domains

RouterOS commands are grouped by domain in `internal/roskit/core/definition/`. When adding new commands, categorize them logically:

| File | Domain | Worker Type |
|---|---|---|
| `hotspot.go` | `ip/hotspot/*` | Stream + Mutation |
| `network.go` | `ip/address`, `route`, `arp`, `dhcp-*`, `dns`, `pool` | Stream + Poll |
| `ppp.go` | `ppp/secret`, `ppp/active`, `ppp/profile` | Stream + Mutation |
| `firewall.go` | `ip/firewall/filter`, `address-list`, `connection` | Stream + Poll |
| `interface.go` | `interface/vlan`, `bridge`, `wireless`, `wireguard` | Stream + Poll |
| `system.go` | `system/resource`, `health`, `script`, `ntp` | Poll + Stream |
| `user_manager.go`| `user-manager/user`, `session`, `profile` | Stream + Poll |

**Stream vs Poll Decision Matrix:**
- **Stream:** Data changes frequently and intense polling is inefficient (e.g., `hotspot/active`, `firewall/connection`). Uses `=follow`.
- **Poll (with follow):** Admin-only configs that rarely change (e.g., `firewall/filter`, `interface/vlan`).
- **Poll (no follow):** Metrics that do not support the `=follow` modifier (e.g., `system/resource`).

---

## Domain Knowledge & Quirks

When interacting with RouterOS, keep these specific implementation details in mind:

### RouterOS Scripts & Syntax
- **Script Variables:** Ensure you use `$"mac-address"` (not `$mac`) and `$user` (not `$username`).
- **Sales Records:** Sales are recorded in `/system/script` with the following format:
  `name = "{date}-|-{time}-|-{user}-|-{price}-|-{ip}-|-{mac}-|-{validity}-|-{profile}-|-{comment}"`
  `owner = "{Month}{Year}"`, `comment = "mikhmon"`
- **Profile Pricing:** Injected via on-login `:put` statement:
  `:put (",{expmode},{price},{validity},{sprice},,{lockuser},{lockserver},")`

### Voucher Types & Comments
- **Types:** `vc` (username equals password), `up` (username does not equal password).
- **User Comment Formats:** The frontend parses comments by exact position. **DO NOT change these formats.**
  - Expiry: `mmm/dd/yyyy hh:mm:ss N` or `mmm/dd/yyyy hh:mm:ss X`
  - Voucher: `vc-{code}-{MM.DD.YY}-{text}` or `up-{code}-{MM.DD.YY}-{text}`
- **Expiration Modes:** `ntf` (notify), `ntfc` (notify+record), `rem` (remove), `remc` (remove+record), `0`+price (none).

### Miscellaneous
- **On-Login Recording:** RouterOS uses `/tool/fetch` to send form data (NOT JSON) to the `/api/v1/events/on-login` webhook.
- **Hotspot User Count:** Returns `count - 1` to exclude the admin user, mirroring legacy PHP behavior.
- **Legacy Migrations:** If handling legacy PHP migrations, be aware of the XOR `"128"` cycled base64 encryption. New configurations should avoid this and rely on HTTPS+JSON.

---

## Migrations

Four SQL files reside in `migrations/` and are embedded directly into the binary via `//go:embed *.sql`. 
Run migrations via `cmd/migrate/main.go` using GORM's `AutoMigrate()`. We do **NOT** use `golang-migrate`.
