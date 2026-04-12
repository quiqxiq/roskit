# Panduan Penggunaan RosKit

## Daftar Isi

- [Instalasi](#instalasi)
- [Konfigurasi](#konfigurasi)
- [Menjalankan](#menjalankan)
- [Memilih Storage Backend](#memilih-storage-backend)
- [Menambah Router](#menambah-router)
- [Menambah Spec Baru](#menambah-spec-baru)
- [Membaca Data dari Redis](#membaca-data-dari-redis)
- [Query Data dari InfluxDB](#query-data-dari-influxdb)
- [Subscribe Event Real-Time](#subscribe-event-real-time)

---

## Instalasi

### Prasyarat

- Go 1.23+
- Docker & Docker Compose
- MikroTik router dengan API access diaktifkan

### Clone & Setup

```bash
git clone https://github.com/quiqxiq/roskit.git
cd roskit
cp docker/.env.example docker/.env
```

### Install Dependencies

```bash
go mod download
```

### Start Infrastruktur

```bash
cd docker
docker compose up -d
cd ..
```

Ini akan menjalankan:
- **InfluxDB 3 Core** pada port `8181`
- **Redis 7** pada port `6380`

---

## Konfigurasi

### Environment Variables

Edit `docker/.env`:

```env
# InfluxDB
INFLUXDB3_DATABASE=roskit

# Redis
REDIS_PORT=6380

# MikroTik
MIKROTIK_ADDRESS=192.168.88.1:8728
MIKROTIK_USERNAME=admin
MIKROTIK_PASSWORD=your-password
```

### RouterConfig

Setiap router dikonfigurasi melalui `domain.RouterConfig`:

```go
domain.RouterConfig{
    ID:       "core-01",              // Identifier unik
    Address:  "192.168.88.1:8728",    // IP:Port API
    Username: "admin",
    Password: "password",
    UseTLS:   false,                  // true untuk port 8729

    // Tuning koneksi per-router
    ReconnectInterval:    5 * time.Second,   // Delay awal reconnect
    MaxReconnectInterval: 60 * time.Second,  // Batas maksimum backoff
    HealthCheckInterval:  30 * time.Second,  // Interval ping health check
    DialTimeout:          10 * time.Second,  // Timeout koneksi TCP
}
```

| Field | Default | Deskripsi |
|-------|---------|-----------|
| `ReconnectInterval` | 5s | Delay awal sebelum reconnect (exponential backoff) |
| `MaxReconnectInterval` | 60s | Batas maksimum delay reconnect |
| `HealthCheckInterval` | 30s | Seberapa sering pool mengirim ping `/system/identity/print` |
| `DialTimeout` | 10s | Timeout TCP + autentikasi |

---

## Menjalankan

### Sebagai Contoh

```bash
go run ./cmd/example/
```

### Sebagai Library di Proyek Anda

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
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

    // 2. Business logic layer
    uc := usecase.NewTelemetryUseCase(influx, redis, redis, logger)

    // 3. Engine dengan connection pool
    engine := collector.NewEngine(uc, logger)

    // 4. Register router + specs
    engine.AddRouter(domain.RouterConfig{
        ID:       "core-01",
        Address:  "192.168.88.1:8728",
        Username: "admin",
        Password: "password",
        ReconnectInterval:   5 * time.Second,
        HealthCheckInterval: 30 * time.Second,
    }, []spec.StreamSpec{
        spec.NewInterfaceStatsSpec(),
        spec.NewSystemResourceSpec(),
        spec.NewQueueSimpleStatsSpec(),
        spec.NewLogSpec(),
    })

    // 5. Start
    ctx, cancel := context.WithCancel(context.Background())
    engine.Start(ctx)

    // 6. Graceful shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    cancel()
    engine.Stop()
}
```

---

## Memilih Storage Backend

RosKit menggunakan 3 repository interface yang **independen**. Anda bisa mengaktifkan/menonaktifkan masing-masing:

### Keduanya (InfluxDB + Redis)

```go
uc := usecase.NewTelemetryUseCase(influxWriter, redisRepo, redisRepo, logger)
```

### Hanya Redis (tanpa InfluxDB)

```go
uc := usecase.NewTelemetryUseCase(
    repository.NoOpTimeSeriesWriter{},  // skip InfluxDB
    redisRepo,                          // cache aktif
    redisRepo,                          // pub/sub aktif
    logger,
)
```

### Hanya InfluxDB (tanpa Redis)

```go
uc := usecase.NewTelemetryUseCase(
    influxWriter,                        // time-series aktif
    repository.NoOpCacheRepository{},    // skip Redis cache
    repository.NoOpPubSubPublisher{},    // skip Redis pub/sub
    logger,
)
```

### Hanya Redis Cache (tanpa Pub/Sub & InfluxDB)

```go
uc := usecase.NewTelemetryUseCase(
    repository.NoOpTimeSeriesWriter{},  // skip InfluxDB
    redisRepo,                          // cache aktif
    repository.NoOpPubSubPublisher{},   // skip pub/sub
    logger,
)
```

> **Note:** `NoOp*` implementations ada di `internal/repository/noop.go`.

---

## Menambah Router

Router bisa ditambahkan secara dinamis, bahkan saat engine sudah berjalan:

```go
// Router utama — setting standar
engine.AddRouter(domain.RouterConfig{
    ID:                   "core-01",
    Address:              "192.168.88.1:8728",
    ReconnectInterval:    5 * time.Second,
    HealthCheckInterval:  30 * time.Second,
}, allSpecs)

// Router edge — aggressive reconnect (link kritis)
engine.AddRouter(domain.RouterConfig{
    ID:                   "edge-01",
    Address:              "10.0.0.1:8729",
    UseTLS:               true,
    ReconnectInterval:    2 * time.Second,
    MaxReconnectInterval: 15 * time.Second,
    HealthCheckInterval:  10 * time.Second,
}, allSpecs)

// AP remote — setting relax (koneksi WAN lambat)
engine.AddRouter(domain.RouterConfig{
    ID:                   "remote-ap",
    Address:              "vpn.example.com:8728",
    ReconnectInterval:    10 * time.Second,
    MaxReconnectInterval: 120 * time.Second,
    HealthCheckInterval:  60 * time.Second,
    DialTimeout:          30 * time.Second,
}, allSpecs)

// Hapus router
engine.RemoveRouter("remote-ap")

// Cek status koneksi pool
for id, state := range engine.Status() {
    fmt.Printf("Router %s: %s\n", id, state.String())
}
```

---

## Menambah Spec Baru

Buat file baru di `internal/spec/` yang mengimplementasikan interface `StreamSpec`:

```go
// internal/spec/wireless_clients.go
package spec

import (
    "time"
    "github.com/go-routeros/routeros/v3/proto"
    "github.com/quiqxiq/roskit/internal/domain"
    "github.com/quiqxiq/roskit/pkg/mikrotik"
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
    pairs := sentence.Map
    if len(pairs) == 0 { return nil, nil }

    id := pairs[".id"]
    if id == "" { return nil, nil }

    now := time.Now()

    if mikrotik.IsDead(pairs) {
        return &domain.TelemetryEvent{
            RouterID: routerID, Measurement: s.Measurement(),
            Type: domain.EventDead, Timestamp: now,
            CacheKey: mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
        }, nil
    }

    return &domain.TelemetryEvent{
        RouterID:    routerID,
        Measurement: s.Measurement(),
        Tags:        map[string]string{"interface": pairs["interface"], "mac_address": pairs["mac-address"]},
        Fields:      map[string]interface{}{"signal_strength": pairs["signal-strength"], "tx_rate": pairs["tx-rate"], "rx_rate": pairs["rx-rate"]},
        Timestamp:   now,
        Type:        domain.EventUpdate,
        CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
        CacheData:   map[string]string{"mac_address": pairs["mac-address"], "signal": pairs["signal-strength"]},
    }, nil
}
```

Kemudian daftarkan:

```go
engine.AddRouter(config, []spec.StreamSpec{
    spec.NewWirelessClientsSpec(),  // spec baru
    spec.NewInterfaceStatsSpec(),   // spec existing
})
```

> **Tips:** Anda tidak perlu mengubah file apapun selain membuat file spec baru dan mendaftarkannya.

---

## Membaca Data dari Redis

### Menggunakan `redis-cli`

```bash
# Lihat semua key
redis-cli -p 6380 KEYS "roskit:*"

# Baca system resource
redis-cli -p 6380 HGETALL "roskit:core-01:system_resource:system"

# Baca interface stats
redis-cli -p 6380 HGETALL "roskit:core-01:interface_stats:ether1"

# Baca queue stats
redis-cli -p 6380 HGETALL "roskit:core-01:queue_simple_stats:TRAFIK"
```

### Dari Kode Go

```go
data, err := redis.GetSnapshot(ctx, "roskit:core-01:system_resource:system")
if err == nil {
    fmt.Println("CPU:", data["cpu_load"])
    fmt.Println("Memory:", data["free_memory"])
    fmt.Println("Board:", data["board_name"])
}
```

### Key Schema

| Pattern | Contoh |
|---------|--------|
| `roskit:{router}:interface_stats:{name}` | `roskit:core-01:interface_stats:ether1` |
| `roskit:{router}:system_resource:system` | `roskit:core-01:system_resource:system` |
| `roskit:{router}:queue_simple_stats:{name}` | `roskit:core-01:queue_simple_stats:TRAFIK` |
| `roskit:{router}:dhcp_lease:{.id}` | `roskit:core-01:dhcp_lease:*1A` |
| `roskit:{router}:ppp_active:{.id}` | `roskit:core-01:ppp_active:*2` |
| `roskit:{router}:ppp_profile:{.id}` | `roskit:core-01:ppp_profile:*1` |
| `roskit:{router}:ppp_secret:{.id}` | `roskit:core-01:ppp_secret:*3` |
| `roskit:{router}:hotspot_active:{.id}` | `roskit:core-01:hotspot_active:*4` |
| `roskit:{router}:router_log:{.id}` | `roskit:core-01:router_log:*4FF` |

---

## Query Data dari InfluxDB

### REST API (SQL)

```bash
curl -s http://localhost:8181/api/v3/query_sql \
  -H "Content-Type: application/json" \
  -d '{"db":"roskit","q":"SELECT * FROM queue_simple_stats ORDER BY time DESC LIMIT 5"}'
```

### Contoh Query

```sql
-- Semua tabel yang tersedia
SHOW TABLES;

-- CPU load terbaru
SELECT cpu_load, free_memory, uptime FROM system_resource
ORDER BY time DESC LIMIT 1;

-- Traffic interface per 5 menit
SELECT name, AVG(rx_byte) as avg_rx, AVG(tx_byte) as avg_tx
FROM interface_stats
WHERE time > now() - INTERVAL '1 hour'
GROUP BY name, DATE_BIN(INTERVAL '5 minutes', time);

-- Queue stats terbaru
SELECT name, target, rate, bytes, dropped
FROM queue_simple_stats
ORDER BY time DESC LIMIT 10;

-- Log entries terakhir
SELECT log_time, topics, message FROM router_log
ORDER BY time DESC LIMIT 20;

-- DHCP leases aktif
SELECT address, mac_address, server FROM dhcp_lease
ORDER BY time DESC LIMIT 10;
```

### Measurement List

| Measurement | Deskripsi |
|-------------|-----------|
| `interface_stats` | Bandwidth per interface (rx/tx bytes, packets, drops) |
| `system_resource` | CPU, memory, disk, uptime, version |
| `queue_simple_stats` | Rate, bytes, dropped per queue rule |
| `router_log` | Log entries (time, topics, message) |
| `dhcp_lease` | DHCP lease (address, MAC, server) |
| `ppp_profile` | PPP profile configuration |
| `ppp_secret` | PPP user accounts |
| `ppp_active` | Active PPPoE/L2TP sessions |
| `hotspot_active` | Active hotspot users |

---

## Subscribe Event Real-Time

### Menggunakan `redis-cli`

```bash
redis-cli -p 6380 SUBSCRIBE "roskit:telemetry:core-01"
```

Setiap event dikirim sebagai JSON:

```json
{
  "router_id": "core-01",
  "measurement": "interface_stats",
  "type": "update",
  "tags": {"name": "ether1", "type": "ether"},
  "fields": {"rx_byte": 1024000, "tx_byte": 512000},
  "timestamp": "2026-04-13T06:00:00+07:00"
}
```

### Dari Kode Go

```go
pubsub := redisClient.Subscribe(ctx, "roskit:telemetry:core-01")
ch := pubsub.Channel()

for msg := range ch {
    var event map[string]interface{}
    json.Unmarshal([]byte(msg.Payload), &event)
    fmt.Printf("[%s] %s: %v\n",
        event["measurement"],
        event["type"],
        event["fields"],
    )
}
```

### Pub/Sub Channel Format

| Channel | Deskripsi |
|---------|-----------|
| `roskit:telemetry:{router_id}` | Semua event dari router tertentu |

---

## Troubleshooting

### Koneksi gagal ke MikroTik

```
connect core-01: dial tcp 192.168.88.1:8728: connect: connection refused
```

**Solusi:** Pastikan API service aktif di router:
```
/ip/service/enable api
```

### InfluxDB reserved column 'time'

```
'time' is a reserved column
```

**Solusi:** Jangan gunakan `"time"` sebagai field name di InfluxDB. RosKit sudah menggunakan `"log_time"` untuk log entries.

### Health check failed

```
health check failed: read tcp: connection reset by peer
```

**Penyebab:** Router me-reset koneksi (reboot, overload, firewall). Pool akan otomatis reconnect sesuai `ReconnectInterval`.
