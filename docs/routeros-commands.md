# RouterOS 7.20.8 — Command Registry Reference

Generated from `docs/mikrotik/routeros_7.20.8.json` (541 endpoints).

## Klasifikasi

```
STREAM   → print + follow=true + data berubah SERING/INTENS
           Persistent async connection, push setiap ada perubahan.

POLL     → print + interval (no follow) ATAU follow=true tapi data JARANG berubah
           Sync connection, dieksekusi tiap N detik/menit.

QUERY    → get / find — on-demand, short-cached 10s
MUTATION → add / set / remove / enable / disable / move / run / reset
```

> **Aturan stream vs poll**: Kemampuan `follow=true` dari RouterOS tidak berarti
> harus selalu di-stream. Keputusan didasarkan pada **frekuensi perubahan di
> dunia nyata**, bukan kemampuan teknis RouterOS.

---

## Status Legend

| Simbol | Arti |
|---|---|
| ✓ | Sudah diimplementasi di `core/definition/` |
| — | Belum diimplementasi |

---

## Hotspot (`core/definition/hotspot.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/hotspot/user/print` | Stream | — | `hotspot_user` | ✓ |
| `ip/hotspot/user/profile/print` | Stream | — | `hotspot_profile` | ✓ |
| `ip/hotspot/active/print` | Stream+WriteTS | — | `hotspot_active` | ✓ |
| `ip/hotspot/print` | Stream | — | `hotspot_server` | ✓ |
| `ip/hotspot/host/print` | Stream | — | `hotspot_host` | ✓ |
| `ip/hotspot/cookie/print` | Stream | — | `hotspot_cookie` | ✓ |
| `ip/hotspot/ip-binding/print` | Stream | — | `ip_binding` | ✓ |
| `ip/hotspot/walled-garden/print` | Stream | — | `walled_garden` | ✓ |
| `ip/hotspot/profile/print` | Poll | 5min | `hotspot_profile` | ✓ |
| `ip/hotspot/walled-garden/ip/print` | Stream | — | `walled_garden_ip` | ✓ |
| `ip/hotspot/service-port/print` | Poll | 30min | `hotspot_service_port` | ✓ |
| user: add/set/remove/enable/disable/reset-counters | Mutation | — | — | ✓ |
| user/profile: add/set/remove/reset | Mutation | — | — | ✓ |
| active: remove/login | Mutation | — | — | ✓ |
| cookie: remove | Mutation | — | — | ✓ |
| ip-binding: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| server: add/set/remove/enable/disable/reset-html | Mutation | — | — | ✓ |
| profile: add/set/remove | Mutation | — | — | ✓ |
| walled-garden/ip: add/remove | Mutation | — | — | ✓ |
| user/find, user/get | Query | — | — | ✓ |
| user/profile/find, user/profile/get | Query | — | — | ✓ |
| active/find, active/get | Query | — | — | ✓ |

---

## System (`core/definition/system.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `system/resource/print` | Poll+WriteTS | 60s | `system_resource` | ✓ |
| `system/identity/print` | Poll | 5min | `system_identity` | ✓ |
| `system/clock/print` | Poll | 60s | `system_clock` | ✓ |
| `system/health/print` | Poll | 30s | `system_health` | ✓ |
| `system/routerboard/print` | Poll | 10min | `system_routerboard` | ✓ |
| `system/scheduler/print` | Stream | — | `system_scheduler` | ✓ |
| `system/script/print` | Stream | — | `system_script` | ✓ |
| `system/log/print` | Query | — | `system_log` | ✓ |
| `system/resource/cpu/print` | Poll | 30s | `resource_cpu` | ✓ |
| `system/logging/print` | Poll | 30min | `system_logging` | ✓ |
| `system/logging/action/print` | Poll | 30min | `logging_action` | ✓ |
| `system/package/print` | Poll | 1hr | `system_package` | ✓ |
| `system/ntp/client/print` | Poll | 1hr | `ntp_client` | ✓ |
| `system/ntp/client/servers/print` | Poll | 1hr | `ntp_servers` | ✓ |
| scheduler: add/set/remove | Mutation | — | — | ✓ |
| script: add/set/remove/run | Mutation | — | — | ✓ |
| shutdown, reboot | Mutation | — | — | ✓ |
| logging: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| ntp/client: set | Mutation | — | — | ✓ |
| ntp/client/servers: add/set/remove | Mutation | — | — | ✓ |
| logging/find | Query | — | — | ✓ |
| package/find | Query | — | — | ✓ |

---

## Network (`core/definition/network.go`)

### Interface & Traffic
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `interface/print` | Stream | — | `interface` | ✓ |
| `interface/monitor-traffic` | Stream+WriteTS | — | `interface_traffic` | ✓ |
| interface: set/enable/disable | Mutation | — | — | ✓ |
| interface: find/get | Query | — | — | ✓ |

### IP Address
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/address/print` | Stream | — | `address` | ✓ |
| address: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| address: find/get | Query | — | — | ✓ |

### IP Route
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/route/print` | Stream | — | `route` | ✓ |
| route: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| route: find | Query | — | — | ✓ |

### ARP & Neighbor
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/arp/print` | Stream | — | `arp` | ✓ |
| `ip/neighbor/print` | Stream | — | `neighbor` | ✓ |
| arp: remove | Mutation | — | — | ✓ |
| arp: find | Query | — | — | ✓ |
| neighbor: find | Query | — | — | ✓ |

### IP Pool
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/pool/print` | Stream | — | `ip_pool` | ✓ |
| `ip/pool/used/print` | Stream | — | `pool_used` | ✓ |
| pool: find | Query | — | — | ✓ |
| pool/used: find | Query | — | — | ✓ |

### DHCP Server
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/dhcp-server/lease/print` | Stream | — | `dhcp_lease` | ✓ |
| `ip/dhcp-server/print` | Poll | 5min | `dhcp_server` | ✓ |
| `ip/dhcp-server/network/print` | Poll | 10min | `dhcp_server_network` | ✓ |
| `ip/dhcp-server/option/print` | Poll | 30min | `dhcp_server_option` | ✓ |
| lease: add/remove | Mutation | — | — | ✓ |
| server: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| server/network: add/set/remove | Mutation | — | — | ✓ |
| server/option: add/set/remove | Mutation | — | — | ✓ |
| lease: find | Query | — | — | ✓ |
| server: find | Query | — | — | ✓ |

### DHCP Client
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/dhcp-client/print` | Poll | 2min | `dhcp_client` | ✓ |
| dhcp-client: add/set/remove/enable/disable/renew/release | Mutation | — | — | ✓ |
| dhcp-client: find | Query | — | — | ✓ |

### DNS
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/dns/print` | Poll | 30min | `dns` | ✓ |
| `ip/dns/static/print` | Stream | — | `dns_static` | ✓ |
| dns: set | Mutation | — | — | ✓ |
| dns/static: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| dns/cache: flush | Mutation | — | — | ✓ |
| dns/static: find | Query | — | — | ✓ |

### Queue Simple
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `queue/simple/print` | Stream | — | `queue_simple` | ✓ |
| queue/simple: add/set/remove | Mutation | — | — | ✓ |
| queue/simple: find/get | Query | — | — | ✓ |

### Service & VRF
| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/service/print` | Poll | 30min | `service` | ✓ |
| `ip/vrf/print` | Poll | 30min | `vrf` | ✓ |
| service: set/enable/disable | Mutation | — | — | ✓ |
| vrf: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| service: find | Query | — | — | ✓ |
| vrf: find | Query | — | — | ✓ |

---

## PPP (`core/definition/ppp.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ppp/secret/print` | Stream | — | `ppp_secret` | ✓ |
| `ppp/active/print` | Stream | — | `ppp_active` | ✓ |
| `ppp/profile/print` | Stream | — | `ppp_profile` | ✓ |
| secret: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| active: remove | Mutation | — | — | ✓ |
| profile: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| secret: find | Query | — | — | ✓ |
| active: find | Query | — | — | ✓ |
| profile: find | Query | — | — | ✓ |

---

## Firewall (`core/definition/firewall.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/firewall/nat/print` | Stream | — | `firewall_nat` | ✓ |
| `ip/firewall/filter/print` | Poll | 10min | `firewall_filter` | ✓ |
| `ip/firewall/address-list/print` | Stream | — | `firewall_address_list` | ✓ |
| `ip/firewall/mangle/print` | Poll | 10min | `firewall_mangle` | ✓ |
| `ip/firewall/raw/print` | Poll | 10min | `firewall_raw` | ✓ |
| `ip/firewall/connection/print` | Stream | — | `firewall_connection` | ✓ |
| nat: add/set/remove/enable/disable/move | Mutation | — | — | ✓ |
| filter: add/set/remove/enable/disable/move/reset-counters | Mutation | — | — | ✓ |
| address-list: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| mangle: add/set/remove/enable/disable/move | Mutation | — | — | ✓ |
| raw: add/set/remove/enable/disable/move | Mutation | — | — | ✓ |
| connection: remove | Mutation | — | — | ✓ |
| filter: find | Query | — | — | ✓ |
| address-list: find | Query | — | — | ✓ |
| connection: find | Query | — | — | ✓ |

---

## Interface Types (`core/definition/interface.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `interface/vlan/print` | Poll | 5min | `interface_vlan` | ✓ |
| `interface/bridge/print` | Poll | 5min | `interface_bridge` | ✓ |
| `interface/bridge/port/print` | Poll | 5min | `bridge_port` | ✓ |
| `interface/bridge/host/print` | Stream | — | `bridge_host` | ✓ |
| `interface/ethernet/print` | Poll | 5min | `interface_ethernet` | ✓ |
| `interface/wireless/print` | Poll | 5min | `interface_wireless` | ✓ |
| `interface/wireless/registration-table/print` | Stream | — | `wireless_registration_table` | ✓ |
| `interface/wifi/print` | Poll | 5min | `interface_wifi` | ✓ |
| `interface/wifi/registration-table/print` | Stream | — | `wifi_registration_table` | ✓ |
| `interface/wireguard/print` | Poll | 5min | `interface_wireguard` | ✓ |
| `interface/wireguard/peers/print` | Poll | 2min | `wireguard_peers` | ✓ |
| `interface/pppoe-client/print` | Poll | 2min | `interface_pppoe_client` | ✓ |
| `interface/l2tp-client/print` | Poll | 2min | `interface_l2tp_client` | ✓ |
| vlan: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| bridge: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| bridge/port: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| ethernet: set/enable/disable/reset-counters | Mutation | — | — | ✓ |
| wireless: set/enable/disable | Mutation | — | — | ✓ |
| wifi: set/enable/disable | Mutation | — | — | ✓ |
| wireguard: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| wireguard/peers: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| pppoe-client: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| l2tp-client: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| vlan: find | Query | — | — | ✓ |
| bridge: find | Query | — | — | ✓ |
| bridge/port: find | Query | — | — | ✓ |
| ethernet: find | Query | — | — | ✓ |
| wireless: find | Query | — | — | ✓ |
| wifi: find | Query | — | — | ✓ |
| wireguard: find | Query | — | — | ✓ |
| wireguard/peers: find | Query | — | — | ✓ |

---

## Queue (`core/definition/queue.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `queue/tree/print` | Poll | 10min | `queue_tree` | ✓ |
| `queue/interface/print` | Poll | 1min | `queue_interface` | ✓ |
| `queue/type/print` | Poll | 30min | `queue_type` | ✓ |
| tree: add/set/remove/enable/disable/reset-counters | Mutation | — | — | ✓ |
| type: add/set/remove | Mutation | — | — | ✓ |
| tree: find | Query | — | — | ✓ |
| interface: find | Query | — | — | ✓ |
| type: find | Query | — | — | ✓ |

---

## Routing (`core/definition/routing.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `routing/route/print` | Stream | — | `routing_route` | ✓ |
| `routing/ospf/instance/print` | Poll | 5min | `ospf_instance` | ✓ |
| `routing/ospf/neighbor/print` | Stream | — | `ospf_neighbor` | ✓ |
| `routing/ospf/interface/print` | Poll | 5min | `ospf_interface` | ✓ |
| `routing/bgp/connection/print` | Poll | 5min | `bgp_connection` | ✓ |
| `routing/bgp/session/print` | Stream | — | `bgp_session` | ✓ |
| routing/route: find | Query | — | — | ✓ |
| ospf/instance: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| ospf/area: add/set/remove | Mutation | — | — | ✓ |
| bgp/connection: add/set/remove/enable/disable | Mutation | — | — | ✓ |

---

## User & Access (`core/definition/user.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `user/print` | Poll | 30min | `user` | ✓ |
| `user/active/print` | Stream | — | `user_active` | ✓ |
| `user/group/print` | Poll | 30min | `user_group` | ✓ |
| user: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| user: find | Query | — | — | ✓ |
| active: find | Query | — | — | ✓ |

---

## RADIUS (`core/definition/radius.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `radius/print` | Poll | 30min | `radius` | ✓ |
| radius: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| radius: find | Query | — | — | ✓ |

---

## IPSec (`core/definition/ipsec.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `ip/ipsec/peer/print` | Poll | 5min | `ipsec_peer` | ✓ |
| `ip/ipsec/active-peers/print` | Stream | — | `ipsec_active_peers` | ✓ |
| `ip/ipsec/policy/print` | Poll | 5min | `ipsec_policy` | ✓ |
| `ip/ipsec/installed-sa/print` | Stream | — | `ipsec_installed_sa` | ✓ |
| peer: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| policy: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| peer: find | Query | — | — | ✓ |
| policy: find | Query | — | — | ✓ |
| active-peers: find | Query | — | — | ✓ |

---

## User Manager (`core/definition/user_manager.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `user-manager/user/print` | Poll | 2min | `user_manager_user` | ✓ |
| `user-manager/session/print` | Stream | — | `user_manager_session` | ✓ |
| `user-manager/profile/print` | Poll | 10min | `user_manager_profile` | ✓ |
| `user-manager/router/print` | Poll | 10min | `user_manager_router` | ✓ |
| user: add/set/remove | Mutation | — | — | ✓ |
| profile: add/set/remove | Mutation | — | — | ✓ |
| user: find | Query | — | — | ✓ |
| session: find | Query | — | — | ✓ |

---

## Tools (`core/definition/tools.go`)

| Command | Type | Interval | Measurement | Status |
|---|---|---|---|---|
| `tool/netwatch/print` | Stream | — | `tool_netwatch` | ✓ |
| netwatch: add/set/remove/enable/disable | Mutation | — | — | ✓ |
| netwatch: find | Query | — | — | ✓ |

---

## Stream vs Poll Decision Matrix

| Data | Verdict | Alasan |
|---|---|---|
| `hotspot/active` | Stream | Active sessions berubah tiap detik |
| `interface/monitor-traffic` | Stream+WriteTS | Real-time bandwidth telemetry |
| `system/resource` | Poll 60s+WriteTS | No follow, interval only |
| `ip/address` | Stream | Failover/DHCP renew |
| `ip/route` | Stream | Dynamic routing changes |
| `ip/firewall/connection` | Stream | Ribuan koneksi per detik |
| `ip/firewall/filter` | Poll 10min | Admin-only changes |
| `ip/firewall/mangle` | Poll 10min | Admin-only changes |
| `ip/firewall/address-list` | Stream | Diisi dinamis oleh fw rules |
| `interface/bridge/host` | Stream | MAC table sering berubah |
| `wireless/registration-table` | Stream | Client WiFi connect/disconnect |
| `routing/ospf/neighbor` | Stream | Link state changes |
| `routing/bgp/session` | Stream | Session state changes |
| `ip/ipsec/active-peers` | Stream | Tunnel up/down |
| `user/active` | Stream | System login/logout events |
| `user-manager/session` | Stream | Seperti hotspot/active |
| `ip/dns/static` | Stream | Dynamic DNS (split-DNS, blocklists) |
| `tool/netwatch` | Stream | Host up/down events |
| `interface/vlan` | Poll 5min | Admin-only config |
| `interface/bridge` | Poll 5min | Admin-only config |
| `interface/ethernet` | Poll 5min | Hardware-bound |
| `interface/wireguard` | Poll 5min | Admin-only config |
| `interface/wireguard/peers` | Poll 2min | Handshake timing |
| `ip/dhcp-server` | Poll 5min | Admin-only config |
| `ip/dhcp-client` | Poll 2min | Renew/rebind events |
| `ip/service` | Poll 30min | Very rarely changes |
| `system/logging` | Poll 30min | Admin-only config |
| `system/package` | Poll 1hr | Upgrade-only |
| `system/ntp/client` | Poll 1hr | Very rarely changes |
| `user` | Poll 30min | Admin-only |
| `radius` | Poll 30min | Admin-only config |
| `ip/ipsec/peer` | Poll 5min | Admin config |
| `user-manager/user` | Poll 2min | Grows over time |
| `queue/tree` | Poll 10min | Admin-only config |
| `queue/interface` | Poll 1min | Aggregate stats |
| `ppp/active` | Stream | Session connect/disconnect |

---

## Files

```
internal/roskit/core/definition/
├── hotspot.go        — ip/hotspot/*
├── system.go         — system/*
├── network.go        — ip/address, ip/route, ip/arp, ip/neighbor, ip/pool*,
│                       ip/dhcp-server, ip/dhcp-client, ip/dns, ip/service,
│                       ip/vrf, interface/print, queue/simple
├── ppp.go            — ppp/*
├── firewall.go       — ip/firewall/*
├── interface.go      — interface/vlan, bridge, ethernet, wireless, wifi,
│                       wireguard, pppoe-client, l2tp-client
├── queue.go          — queue/tree, queue/interface, queue/type
├── routing.go        — routing/route, routing/ospf/*, routing/bgp/*
├── user.go           — user/*, user/active, user/group
├── radius.go         — radius/*
├── ipsec.go          — ip/ipsec/*
├── user_manager.go   — user-manager/*
└── tools.go          — tool/netwatch
```
