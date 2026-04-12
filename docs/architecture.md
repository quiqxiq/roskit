# Arsitektur RosKit

## Daftar Isi

- [Gambaran Umum](#gambaran-umum)
- [Clean Architecture](#clean-architecture)
- [Alur Data](#alur-data)
- [Connection Pool](#connection-pool)
- [StreamSpec Pattern](#streamspec-pattern)
- [Storage Layer](#storage-layer)
- [Dead Entity Handling](#dead-entity-handling)

---

## Gambaran Umum

RosKit menggunakan **Clean Architecture** yang memisahkan kode menjadi lapisan-lapisan independen:

```
┌──────────────────────────────────────────────────┐
│                  cmd/example/                     │
│              (Aplikasi / Entry Point)             │
├──────────────────────────────────────────────────┤
│               internal/collector/                 │
│    Engine → ConnPool → RouterCollector             │
│         (Infrastruktur: Koneksi Router)           │
├──────────────────────────────────────────────────┤
│                internal/usecase/                  │
│             TelemetryUseCase                      │
│         (Logika Bisnis: Routing Event)            │
├──────────────────────────────────────────────────┤
│              internal/repository/                 │
│     InfluxDB Writer │ Redis Cache │ Pub/Sub       │
│         (Infrastruktur: Penyimpanan)              │
├──────────────────────────────────────────────────┤
│                internal/domain/                   │
│    RouterConfig │ InterfaceStats │ LogEntry │ ...  │
│         (Entity: Tanpa Dependency)                │
├──────────────────────────────────────────────────┤
│                 internal/spec/                    │
│   InterfaceStatsSpec │ LogSpec │ DHCPLeaseSpec     │
│         (Command Pattern: Parser)                 │
└──────────────────────────────────────────────────┘
```

> **Aturan Dependency:** Lapisan dalam tidak boleh mengimpor lapisan luar. `domain` tidak tahu tentang Redis atau InfluxDB.

---

## Clean Architecture

### Layer 1: Domain (`internal/domain/`)

Entity murni tanpa dependency eksternal:

| File | Entity | Deskripsi |
|------|--------|-----------|
| `config.go` | `RouterConfig` | Konfigurasi koneksi + reconnect per router |
| `events.go` | `TelemetryEvent` | Event container (update/dead) |
| `interface_stats.go` | `InterfaceStats` | Statistik traffic interface |
| `system_resource.go` | `SystemResource` | CPU, memory, disk, uptime |
| `queue_simple.go` | `QueueSimpleStats` | Bandwidth per queue rule |
| `hotspot_active.go` | `HotspotActiveUser` | User hotspot aktif |
| `ppp_active.go` | `PPPActiveSession` | Sesi PPPoE/L2TP aktif |
| `ppp_config.go` | `PPPProfile`, `PPPSecret` | Konfigurasi PPP |
| `log_entry.go` | `LogEntry` | Log router |
| `dhcp_lease.go` | `DHCPLease` | DHCP lease |

Setiap entity menyediakan:
- `ToTags()` → InfluxDB tag set
- `ToFields()` → InfluxDB field set
- `ToCacheData()` → Redis HSET data

### Layer 2: UseCase (`internal/usecase/`)

`TelemetryUseCase` menerima `TelemetryEvent` dan merutekannya:

```
ProcessEvent(event)
  ├── EventUpdate → WritePoint + SetSnapshot + Publish
  └── EventDead   → DeleteSnapshot + Publish
```

### Layer 3: Repository (`internal/repository/`)

Interface abstrak untuk storage:

```go
type TimeSeriesWriter interface {
    WritePoint(ctx, measurement, tags, fields, timestamp) error
    Flush(ctx) error
    Close() error
}

type CacheRepository interface {
    SetSnapshot(ctx, key, data) error
    GetSnapshot(ctx, key) (map[string]string, error)
    DeleteSnapshot(ctx, key) error
    Close() error
}

type PubSubPublisher interface {
    Publish(ctx, channel, message) error
    Close() error
}
```

Implementasi:
- `influxdb.go` → InfluxDB 3 Core (async batching)
- `redis.go` → Redis 7 (HSET + PUBLISH)
- `noop.go` → No-Op (disable storage)

### Layer 4: Collector (`internal/collector/`)

```
Engine
  ├── ConnPool (owns persistent connections)
  │     └── RouterConn per router
  │           ├── Connect / Reconnect
  │           ├── AcquireAsync / Release
  │           └── IsAlive (health check)
  └── RouterCollector per router
        ├── Borrows connection from pool
        ├── Runs all specs concurrently
        └── Returns connection on error/done
```

### Layer 5: Spec (`internal/spec/`)

Command pattern — satu file per data source:

```go
type StreamSpec interface {
    Command() []string
    Tag() string
    Measurement() string
    Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error)
}
```

---

## Alur Data

```
MikroTik Router
    │
    │ RouterOS API (biner via TCP)
    │ Mode: Async() + ListenArgsQueueContext()
    ▼
┌───────────────────────────┐
│     ConnPool.RouterConn   │  Persistent TCP connection
│     Health: /system/identity/print every N seconds
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│     RouterCollector       │  AcquireAsync() → runs specs
│     9 specs concurrent    │  goroutine per spec
└───────────┬───────────────┘
            │ *proto.Sentence (raw API response)
            ▼
┌───────────────────────────┐
│     StreamSpec.Parse()    │  Converts to TelemetryEvent
│     Handles .dead=yes     │  EventUpdate or EventDead
└───────────┬───────────────┘
            │ *domain.TelemetryEvent
            ▼
┌───────────────────────────┐
│  TelemetryUseCase         │
│  ProcessEvent()           │
│                           │
│  ┌──────────────────────┐ │
│  │ ① tsWriter           │ │ → InfluxDB (batched time-series)
│  │ ② cache              │ │ → Redis HSET (latest snapshot)
│  │ ③ publisher          │ │ → Redis PUBLISH (realtime event)
│  └──────────────────────┘ │
└───────────────────────────┘
```

---

## Connection Pool

### Desain

Setiap router memiliki **1 koneksi TCP persisten** yang dikelola oleh `ConnPool`:

```
ConnPool
  ├── RouterConn["core-01"]
  │     state: connected
  │     health: ping every 30s
  │     dial_timeout: 10s
  │     reconnect: 5s → 10s → 20s → 40s → 60s (max)
  │
  ├── RouterConn["edge-01"]
  │     state: connected
  │     health: ping every 15s
  │     reconnect: 2s → 4s → 8s → 15s (max)
  │
  └── RouterConn["remote-ap"]
        state: disconnected
        health: ping every 60s
        reconnect: 10s → 20s → 40s → 80s → 120s (max)
```

### Lifecycle

```
Register → Start → Connect → [Health Loop]
                        ↓
                  AcquireAsync (collector borrows)
                        ↓
                  Streaming specs...
                        ↓
                  Release (collector returns)
                        ↓
                  [Pool reconnects] → Connect → ...
```

### Health Check

Pool mengirim `/system/identity/print` ke setiap router pada interval yang dikonfigurasi. Jika gagal, koneksi di-close dan di-reconnect pada cycle berikutnya.

---

## StreamSpec Pattern

### 2 Mode Streaming

| Mode | Command | Behavior |
|------|---------|----------|
| **Follow** | `=follow` | Event-driven: data dikirim hanya saat berubah |
| **Stats/Interval** | `=stats =interval=1s` | Periodic: data dikirim setiap N detik |

### Spec yang Tersedia

| Spec | Command | Mode | Data |
|------|---------|------|------|
| `InterfaceStatsSpec` | `/interface/print` | follow + interval | rx/tx bytes, packets, drops |
| `SystemResourceSpec` | `/system/resource/print` | interval=5s | CPU, memory, disk |
| `QueueSimpleStatsSpec` | `/queue/simple/print` | stats + interval=1s | rate, bytes, dropped per queue |
| `HotspotActiveSpec` | `/ip/hotspot/active/print` | follow | user sessions |
| `PPPActiveSpec` | `/ppp/active/print` | follow | PPPoE/L2TP sessions |
| `PPPProfileSpec` | `/ppp/profile/print` | follow | profile config changes |
| `PPPSecretSpec` | `/ppp/secret/print` | follow | user account changes |
| `LogSpec` | `/log/print` | follow | router log entries |
| `DHCPLeaseSpec` | `/ip/dhcp-server/lease/print` | follow | DHCP lease changes |

---

## Storage Layer

### InfluxDB 3 Core (Time-Series)

- **Tujuan:** Penyimpanan historis untuk grafik, trend, analisis
- **Write Mode:** Async batching (5000 points / 1s flush)
- **Format:** Line Protocol (`measurement,tags fields timestamp`)
- **Query:** SQL via REST API

### Redis (Cache + Pub/Sub)

- **HSET Cache:** Snapshot terkini per entity — instant read tanpa query router
- **Pub/Sub:** Broadcast JSON event ke semua subscriber
- **TTL:** Tidak ada TTL default — data bertahan sampai `.dead=yes` atau FLUSHALL

### Kapan Pakai Apa?

| Kebutuhan | Backend |
|-----------|---------|
| Grafik historis (Grafana) | InfluxDB |
| Dashboard realtime | Redis Cache |
| Alert/Notification | Redis Pub/Sub |
| AI Agent query | Redis Cache |
| Capacity planning | InfluxDB |
| Instant status check | Redis Cache |

---

## Dead Entity Handling

Ketika entity dihapus dari router (user disconnect, lease expire, interface remove), RouterOS mengirim atribut `.dead=yes`.

```
Router → .dead=yes → Parse() → EventDead
    ↓
TelemetryUseCase
    ├── cache.DeleteSnapshot()  → hapus dari Redis
    └── publisher.Publish()     → broadcast dead event
```

**Tidak ditulis ke InfluxDB** — dead event hanya menghapus cache dan notifikasi subscriber.
