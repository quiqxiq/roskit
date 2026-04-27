# Gap Analysis: Mikhmon (PHP) vs Roskit (Go)

> Perbandingan command RouterOS dan cara kerja antara **Mikhmon v3** (PHP monolith)
> dan **Roskit** (Go backend). Fokus pada: command yang dipakai, yang hilang, yang baru,
> dan perbedaan arsitektur eksekusinya.

---

## 1. Cara Kerja — Perbandingan Fundamental

| Aspek | Mikhmon (PHP) | Roskit (Go) |
|-------|--------------|-------------|
| **Koneksi ke RouterOS** | Buka TCP 8728 per-request, tutup setelah selesai | Persistent pool, 1 koneksi per router, hidup seumur proses |
| **Model baca data** | `$API->comm("/path/print")` langsung ke RouterOS tiap halaman dibuka | Stream (`=follow`) + Poll ticker → data masuk Redis, handler baca dari cache |
| **Caching** | Tidak ada. Setiap render halaman = 1+ API call ke router | Redis HSET dengan TTL. Baca dari cache, fallback ke router |
| **Realtime push** | Polling AJAX dari browser (setInterval setiap 8–65 detik) | Redis Pub/Sub → SSE ke frontend (push, bukan poll) |
| **Penyimpanan data jual** | `/system/script` di RouterOS digunakan sebagai "database" | PostgreSQL (`voucher_sales` table) |
| **Koneksi bersamaan** | 1 user = 1 koneksi PHP = 1 koneksi RouterOS | N user share 1 pool koneksi per router |
| **Fault tolerance** | Koneksi gagal = halaman error | Auto-reconnect dengan exponential backoff |

---

## 2. Command RouterOS — Mikhmon (46 path unik)

Diekstrak dari 180 pemanggilan `$API->comm()` di seluruh file `.php`.

### 2.1 READ (print)

```
/interface/monitor-traffic          # traffic chart (polling AJAX tiap 8 detik)
/interface/print                    # daftar interface
/ip/arp/print                       # ARP table
/ip/dhcp-server/lease/print         # DHCP leases
/ip/hotspot/active/print            # sesi hotspot aktif
/ip/hotspot/cookie/print            # hotspot cookies
/ip/hotspot/host/print              # hotspot hosts
/ip/hotspot/ip-binding/print        # IP binding
/ip/hotspot/print                   # daftar hotspot server
/ip/hotspot/server/print            # server list (legacy/alias path)
/ip/hotspot/user/print              # daftar user hotspot
/ip/hotspot/user/profile/print      # daftar user profile
/ip/pool/print                      # IP pool
/log/print                          # system log
/ppp/active/print                   # PPP active sessions (via index.php)
/ppp/profile/print                  # PPP profiles (via index.php)
/ppp/secret/print                   # PPP secrets (via index.php)
/queue/simple/print                 # simple queues
/sys/sch/print                      # scheduler (abbreviated alias path)
/system/clock/print                 # jam router
/system/identity/print              # nama router
/system/logging/print               # logging rules
/system/resource/print              # CPU/memory/uptime
/system/routerboard/print           # hardware info
/system/scheduler/print             # scheduler list
/system/script/print                # scripts (digunakan sebagai DB laporan)
```

### 2.2 WRITE (add/set/remove/enable/disable)

```
# Hotspot user
/ip/hotspot/user/add
/ip/hotspot/user/set
/ip/hotspot/user/remove
/ip/hotspot/user/reset-counters
/ip/hotspot/user/enable             # via process/enablehotspotuser.php
/ip/hotspot/user/disable            # via process/disablehotspotuser.php

# Hotspot profile
/ip/hotspot/user/profile/add
/ip/hotspot/user/profile/set
/ip/hotspot/user/profile/remove

# Hotspot lain
/ip/hotspot/active/remove           # disconnect user aktif
/ip/hotspot/cookie/remove
/ip/hotspot/host/remove             # ← PENTING: ada di mikhmon
/ip/hotspot/ip-binding/set
/ip/hotspot/ip-binding/remove
/ip/hotspot/ip-binding/enable       # via process/pipbinding.php
/ip/hotspot/ip-binding/disable      # via process/pipbinding.php

# PPP
/ppp/active/remove                  # disconnect PPP
/ppp/secret/add                     # via index.php
/ppp/secret/set
/ppp/secret/remove
/ppp/secret/enable
/ppp/secret/disable
/ppp/profile/add                    # via index.php
/ppp/profile/set
/ppp/profile/remove

# Network
/ip/arp/remove
/ip/dhcp-server/lease/remove
/queue/simple/remove

# System
/system/logging/add
/system/scheduler/add
/system/scheduler/set
/system/scheduler/remove
/system/scheduler/enable            # via index.php
/system/scheduler/disable           # via index.php
/system/script/add
/system/script/set
/system/script/remove
/system/reboot                      # via process/reboot.php
/system/shutdown                    # via process/shutdown.php
```

---

## 3. Gap: Ada di Mikhmon, TIDAK Terdaftar di Roskit

Command-command ini dipakai mikhmon tetapi **belum diregistrasi** di `core/definition/`.

| Command Mikhmon | Keterangan | Prioritas |
|----------------|-----------|-----------|
| `/ip/hotspot/host/remove` | Mikhmon bisa hapus host. Roskit tidak punya mutation ini di `hotspot.go` | **TINGGI** |
| `/log/print` | Mikhmon pakai path ini untuk system log. Roskit mendaftar `system/log/print` yang berbeda — path RouterOS yang benar adalah `/log/print` tanpa prefix `system` | **TINGGI** |
| `/ip/hotspot/server/print` | Mikhmon memanggil path ini. Kemungkinan alias dari `/ip/hotspot/print`, perlu diverifikasi di router fisik | Sedang |
| `/sys/sch/print` | Abbreviated path. Tidak standar, hanya untuk kompatibilitas mikhmon lama | Rendah |

### Detail: `/log/print` vs `system/log/print`

Roskit mendaftar:
```go
// system.go
command.Register(command.CommandMeta{
    Path: "system/log/print",   // ← INI SALAH
    Type: command.CommandTypeQuery,
    ...
})
```

RouterOS yang benar (sesuai mikhmon dan dokumentasi RouterOS):
```
/log/print   →   path: "log/print"   (bukan "system/log/print")
```

---

## 4. Gap: Ada di Mikhmon, BELUM LENGKAP di Roskit

Roskit sudah mendaftar command-nya tapi **belum ada service adapter** yang mengeksposnya.

| Domain | Command Mikhmon | Status di Roskit |
|--------|----------------|-----------------|
| PPP | `ppp/secret/print`, CRUD + enable/disable | Registered ✓, ada `adapter/service/ppp.go` ✓ |
| PPP | `ppp/profile/print`, add/set/remove | Registered ✓, **tapi ppp.go hanya punya ListSecrets, AddSecret, DisconnectActive — belum ada ProfileCRUD** |
| Hotspot | `ip/hotspot/user/enable`, `ip/hotspot/user/disable` | Registered ✓, ada di hotspot.go mutations — **perlu dicek apakah hotspot service mengeksposnya** |
| Scheduler | `system/scheduler/enable`, `system/scheduler/disable` | **Tidak terdaftar** di roskit (hanya add/set/remove terdaftar) |
| Queue | `queue/simple/remove` | Registered ✓ (`queue/simple/remove` ada di network.go) |

---

## 5. Gap: Ada di Roskit, TIDAK ADA di Mikhmon (Kapabilitas Baru)

Roskit meregistrasi domain-domain berikut yang **tidak pernah ada di mikhmon**.

### 5.1 Stream Baru (real-time monitoring)

| Path | Measurement | Mikhmon |
|------|-------------|---------|
| `ip/address/print` | `address` | ✗ |
| `ip/route/print` | `route` | ✗ |
| `ip/neighbor/print` | `neighbor` | ✗ |
| `ip/pool/used/print` | `pool_used` | ✗ |
| `ip/dns/static/print` | `dns_static` | ✗ |
| `ip/firewall/nat/print` | `firewall_nat` | ✗ |
| `ip/firewall/address-list/print` | `firewall_address_list` | ✗ |
| `ip/firewall/connection/print` | `firewall_connection` | ✗ |
| `ip/hotspot/walled-garden/print` | `walled_garden` | ✗ |
| `ip/hotspot/walled-garden/ip/print` | `walled_garden_ip` | ✗ |
| `routing/route/print` | `routing_route` | ✗ |
| `routing/ospf/neighbor/print` | `ospf_neighbor` | ✗ |
| `routing/bgp/session/print` | `bgp_session` | ✗ |
| `ip/ipsec/active-peers/print` | `ipsec_active_peers` | ✗ |
| `ip/ipsec/installed-sa/print` | `ipsec_installed_sa` | ✗ |
| `user/active/print` | `user_active` | ✗ |
| `user-manager/session/print` | `user_manager_session` | ✗ |
| `tool/netwatch/print` | `tool_netwatch` | ✗ |
| `interface/bridge/host/print` | `bridge_host` | ✗ |
| `interface/wireless/registration-table/print` | `wireless_registration_table` | ✗ |
| `interface/wifi/registration-table/print` | `wifi_registration_table` | ✗ |
| `ppp/profile/print` | `ppp_profile` | ✗ (mikhmon poll per-request) |

### 5.2 Poll Baru (periodic snapshot)

| Path | Interval | Domain |
|------|----------|--------|
| `system/resource/cpu/print` | 30s | System |
| `system/health/print` | 30s | System |
| `system/logging/action/print` | 30m | System |
| `system/package/print` | 60m | System |
| `system/ntp/client/print` | 60m | System |
| `system/ntp/client/servers/print` | 60m | System |
| `ip/hotspot/profile/print` | 5m | Hotspot (server profile) |
| `ip/hotspot/service-port/print` | 30m | Hotspot |
| `ip/dhcp-server/print` | 5m | Network |
| `ip/dhcp-server/network/print` | 10m | Network |
| `ip/dhcp-server/option/print` | 30m | Network |
| `ip/dhcp-client/print` | 2m | Network |
| `ip/dns/print` | 30m | Network |
| `ip/service/print` | 30m | Network |
| `ip/vrf/print` | 30m | Network |
| `interface/vlan/print` | 5m | Interface |
| `interface/bridge/print` + `bridge/port` | 5m | Interface |
| `interface/ethernet/print` | 5m | Interface |
| `interface/wireless/print` | 5m | Interface |
| `interface/wifi/print` | 5m | Interface (WiFi 6) |
| `interface/wireguard/print` + `peers` | 5m / 2m | Interface |
| `interface/pppoe-client/print` | 2m | Interface |
| `interface/l2tp-client/print` | 2m | Interface |
| `ip/firewall/filter/print` | 10m | Firewall |
| `ip/firewall/mangle/print` | 10m | Firewall |
| `ip/firewall/raw/print` | 10m | Firewall |
| `routing/ospf/instance/print` + `interface` | 5m | Routing |
| `routing/bgp/connection/print` | 5m | Routing |
| `queue/tree/print` + `interface` + `type` | 10m / 1m / 30m | Queue |
| `ip/ipsec/peer/print` + `policy` | 5m | IPSec |
| `user/print` + `user/group` | 30m | User |
| `user-manager/user/print` + `profile` + `router` | 2m–10m | User Manager |
| `radius/print` | 30m | RADIUS |

### 5.3 Mutation Baru (write capability)

Domain yang sama sekali tidak ada di mikhmon:

| Domain | Contoh Command |
|--------|---------------|
| **Firewall filter/mangle/raw** | add, set, remove, enable, disable, move, reset-counters |
| **Routing OSPF/BGP** | instance/area/bgp-connection CRUD |
| **IPSec peer/policy** | add, set, remove, enable, disable |
| **Interface VLAN** | add, set, remove, enable, disable |
| **Interface Bridge** | bridge + bridge/port CRUD |
| **Interface Ethernet** | set, enable, disable, reset-counters |
| **Interface Wireless/WiFi** | set, enable, disable |
| **Interface WireGuard** | wireguard + peers CRUD |
| **Interface PPPoE/L2TP client** | add, set, remove, enable, disable |
| **IP Address** | add, set, remove, enable, disable |
| **IP Route** | add, set, remove, enable, disable |
| **IP DHCP Server** | server + network + option CRUD |
| **IP DHCP Client** | add, set, remove, enable, disable, renew, release |
| **IP DNS** | set + static CRUD + cache/flush |
| **IP Service** | set, enable, disable |
| **IP VRF** | add, set, remove, enable, disable |
| **IP Firewall NAT** | add, set, remove, enable, disable, move |
| **Queue Tree/Type** | CRUD + reset-counters |
| **Hotspot Server** | add, set, remove, enable, disable, reset-html |
| **Hotspot Walled Garden** | add, remove |
| **User (local)** | add, set, remove, enable, disable |
| **User Manager** | user + profile CRUD |
| **RADIUS** | add, set, remove, enable, disable |
| **Tool Netwatch** | add, set, remove, enable, disable |
| **NTP** | set + servers CRUD |
| **Logging** | add, set, remove, enable, disable |
| **Hotspot active/login** | `ip/hotspot/active/login` — mikhmon tidak punya ini |

---

## 6. Perbedaan Cara Kerja per Fitur

### 6.1 Membaca Daftar User Hotspot

**Mikhmon:**
```
Browser buka /index.php?hotspot=users
  → PHP buka TCP 8728
  → $API->comm("/ip/hotspot/user/print")
  → RouterOS kembalikan semua user
  → PHP render HTML
  → TCP tutup
```
Setiap reload halaman = 1 koneksi baru + 1 full query ke router.

**Roskit:**
```
Startup: Stream worker mulai, kirim =follow ke RouterOS
  → RouterOS push setiap ada perubahan user
  → Processor simpan ke Redis: HSET roskit:{id}:hotspot_user:{uid} ...
  → InactiveAggregator update daftar inactive

GET /api/v1/hotspot/users
  → handler → bridge.Query("ip/hotspot/user/print")
  → query.Handler: HGETALL dari Redis (O(1) per user)
  → return JSON (tidak menyentuh router)
```

### 6.2 Monitoring Traffic Interface

**Mikhmon:**
```
Browser buka trafficmonitor.php
  → JavaScript setInterval(8000): AJAX ke traffic.php
  → PHP: $API->comm("/interface/monitor-traffic", [interval=1, once=])
  → Kembalikan 1 data point
  → Highcharts tambah titik baru
```
Setiap 8 detik = 1 koneksi RouterOS baru.

**Roskit:**
```
Startup: MonitorDef worker kirim =interval=1s ke RouterOS
  → RouterOS push tiap 1 detik
  → Processor: timeseries.WritePoint() → InfluxDB (karena WriteTimeSeries=true)
  → pubsub.Publish() → Redis channel (karena "interface_traffic" = realtime)

GET /api/v1/events/sse/telemetry/{routerID}/interface_traffic
  → SSE handler subscribe Redis channel
  → Setiap push dari RouterOS → langsung ke browser via SSE (tanpa polling)
```

### 6.3 Laporan Penjualan

**Mikhmon:**
```
On-login script RouterOS:
  :local record ("$tgl-|-$waktu-|-$user-|-$harga-|-...")
  /system/script/add name=$record source=""
  ← RouterOS = database penjualan

Buka report/selling.php:
  $API->comm("/system/script/print", ["?owner=<bulan>"])
  → parse nama script → render tabel
```
Data penjualan hidup di router. Jika router factory reset = data hilang.

**Roskit:**
```
On-login POST /api/v1/events/on-login
  → ParseOnLoginPut() baca metadata dari script RouterOS
  → VoucherService.RecordSale() → INSERT voucher_sales (PostgreSQL)
  → Jika expire mode "rem"/"remc": hapus user dari router

GET /api/v1/reports/daily?date=...
  → SaleRepo.FetchDailySales() → SELECT dari PostgreSQL
  → Data aman meski router direset
```

### 6.4 PPP Management

**Mikhmon:**
- Hanya punya: `ppp/active/remove` (kick user), `ppp/secret/print/add/set/remove/enable/disable`, `ppp/profile/print/add/set/remove`
- Tidak ada streaming — setiap buka halaman = query ulang

**Roskit:**
- Stream `ppp/secret/print` + `ppp/active/print` + `ppp/profile/print` → Redis
- InactiveAggregator otomatis hitung PPP secret yang tidak aktif (tidak di-session)
- Full mutation CRUD terdaftar

---

## 7. Ringkasan Gap — Action Items

### Harus Diperbaiki (Bug/Ketidaksesuaian)

| # | Masalah | File | Fix |
|---|---------|------|-----|
| 1 | `system/log/print` path salah — RouterOS pakai `/log/print` | `core/definition/system.go` | Ubah path ke `"log/print"` |
| 2 | `ip/hotspot/host/remove` tidak terdaftar | `core/definition/hotspot.go` | Tambah `MutationDef("ip/hotspot/host/remove")` |
| 3 | `system/scheduler/enable` + `disable` tidak terdaftar | `core/definition/system.go` | Tambah ke mutation list scheduler |
| 4 | `MultiSink` + `InactiveAggregator` belum diwire di engine/main | `orchestrator/engine.go`, `cmd/api/main.go` | Wire ke stream sink |

### Fitur Mikhmon yang Perlu Diverifikasi di Roskit

| Fitur Mikhmon | Status Roskit |
|--------------|--------------|
| PPP profile CRUD (add/set/remove) | Registered tapi belum ada service adapter method |
| Hotspot user enable/disable | Registered, perlu dicek di `adapter/service/hotspot.go` |
| Scheduler enable/disable | **Belum terdaftar** |
| Export user CSV/script | Ada di `report_service.go` tapi scope berbeda |

### Kapabilitas Baru Roskit (Tidak Perlu di Mikhmon)

Semua domain di **section 5** — firewall, routing OSPF/BGP, IPSec, WireGuard, VLAN, bridge, WiFi6, User Manager, RADIUS — adalah fitur jaringan level enterprise yang tidak ada di mikhmon dan merupakan ekspansi signifikan scope roskit.
