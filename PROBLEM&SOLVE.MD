# Architecture Improvement Plan

## Context

Ditemukan 4 masalah arsitektur setelah analisis codebase, go-routeros/v3 library, dan RouterOS 7.20.8 command spec. Target: setiap router hanya butuh **2 koneksi TCP** (turun dari 19–20), data InfluxDB hanya berisi **telemetry** (bukan config), cache di-invalidate eksplisit setelah mutation.

---

## Fix 1 — Storage Routing: Hanya Telemetry ke InfluxDB

### Problem
`event.Processor` menulis SEMUA measurement ke InfluxDB + Redis + pub-sub tanpa filter. Dari 19 stream measurement, hanya 3 yang benar-benar time-series.

### Yang Harus Masuk InfluxDB (telemetry)
| Measurement | Type | Alasan |
|---|---|---|
| `system_resource` | Poll | CPU/mem over time |
| `interface_traffic` | Stream | Bandwidth tx/rx per menit |
| `hotspot_active` | Stream | Monitoring puncak sesi aktif |

### Yang Hanya ke Redis (config/state)
`hotspot_user`, `hotspot_profile`, `hotspot_server`, `hotspot_host`, `hotspot_cookie`, `ip_binding`, `walled_garden`, `system_scheduler`, `system_script`, `interface`, `dhcp_lease`, `arp`, `ip_pool`, `firewall_nat`, `queue_simple`, `ppp_secret`, `ppp_active`, `system_identity`, `system_clock`, `system_routerboard`

### Implementasi

**File: `internal/roskit/core/command/meta.go`**
Tambah field `WriteTimeSeries bool` ke struct `CommandMeta`:
```go
type CommandMeta struct {
    // ... existing fields ...
    WriteTimeSeries bool // true only for telemetry measurements
}
```

**File: `internal/roskit/core/definition/system.go` dan `hotspot.go`**
Set `WriteTimeSeries: true` hanya untuk:
```go
// system_resource
{Measurement: "system_resource", WriteTimeSeries: true, ...}
// interface_traffic
{Measurement: "interface_traffic", WriteTimeSeries: true, ...}
// hotspot_active
{Measurement: "hotspot_active", WriteTimeSeries: true, ...}
```
Semua definisi lain: `WriteTimeSeries: false` (default, tidak perlu ditulis eksplisit).

**File: `internal/roskit/pipeline/event/processor.go`**
Tambah guard sebelum `WritePoint` di `OnEvent()` dan `OnPoll()`:
```go
// BEFORE (menulis ke InfluxDB tanpa kondisi)
p.timeseries.WritePoint(ctx, point)

// AFTER
if meta.WriteTimeSeries {
    p.timeseries.WritePoint(ctx, point)
}
```

---

## Fix 2 — Cache Invalidation Setelah Mutation

### Problem
`mutation/handler.go` eksekusi RouterOS command (add/set/remove/enable/disable) tapi TIDAK invalidate Redis cache. Cache stale sampai stream event masuk (~1–5 detik delay).

### Implementasi

**File: `internal/roskit/behavior/mutation/handler.go`**

Handler sudah menerima `routerID` dan `meta.Measurement`. Cache interface sudah punya `DeleteSnapshot()` dan `DeleteIndex()`. Tambah cache ke handler dan panggil setelah successful mutation:

```go
type Handler struct {
    pool   *execution.Pool
    query  *query.Handler
    cache  cache.Repository // ADD
    logger *slog.Logger
}

func (h *Handler) Execute(ctx context.Context, routerID string, meta *command.CommandMeta, args []string) (*routeros.Reply, error) {
    reply, err := h.executor.Run(ctx, routerID, meta, args)
    if err != nil {
        return nil, err
    }
    // Invalidate cache for this measurement after successful write
    if h.cache != nil && meta.Measurement != "" {
        snapshots, _ := h.cache.ScanByMeasurement(ctx, routerID, meta.Measurement)
        for _, key := range snapshots {
            h.cache.DeleteSnapshot(ctx, key)
        }
        h.cache.DeleteIndex(ctx, cache.FormatIndexKey(routerID, meta.Measurement), "")
    }
    return reply, nil
}
```

Constructor `mutation.NewHandler` perlu menerima `cache.Repository` tambahan. Update di `orchestrator/engine.go` saat konstruksi `mutateH`.

---

## Fix 3 — Shared Async Connection: 19 Koneksi → 2 Per Router

### Problem
Setiap stream worker memanggil `pool.DialSync()` → masing-masing dapat **koneksi TCP terpisah**. 19 stream workers = 19 koneksi. RouterOS limit 20, sudah menyentuh batas dengan 1 router.

### Solusi
go-routeros/v3 mendukung **tag multiplexing** via `client.Async()` + `client.ListenArgsQueueContext()`. SEMUA stream worker bisa berbagi **1 async connection**:
- `client.Async()` → enable async mode, mulai goroutine internal untuk dispatch response by tag
- `client.ListenArgsQueueContext(ctx, sentence, queueSize)` → returns `*ListenReply` dengan channel sendiri per stream
- Library secara otomatis routing response `.tag=N` ke channel yang benar

**Hasil: 1 async conn (stream) + 1 sync conn (poll/query/mutation) = 2 koneksi per router**

### Implementasi

**File: `internal/roskit/execution/pool.go`**

Tambah `asyncConn *routeros.Client` di `RouterConn` struct:
```go
type RouterConn struct {
    mu      sync.RWMutex
    cmdMu   sync.Mutex
    cfg     ConnConfig
    client  *routeros.Client          // sync conn: poll/query/mutation
    async   *routeros.Client          // async conn: semua stream workers
    state   ConnState
    logger  *slog.Logger
}
```

Tambah method `BorrowAsync()` — dial async connection on first call, reuse on subsequent:
```go
func (rc *RouterConn) BorrowAsync(ctx context.Context) (*routeros.Client, error) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    if rc.async != nil {
        return rc.async, nil
    }
    client, err := routeros.DialContext(ctx, rc.cfg.Address, rc.cfg.Username, rc.cfg.Password)
    if err != nil {
        return nil, fmt.Errorf("async dial %s: %w", rc.cfg.Address, err)
    }
    client.Async() // enable tag multiplexing
    rc.async = client
    return client, nil
}

func (rc *RouterConn) Close() {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    if rc.async != nil {
        rc.async.Close()
        rc.async = nil
    }
    if rc.client != nil {
        rc.client.Close()
        rc.client = nil
    }
    rc.state = ConnStateDisconnected
}
```

Tambah method di `Pool`:
```go
func (p *Pool) BorrowAsync(ctx context.Context, routerID string) (*routeros.Client, error) {
    p.mu.RLock()
    conn, ok := p.conns[routerID]
    p.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("pool: router %q not registered", routerID)
    }
    return conn.BorrowAsync(ctx)
}
```

**File: `internal/roskit/behavior/stream/worker.go`**

Ganti `pool.DialSync()` dengan `pool.BorrowAsync()`. Hapus `client.Async()` call (sudah dilakukan di `BorrowAsync`). `ListenArgsQueueContext` tetap sama — library sudah handle multiplexing.

```go
// BEFORE
client, err := w.pool.DialSync(ctx, routerID)
// ...
client.Async()
reply, err := client.ListenArgsQueueContext(ctx, sentence, defaultQueueSize)

// AFTER
client, err := w.pool.BorrowAsync(ctx, routerID)
// Async() sudah dipanggil di BorrowAsync
reply, err := client.ListenArgsQueueContext(ctx, sentence, defaultQueueSize)
```

**Catatan penting**: Stream workers TIDAK close connection setelah selesai — connection dibagikan. Hanya `pool.Unregister()` / `conn.Close()` yang close async connection.

---

## Fix 4 — Pub-Sub: Publish Hanya untuk Telemetry & Active Sessions

### Problem
Pub-sub dipublish untuk SEMUA measurement tapi tidak ada subscriber yang jelas di codebase. Config data (hotspot_user, dll) tidak perlu di-broadcast realtime.

### Implementasi

**File: `internal/roskit/pipeline/event/processor.go`**

Tambah kondisi sebelum publish. Definisikan measurement yang perlu realtime broadcast:

```go
// Pub-sub hanya untuk measurement yang berguna untuk realtime frontend
var realtimeMeasurements = map[string]bool{
    "hotspot_active":    true,
    "interface_traffic": true,
    "system_resource":   true,
    "dhcp_lease":        true, // useful untuk monitoring
}

// Di OnEvent() dan OnPoll():
if realtimeMeasurements[meta.Measurement] {
    p.pubsub.Publish(ctx, channel, msg)
}
```

---

## Urutan Implementasi

```
Fix 1: WriteTimeSeries flag      (meta.go → definition/*.go → processor.go)
Fix 2: Cache invalidation        (mutation/handler.go + engine.go wiring)
Fix 3: Shared async connection   (pool.go BorrowAsync → stream/worker.go)
Fix 4: Pub-sub filter            (processor.go realtimeMeasurements)
```

Fix 1 dan 2 bisa dilakukan paralel. Fix 3 bergantung pada pool.go selesai. Fix 4 bisa dilakukan kapan saja.

---

## File Kritis

| File | Fix |
|---|---|
| `internal/roskit/core/command/meta.go` | Fix 1: tambah `WriteTimeSeries bool` |
| `internal/roskit/core/definition/system.go` | Fix 1: set flag true untuk system_resource |
| `internal/roskit/core/definition/hotspot.go` | Fix 1: set flag true untuk interface_traffic, hotspot_active |
| `internal/roskit/pipeline/event/processor.go` | Fix 1 + Fix 4: guard InfluxDB + pub-sub |
| `internal/roskit/behavior/mutation/handler.go` | Fix 2: invalidate cache setelah write |
| `internal/roskit/orchestrator/engine.go` | Fix 2: pass cache ke mutation.NewHandler |
| `internal/roskit/execution/pool.go` | Fix 3: `asyncConn` field + `BorrowAsync()` |
| `internal/roskit/behavior/stream/worker.go` | Fix 3: ganti DialSync → BorrowAsync |

---

## Verifikasi

### Fix 1 — InfluxDB hanya telemetry
```bash
# Setelah implementasi, restart docker dan tunggu 2 menit
docker exec docker-influxdb-1 influxdb3 query --token mytoken --database mikhmon \
  "SELECT count(*) FROM hotspot_user"
# Harus: 0 rows (tidak ada lagi data config di InfluxDB)

docker exec docker-influxdb-1 influxdb3 query --token mytoken --database mikhmon \
  "SELECT count(*) FROM system_resource"
# Harus: terus bertambah (telemetry tetap masuk)
```

### Fix 2 — Cache invalidation
```bash
# Tambah user via API, cek Redis langsung setelah
docker exec docker-redis-1 redis-cli hgetall "roskit:1:idx:hotspot_user"
# User baru harus hilang dari cache (invalidated), bukan stale sampai stream event
```

### Fix 3 — Hanya 2 koneksi per router
```bash
# Cek jumlah koneksi TCP dari container ke MikroTik
docker exec docker-api-1 ss -tn | grep 8728 | wc -l
# Harus: 2 (bukan 20)
```

### Fix 4 — Pub-sub hanya measurement relevan
```bash
# Subscribe ke Redis pub-sub dan cek hanya measurement tertentu yang publish
docker exec docker-redis-1 redis-cli psubscribe "roskit:1:*"
# Hanya interface_traffic, hotspot_active, system_resource, dhcp_lease yang muncul
```

### Full test suite
```bash
TEST_PASSWORD=admin1234 ROUTER_ID=1 ROUTER_IP=192.168.233.1 ROUTER_PASSWORD=r00t \
  python -m pytest tests/http/ -v --tb=short -q
# Target: 127/127 passed
```
