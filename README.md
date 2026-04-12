# RosKit

**Real-time MikroTik RouterOS telemetry streaming to InfluxDB 3 Core & Redis.**

RosKit is a Go library that streams live telemetry from MikroTik routers using the RouterOS API, persists time-series metrics to InfluxDB 3 Core, and caches the latest state in Redis for instant access by dashboards and AI agents.

---

## Features

- **Multi-Router Connection Pool** — Each router gets a dedicated persistent TCP connection with independent health monitoring and auto-reconnect.
- **Streaming Specs** — Modular, pluggable data sources using the Command pattern. Add new metrics by creating a single file.
- **InfluxDB 3 Core** — Async batched writes (configurable batch size & flush interval) for high-throughput time-series storage.
- **Redis Snapshots** — Latest state cached via `HSET` for instant reads without querying the router.
- **Redis Pub/Sub** — Real-time event broadcasting for dashboards and alerting systems.
- **Dead Entity Tracking** — Automatic cache cleanup when users disconnect or interfaces are removed (`.dead=yes`).
- **Per-Router Config** — Independent reconnect intervals, health check frequency, dial timeouts per router.
- **Clean Architecture** — Domain → UseCase → Repository layers with dependency injection.

## Architecture

```
MikroTik Routers                          Storage
┌─────────────┐                          ┌──────────────────┐
│  core-01    │──TCP──┐                  │  InfluxDB 3 Core │
│  edge-01    │──TCP──┤   ┌──────────┐   │  (time-series)   │
│  ap-01      │──TCP──┼──▶│  RosKit  │──▶├──────────────────┤
└─────────────┘       │   │  Engine  │   │  Redis 7         │
                      │   └──────────┘   │  (cache + pubsub)│
                      │                  └──────────────────┘
                      │
              ConnPool (1 persistent connection per router)
              ├── RouterConn["core-01"] → health check every 30s
              ├── RouterConn["edge-01"] → health check every 15s
              └── RouterConn["ap-01"]   → health check every 60s
```

## Project Structure

```
roskit/
├── cmd/example/          # Usage example with full lifecycle
│   └── main.go
├── internal/
│   ├── domain/           # Core business entities (zero dependencies)
│   │   ├── models.go     # RouterConfig, InterfaceStats, SystemResource, etc.
│   │   └── events.go     # TelemetryEvent, EventUpdate/EventDead
│   ├── usecase/          # Business logic orchestration
│   │   └── telemetry.go  # Routes events to InfluxDB + Redis + Pub/Sub
│   ├── repository/       # Data persistence interfaces & implementations
│   │   ├── interfaces.go # TimeSeriesWriter, CacheRepository, PubSubPublisher
│   │   ├── influxdb.go   # InfluxDB 3 Core with async batching
│   │   └── redis.go      # Redis HSET cache + PUBLISH
│   ├── collector/        # Connection management & streaming engine
│   │   ├── pool.go       # ConnPool: persistent connections + health monitor
│   │   ├── router.go     # RouterCollector: per-router spec execution
│   │   └── engine.go     # Engine: top-level orchestrator
│   └── spec/             # StreamSpec implementations (Command pattern)
│       ├── spec.go       # StreamSpec interface
│       ├── interfaces.go # /interface/print follow (bandwidth)
│       ├── resource.go   # /system/resource/print interval (CPU/memory)
│       ├── hotspot_active.go  # /ip/hotspot/active/print follow
│       └── ppp_active.go      # /ppp/active/print follow (PPPoE)
├── pkg/mikrotik/         # Public utilities
│   ├── parser.go         # MikroTik value parsers (duration, uint64, etc.)
│   └── parser_test.go    # Unit tests (30 cases)
└── docker/               # Infrastructure
    ├── docker-compose.yml
    └── .env.example
```

## Quick Start

### 1. Prerequisites

- Go 1.23+
- Docker & Docker Compose
- A MikroTik router with API access enabled

### 2. Clone & Setup

```bash
git clone https://github.com/quiqxiq/roskit.git
cd roskit

# Copy environment template and edit with your settings
cp docker/.env.example docker/.env
```

Edit `docker/.env` with your MikroTik credentials:

```env
MIKROTIK_ADDRESS=192.168.88.1:8728
MIKROTIK_USERNAME=admin
MIKROTIK_PASSWORD=your-password
```

### 3. Start Infrastructure

```bash
cd docker
docker compose up -d
cd ..
```

This starts:
- **InfluxDB 3 Core** on port `8181` (no auth in dev mode)
- **Redis 7** on port `6380`

### 4. Run the Example

```bash
go run ./cmd/example/
```

You should see output like:

```
level=INFO msg="Redis repository initialized" addr=localhost:6380
level=INFO msg="InfluxDB writer initialized" host=http://localhost:8181 database=roskit
level=INFO msg="router registered in pool" router=core-01 reconnect_interval=5s health_check_interval=30s
level=INFO msg="connection established" router=core-01
level=INFO msg="connection pool status" router=core-01 state=connected
level=INFO msg="health monitor started" router=core-01 interval=30s
level=INFO msg="starting stream" spec=iface-stats
level=INFO msg="starting stream" spec=sys-resource
level=INFO msg="starting stream" spec=hotspot-active
level=INFO msg="starting stream" spec=ppp-active
level=DEBUG msg="flushed points to InfluxDB" count=7
level=INFO msg="cached system resource" cpu_load=4 board_name=RB750G version="6.49.11 (stable)"
```

### 5. Verify Data

**Redis** — cached interface stats and system resource:

```bash
docker exec docker-redis-1 redis-cli KEYS "roskit:*"
# roskit:core-01:interface_stats:ether1
# roskit:core-01:interface_stats:ether2
# roskit:core-01:system_resource:system

docker exec docker-redis-1 redis-cli HGETALL "roskit:core-01:system_resource:system"
# cpu_load       4
# free_memory    7651328
# board_name     RB750G
# version        6.49.11 (stable)
```

**InfluxDB** — time-series data:

```bash
curl -s http://localhost:8181/api/v3/query_sql \
  -H "Content-Type: application/json" \
  -d '{"db":"roskit","q":"SELECT * FROM system_resource ORDER BY time DESC LIMIT 3"}'
```

## Usage as a Library

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/quiqxiq/roskit/internal/collector"
    "github.com/quiqxiq/roskit/internal/domain"
    "github.com/quiqxiq/roskit/internal/repository"
    "github.com/quiqxiq/roskit/internal/spec"
    "github.com/quiqxiq/roskit/internal/usecase"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // 1. Setup repositories
    redis, _ := repository.NewRedisRepository(repository.RedisConfig{
        Addr: "localhost:6380",
    }, logger)
    defer redis.Close()

    influx, _ := repository.NewInfluxDBWriter(repository.InfluxDBConfig{
        Host:     "http://localhost:8181",
        Database: "roskit",
    }, logger)
    defer influx.Close()

    // 2. Create use case (business logic)
    uc := usecase.NewTelemetryUseCase(influx, redis, redis, logger)

    // 3. Create engine with connection pool
    engine := collector.NewEngine(uc, logger)

    // 4. Register routers with per-router settings
    engine.AddRouter(domain.RouterConfig{
        ID:       "core-01",
        Address:  "192.168.88.1:8728",
        Username: "admin",
        Password: "password",

        ReconnectInterval:    5 * time.Second,
        MaxReconnectInterval: 60 * time.Second,
        HealthCheckInterval:  30 * time.Second,
        DialTimeout:          10 * time.Second,
    }, []spec.StreamSpec{
        spec.NewInterfaceStatsSpec(),
        spec.NewSystemResourceSpec(),
        spec.NewHotspotActiveSpec(),
        spec.NewPPPActiveSpec(),
    })

    // 5. Start and wait
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    engine.Start(ctx)

    // Read cached data from Redis (no router hit!)
    data, _ := redis.GetSnapshot(ctx, "roskit:core-01:system_resource:system")
    logger.Info("CPU load", "value", data["cpu_load"])

    // Check connection pool status
    for id, state := range engine.Status() {
        logger.Info("pool", "router", id, "state", state.String())
    }

    // Graceful shutdown
    // cancel()
    // engine.Stop()
}
```

## Adding Custom Specs

Create a new file in `internal/spec/` implementing the `StreamSpec` interface:

```go
package spec

import (
    "github.com/go-routeros/routeros/v3/proto"
    "github.com/quiqxiq/roskit/internal/domain"
)

type WirelessClientsSpec struct{}

func NewWirelessClientsSpec() *WirelessClientsSpec {
    return &WirelessClientsSpec{}
}

func (s *WirelessClientsSpec) Command() []string {
    return []string{
        "/interface/wireless/registration-table/print",
        "=follow",
        "=.proplist=.id,interface,mac-address,signal-strength,tx-rate,rx-rate",
    }
}

func (s *WirelessClientsSpec) Tag() string         { return "wireless-clients" }
func (s *WirelessClientsSpec) Measurement() string  { return "wireless_clients" }

func (s *WirelessClientsSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
    // Parse sentence.Map into a TelemetryEvent...
    return nil, nil
}
```

Then register it:

```go
engine.AddRouter(config, []spec.StreamSpec{
    spec.NewInterfaceStatsSpec(),
    spec.NewWirelessClientsSpec(), // Your new spec
})
```

## Per-Router Connection Config

Each router can have independent reconnect behavior:

| Field | Default | Description |
|-------|---------|-------------|
| `ReconnectInterval` | `5s` | Initial delay before reconnect attempt |
| `MaxReconnectInterval` | `60s` | Maximum backoff cap (exponential) |
| `HealthCheckInterval` | `30s` | How often pool pings the router |
| `DialTimeout` | `10s` | Max time for TCP + auth handshake |

```go
// Core router — standard settings
engine.AddRouter(domain.RouterConfig{
    ID:                   "core-01",
    ReconnectInterval:    5 * time.Second,
    MaxReconnectInterval: 60 * time.Second,
    HealthCheckInterval:  30 * time.Second,
}, specs)

// Edge router — aggressive reconnect (mission-critical)
engine.AddRouter(domain.RouterConfig{
    ID:                   "edge-01",
    ReconnectInterval:    2 * time.Second,
    MaxReconnectInterval: 15 * time.Second,
    HealthCheckInterval:  10 * time.Second,
}, specs)

// Remote AP — relaxed settings (slow WAN link)
engine.AddRouter(domain.RouterConfig{
    ID:                   "remote-ap",
    ReconnectInterval:    10 * time.Second,
    MaxReconnectInterval: 120 * time.Second,
    HealthCheckInterval:  60 * time.Second,
    DialTimeout:          30 * time.Second,
}, specs)
```

## Data Flow

```
MikroTik Router
    │ (RouterOS API binary protocol)
    ▼
ConnPool.RouterConn
    │ persistent TCP, health check via /system/identity/print
    ▼
RouterCollector.AcquireAsync()
    │ ListenArgsQueueContext → <-chan *proto.Sentence
    ▼
StreamSpec.Parse(sentence)
    │ → *domain.TelemetryEvent (EventUpdate or EventDead)
    ▼
TelemetryUseCase.ProcessEvent()
    ├──▶ InfluxDBWriter.WritePoint()      → batched → InfluxDB 3 Core
    ├──▶ RedisRepository.SetSnapshot()    → HSET roskit:{router}:{measurement}:{id}
    └──▶ RedisRepository.Publish()        → PUBLISH roskit:telemetry:{router}
```

## Redis Key Schema

| Pattern | Example | Content |
|---------|---------|---------|
| `roskit:{router}:interface_stats:{name}` | `roskit:core-01:interface_stats:ether1` | rx/tx bytes, packets, drops, errors |
| `roskit:{router}:system_resource:system` | `roskit:core-01:system_resource:system` | CPU, memory, disk, uptime, version |
| `roskit:{router}:hotspot_active:{.id}` | `roskit:core-01:hotspot_active:*1` | user, address, MAC, bytes in/out |
| `roskit:{router}:ppp_active:{.id}` | `roskit:core-01:ppp_active:*2` | name, service, caller-id, address |

**Pub/Sub channel:** `roskit:telemetry:{router}` — JSON events for real-time subscribers.

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| [`go-routeros/routeros/v3`](https://github.com/go-routeros/routeros) | v3.0.1 | MikroTik API client |
| [`influxdb3-go/v2`](https://github.com/InfluxCommunity/influxdb3-go) | v2.13.0 | InfluxDB 3 Core client |
| [`go-redis/v9`](https://github.com/redis/go-redis) | v9.18.0 | Redis client |
| [`godotenv`](https://github.com/joho/godotenv) | v1.5.1 | `.env` file loading |

## License

MIT
