# Referensi StreamSpec

> Dokumentasi lengkap semua spec yang tersedia di RosKit.

## Daftar Isi

- [Konsep](#konsep)
- [Interface Stats](#interface-stats)
- [System Resource](#system-resource)
- [Queue Simple Stats](#queue-simple-stats)
- [Router Log](#router-log)
- [DHCP Lease](#dhcp-lease)
- [Hotspot Active](#hotspot-active)
- [PPP Active](#ppp-active)
- [PPP Profile](#ppp-profile)
- [PPP Secret](#ppp-secret)

---

## Konsep

Setiap spec mengimplementasikan interface:

```go
type StreamSpec interface {
    Command() []string
    Tag() string
    Measurement() string
    Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error)
}
```

**2 mode streaming:**

| Mode | Ciri | Contoh |
|------|------|--------|
| `follow` | Event-driven, data saat berubah | PPP active, DHCP lease, log |
| `stats + interval` | Periodik, data setiap N detik | Interface stats, queue stats |

---

## Interface Stats

**File:** `internal/spec/interfaces.go`

| Property | Value |
|----------|-------|
| **Command** | `/interface/print =follow =interval=1s` |
| **Tag** | `interface-stats` |
| **Measurement** | `interface_stats` |

### Proplist

`.id`, `name`, `type`, `rx-byte`, `tx-byte`, `rx-packet`, `tx-packet`, `rx-drop`, `tx-drop`, `rx-error`, `tx-error`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `name` | tag | Nama interface (ether1, wlan1) |
| `type` | tag | Tipe interface (ether, wlan, bridge) |
| `rx_byte` | field (int) | Total bytes diterima |
| `tx_byte` | field (int) | Total bytes dikirim |
| `rx_packet` | field (int) | Total paket diterima |
| `tx_packet` | field (int) | Total paket dikirim |
| `rx_drop` | field (int) | Total paket drop (masuk) |
| `tx_drop` | field (int) | Total paket drop (keluar) |
| `rx_error` | field (int) | Total error (masuk) |
| `tx_error` | field (int) | Total error (keluar) |

### Redis Key

```
roskit:{router}:interface_stats:{name}
```

---

## System Resource

**File:** `internal/spec/resource.go`

| Property | Value |
|----------|-------|
| **Command** | `/system/resource/print =interval=5s` |
| **Tag** | `system-resource` |
| **Measurement** | `system_resource` |

### Proplist

`cpu-load`, `free-memory`, `total-memory`, `free-hdd-space`, `total-hdd-space`, `uptime`, `board-name`, `version`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `board_name` | tag | Model hardware (RB750G, CCR1009) |
| `version` | tag | Versi RouterOS (6.49.11, 7.20.8) |
| `cpu_load` | field (int) | Penggunaan CPU (%) |
| `free_memory` | field (int) | Memori tersedia (bytes) |
| `total_memory` | field (int) | Total memori (bytes) |
| `free_hdd_space` | field (int) | Disk tersedia (bytes) |
| `total_hdd_space` | field (int) | Total disk (bytes) |
| `uptime` | field (string) | Uptime router (e.g., "1d8h27m") |

### Redis Key

```
roskit:{router}:system_resource:system
```

---

## Queue Simple Stats

**File:** `internal/spec/queue_simple.go`

| Property | Value |
|----------|-------|
| **Command** | `/queue/simple/print =stats =interval=1s` |
| **Tag** | `queue-simple` |
| **Measurement** | `queue_simple_stats` |

### Proplist

`name`, `target`, `rate`, `packet-rate`, `queued-bytes`, `queued-packets`, `bytes`, `packets`, `dropped`, `total-rate`, `total-bytes`, `total-packets`, `total-dropped`, `total-queued-bytes`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `name` | tag | Nama queue rule (TRAFIK, DIST) |
| `target` | tag | Target addresses |
| `rate` | field (string) | Rate saat ini "upload/download" (bps) |
| `packet_rate` | field (string) | Packet rate "upload/download" |
| `bytes` | field (string) | Total bytes "upload/download" |
| `packets` | field (string) | Total packets "upload/download" |
| `dropped` | field (string) | Total dropped "upload/download" |
| `queued_bytes` | field (string) | Bytes dalam antrian |
| `queued_packets` | field (string) | Paket dalam antrian |
| `total_*` | field (string) | Aggregasi termasuk children |

### Redis Key

```
roskit:{router}:queue_simple_stats:{name}
```

---

## Router Log

**File:** `internal/spec/log.go`

| Property | Value |
|----------|-------|
| **Command** | `/log/print =follow` |
| **Tag** | `log` |
| **Measurement** | `router_log` |

### Proplist

`.id`, `time`, `topics`, `message`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `topics` | tag | Log topics (system,info,account) |
| `log_time` | field (string) | Timestamp dari router |
| `message` | field (string) | Isi log message |

> **Note:** Field bernama `log_time` bukan `time` karena `time` adalah reserved column di InfluxDB.

### Redis Key

```
roskit:{router}:router_log:{.id}
```

---

## DHCP Lease

**File:** `internal/spec/dhcp_lease.go`

| Property | Value |
|----------|-------|
| **Command** | `/ip/dhcp-server/lease/print =follow` |
| **Tag** | `dhcp-lease` |
| **Measurement** | `dhcp_lease` |

### Proplist

`.id`, `address`, `mac-address`, `client-id`, `server`, `lease-time`, `comment`, `disabled`, `block-access`, `rate-limit`, `routes`, `address-lists`, `dhcp-option`, `dhcp-option-set`, `always-broadcast`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `address` | tag | IP address client |
| `mac_address` | tag | MAC address client |
| `server` | tag | DHCP server yang memberikan lease |
| `client_id` | field (string) | DHCP client identifier |
| `lease_time` | field (string) | Durasi lease |
| `disabled` | field (bool) | Apakah lease dinonaktifkan |
| `block_access` | field (bool) | Apakah akses diblokir |
| `rate_limit` | field (string) | Speed limit |
| `comment` | field (string) | Komentar user |

### Redis Key

```
roskit:{router}:dhcp_lease:{.id}
```

---

## Hotspot Active

**File:** `internal/spec/hotspot_active.go`

| Property | Value |
|----------|-------|
| **Command** | `/ip/hotspot/active/print =follow` |
| **Tag** | `hotspot-active` |
| **Measurement** | `hotspot_active` |

### Proplist

`.id`, `user`, `address`, `mac-address`, `uptime`, `bytes-in`, `bytes-out`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `user` | tag | Username hotspot |
| `address` | tag | IP address client |
| `mac_address` | tag | MAC address client |
| `uptime` | field (string) | Durasi sesi |
| `bytes_in` | field (int) | Download bytes |
| `bytes_out` | field (int) | Upload bytes |

### Redis Key

```
roskit:{router}:hotspot_active:{.id}
```

---

## PPP Active

**File:** `internal/spec/ppp_active.go`

| Property | Value |
|----------|-------|
| **Command** | `/ppp/active/print =follow` |
| **Tag** | `ppp-active` |
| **Measurement** | `ppp_active` |

### Proplist

`.id`, `name`, `service`, `caller-id`, `address`, `uptime`, `encoding`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `name` | tag | Username |
| `service` | tag | Tipe (pppoe, pptp, l2tp) |
| `caller_id` | tag | MAC address caller |
| `address` | field (string) | IP address assigned |
| `uptime` | field (string) | Durasi sesi |
| `encoding` | field (string) | Metode enkripsi |

### Redis Key

```
roskit:{router}:ppp_active:{.id}
```

---

## PPP Profile

**File:** `internal/spec/ppp_profile.go`

| Property | Value |
|----------|-------|
| **Command** | `/ppp/profile/print =follow` |
| **Tag** | `ppp-profile` |
| **Measurement** | `ppp_profile` |

### Proplist

`.id`, `name`, `local-address`, `remote-address`, `rate-limit`, `address-list`, `dns-server`, `on-up`, `on-down`, `session-timeout`, `idle-timeout`, `comment`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `name` | tag | Nama profile |
| `local_address` | field (string) | Local IP |
| `remote_address` | field (string) | Remote IP/pool |
| `rate_limit` | field (string) | Speed limit (rx/tx) |
| `dns_server` | field (string) | DNS server |
| `session_timeout` | field (string) | Max session time |
| `idle_timeout` | field (string) | Idle timeout |

### Redis Key

```
roskit:{router}:ppp_profile:{.id}
```

---

## PPP Secret

**File:** `internal/spec/ppp_secret.go`

| Property | Value |
|----------|-------|
| **Command** | `/ppp/secret/print =follow` |
| **Tag** | `ppp-secret` |
| **Measurement** | `ppp_secret` |

### Proplist

`.id`, `name`, `service`, `caller-id`, `profile`, `local-address`, `remote-address`, `routes`, `limit-bytes-in`, `limit-bytes-out`, `disabled`, `comment`

### InfluxDB Schema

| Column | Type | Deskripsi |
|--------|------|-----------|
| `name` | tag | Username |
| `service` | tag | Tipe service (any, pppoe, l2tp) |
| `profile` | tag | Profile yang diassign |
| `caller_id` | field (string) | MAC filter |
| `local_address` | field (string) | Local IP override |
| `remote_address` | field (string) | Remote IP override |
| `routes` | field (string) | Static routes |
| `limit_bytes_in` | field (int) | Download limit (0 = unlimited) |
| `limit_bytes_out` | field (int) | Upload limit (0 = unlimited) |
| `disabled` | field (bool) | Apakah dinonaktifkan |

### Redis Key

```
roskit:{router}:ppp_secret:{.id}
```
