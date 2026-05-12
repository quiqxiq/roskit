# roskit — AGENTS.md

Go backend for MikroTik hotspot management. Module: `github.com/quiqxiq/roskit`. Stack: Go 1.24, Gin, GORM, PostgreSQL, Redis.

## Build & Run

```bash
go build ./...                        # must pass after every change
go run ./cmd/api                      # API server (port 8080)
go run ./cmd/migrate up               # run DB schema migrations (GORM AutoMigrate)
go run ./cmd/migrate import --config-file=PATH [--dry-run] [--router=name]
go run ./cmd/migrate sync-profiles [--router=name]
go run ./cmd/worker                   # background worker
```

Makefile targets: `make build-api`, `make build-migrate`, `make build-worker`, `make migrate-up`, `make test`.

Docker dev stack: `make docker-up` (uses `docker/docker-compose.dev.yml` — PostgreSQL 16, Redis 7, InfluxDB 3).

## Tests

Go tests: only `pkg/encrypt/encrypt_test.go` exists. Run with `go test -race ./...`.

Python HTTP integration tests in `tests/http/` (pytest). Requires running API server + RouterOS device.

## Architecture

### Roskit subsystem (`internal/roskit/`)

Behavior-classified design. Services NEVER touch the pool or execution layer directly.

```
internal/roskit/
├── core/           # PURE — no Redis, no Influx, no network
│   ├── command/    # Global registry (Register/Lookup/ByType). Commands auto-registered via init() in definition/
│   ├── definition/ # Registers all commands by calling command.Register() — import with blank identifier
│   ├── parser/     # Parses RouterOS reply maps into structured types
│   └── model/      # Domain models for parsed data
├── behavior/       # HOW commands behave
│   ├── stream/     # Persistent listen/follow workers with reconnection
│   ├── poll/       # Periodic polling workers
│   ├── query/      # Redis-first reads, fallback to live RouterOS
│   ├── mutation/   # Write operations (add/set/remove)
│   └── interfaces.go # Key interfaces: StreamHandler, QueryHandler, MutationHandler, Dispatcher
├── execution/      # RouterOS TCP I/O only
│   ├── pool.go     # Pool (Register/Unregister/Borrow/Return) + RouterConn (Connect/ConnectWithBackoff)
│   └── executor.go # Low-level Run/Add/Set/Remove sentence helpers
├── pipeline/       # Redis + InfluxDB
│   ├── cache/      # RedisRepository (10 methods) — key prefix: roskit:{routerID}:{measurement}:{id}
│   ├── event/      # Processor (update→cache+timeseries+pubsub) + Aggregator (active/inactive tracking)
│   ├── pubsub/     # Publisher (broadcast) + Subscriber (SSE feed)
│   └── timeseries/ # Writer + Reader interfaces for InfluxDB
├── orchestrator/   # engine.go (lifecycle), dispatcher.go (command routing), router_manager.go
└── adapter/service/ # Bridge — the single service-layer entry point
```

### DI wiring (cmd/api/main.go)

```
config.Load()
  → database.Connect() + redis.Connect()
  → roskitcache.NewRedisRepository()     // telemetry cache
  → roskitpubsub.NewRedisSubscriber()    // SSE subscriber
  → orchestrator.New(Config{Cache, TimeSeries, PubSub})
  → service.NewBridge(engine.Dispatcher(), cacheRepo)  ← inject THIS into services
  → services.NewRouterService(repo, engine, cache, aesKey)
  → routerSvc.SeedEngineFromDB()  → engine.Start()
  → api.NewRouter(cfg, db, cache, bridge, routerSvc, tsReader, subscriber)
```

### Two cache systems — never confuse them

| | Cache A (RouterOS telemetry) | Cache B (App cache) |
|---|---|---|
| **Package** | `internal/roskit/pipeline/cache` | `pkg/redis/cache.go` |
| **Key** | `roskit:{routerID}:{measurement}:{id}` | `mikhmon:{resource}:{id}` |
| **Managed by** | Streaming pipeline (auto, 5min TTL) | Service layer (manual) |
| **Invalidation** | NEVER manually — TTL handles it | After writes via `cache.Delete()` |

### Data flow

- **Write path**: RouterOS TCP → execution.Pool → stream/poll worker → parser → event.Processor → Redis HSET + InfluxDB + Redis PUBLISH
- **Read path**: Bridge.Query() → Dispatcher → query.Handler → Redis first, fallback to live RouterOS

## Key Rules

1. **Services inject `*service.Bridge`** — never `*execution.Pool`, never raw `*routeros.Client`
2. **Router IDs are `string`** in Bridge/Pool (e.g., `"42"`), but `uint` in API URL params — convert via `fmt.Sprintf("%d", router.ID)`
3. **`core/` is pure** — no Redis, no Influx, no network imports allowed
4. **`execution/` is the only layer** that imports `go-routeros`
5. **`pipeline/` is the only layer** that imports Redis/InfluxDB clients
6. **Command registry is global** — new commands must call `command.Register()` in `core/definition/` via `init()`
7. **Engine auto-discovers commands** — `engine.AddRouter()` starts all registered stream/poll workers; no manual spec lists needed
8. **TestConnection is standalone** — uses `routeros.DialContext()` directly, not through engine/pool
9. **DB stores ONLY**: voucher sales, router configs, system users, audit logs, templates. **NEVER create GORM models for**: hotspot_users, queues, interfaces, active_sessions

## Coding Conventions

- `context.Context` as first param on all functions
- JSON response envelope: `{"data": ..., "error": null}` or `{"data": null, "error": "msg"}`
- Use `AppError` from `pkg/errors` for all error responses (`NewNotFound`, `NewUnauthorized`, `NewBadRequest`, `NewInternal`)
- Layers: Handler (bind/validate/respond) → Service (orchestrate via Bridge + repo + cache B) → Repository (pure GORM)
- No raw SQL unless aggregation requires it; no global variables; no panics in goroutines without recovery
- `NO_COMMENTS` unless explicitly asked

## Migrations

4 SQL files in `migrations/` (embedded via `//go:embed *.sql`). Run via `cmd/migrate/main.go` using GORM `AutoMigrate()`, NOT golang-migrate.

## Domain Knowledge

**Sales records** in RouterOS `/system/script`:
- `name = "{date}-|-{time}-|-{user}-|-{price}-|-{ip}-|-{mac}-|-{validity}-|-{profile}-|-{comment}"`
- `owner = "{Month}{Year}"`, `comment = "mikhmon"`

**Profile pricing** in on-login `:put` statement:
`:put (",{expmode},{price},{validity},{sprice},,{lockuser},{lockserver},")`

**Voucher types**: `vc` (username=password), `up` (username≠password)

**On-login recording**: `/tool/fetch` sends form data (NOT JSON) to `/api/v1/events/on-login`

**RouterOS script variables**: `$"mac-address"` (not `$mac`), `$user` (not `$username`)

**Legacy PHP encryption** (for data migration):
- `enc_rypt`/`dec_rypt`: XOR key `"128"` cycled, then base64
- `blah()`/`unblah()`: `base64decode → XOR 10 → double base64decode`
- `jsEncode`: do NOT replicate — use HTTPS+JSON

**Hotspot user count**: returns `count - 1` to exclude admin user (matches PHP behavior)

**User comment formats** (frontend parses by position — DO NOT change):
- Expiry: `mmm/dd/yyyy hh:mm:ss N` or `mmm/dd/yyyy hh:mm:ss X`
- Voucher: `vc-{code}-{MM.DD.YY}-{text}` or `up-{code}-{MM.DD.YY}-{text}`

**On-login expiration modes**: `ntf` (notify), `ntfc` (notify+record), `rem` (remove), `remc` (remove+record), `0`+price (none)

<!-- code-review-graph MCP tools -->
## MCP Tools: code-review-graph

**IMPORTANT: This project has a knowledge graph. ALWAYS use the
code-review-graph MCP tools BEFORE using Grep/Glob/Read to explore
the codebase.** The graph is faster, cheaper (fewer tokens), and gives
you structural context (callers, dependents, test coverage) that file
scanning cannot.

### When to use graph tools FIRST

- **Exploring code**: `semantic_search_nodes` or `query_graph` instead of Grep
- **Understanding impact**: `get_impact_radius` instead of manually tracing imports
- **Code review**: `detect_changes` + `get_review_context` instead of reading entire files
- **Finding relationships**: `query_graph` with callers_of/callees_of/imports_of/tests_for
- **Architecture questions**: `get_architecture_overview` + `list_communities`

Fall back to Grep/Glob/Read **only** when the graph doesn't cover what you need.

### Key Tools

| Tool | Use when |
| ------ | ---------- |
| `detect_changes` | Reviewing code changes — gives risk-scored analysis |
| `get_review_context` | Need source snippets for review — token-efficient |
| `get_impact_radius` | Understanding blast radius of a change |
| `get_affected_flows` | Finding which execution paths are impacted |
| `query_graph` | Tracing callers, callees, imports, tests, dependencies |
| `semantic_search_nodes` | Finding functions/classes by name or keyword |
| `get_architecture_overview` | Understanding high-level codebase structure |
| `refactor_tool` | Planning renames, finding dead code |

### Workflow

1. The graph auto-updates on file changes (via hooks).
2. Use `detect_changes` for code review.
3. Use `get_affected_flows` to understand impact.
4. Use `query_graph` pattern="tests_for" to check coverage.
