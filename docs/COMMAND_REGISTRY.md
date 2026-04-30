# RouterOS 7.20.8 Command Registry — Roskit Classification

Source: `docs/mikrotik/routeros_7.20.8.json` (541 endpoints)
Engine: 4 behavior types — **STREAM**, **POLL**, **QUERY**, **MUTATION**

## Classification Rules (Volatility-Based)

| Type | Criteria | Behavior |
|------|----------|----------|
| **STREAM** | Data berubah cepat / kontinu. Termasuk: `monitor`, `monitor-traffic`, `print follow` (data dinamis), `print stats`, `print interval` (metrics kontinu) | RouterOS push data secara real-time atau interval tetap. Client buka koneksi TCP persisten. |
| **POLL** | Data jarang berubah (config, profile, singleton). `print` dengan atau tanpa `follow` tapi data statis | Client query periodik via ticker. Snapshot saja. |
| **QUERY** | `find`, `get` commands | On-demand lookup. Optionally short-cached. |
| **MUTATION** | `add`, `set`, `remove`, `enable`, `disable`, `move`, `reset-*`, `run`, etc. | Write operation. Invalidates related cache. |

### Mekanisme STREAM di RouterOS

| Mekanisme | Contoh | Karakteristik |
|-----------|--------|---------------|
| `print =follow` | `/ip/hotspot/user/print =follow` | Event-driven: push hanya saat data berubah |
| `monitor` | `/interface/ethernet/monitor once interval=1s` | Interval-driven: push setiap N detik, selalu kontinu |
| `monitor-traffic` | `/interface/monitor-traffic interface=ether1 once interval=1s` | Per-interface traffic stats (rx/tx bytes, packets) |
| `print =stats` | `/queue/simple/print =stats =follow` | Counter (bytes/packets) + follow untuk update kontinu |
| `print =interval=1s` | `/system/resource/print =interval=1s` | Snapshot periodik, bukan event-driven |

### Volatility Decision Matrix

| Pattern | Volatilitas | Tipe | Contoh |
|---------|-------------|------|--------|
| `*/monitor` | Selalu TINGGI | STREAM | `interface/monitor-traffic`, `interface/ethernet/monitor` |
| `*/print stats` | Selalu TINGGI | STREAM | `queue/simple/print stats` |
| `print follow` + sessions/active | TINGGI | STREAM | `ppp/active/print`, `ip/hotspot/active/print` |
| `print follow` + user/config | SEDANG | STREAM (TTL pendek) | `ppp/secret/print`, `ip/hotspot/user/print` |
| `print follow` + profiles/rules | RENDAH | POLL (follow utk change detection, TTL panjang) | `ppp/profile/print`, `ip/firewall/filter/print` |
| `print` tanpa follow + singleton | SANGAT RENDAH | POLL (interval panjang) | `ppp/aaa`, `system/identity` |

## Legend

- **Mechanism**: Cara kerja stream: `follow`, `monitor`, `monitor-traffic`, `stats`, `interval`
- **Measurement**: Redis key segment + InfluxDB measurement (`roskit:{routerID}:{measurement}:{id}`)
- **Index**: Secondary Redis lookup field
- **TS**: Writes to InfluxDB time-series
- **TTL**: Default cache TTL
- **Interval**: Poll interval (POLL type only)
- **Status**: REGISTERED = sudah ada di `core/definition/`, NEW = belum diregistrasi

---

## 1. Hotspot (Category: `hotspot`)

### STREAM

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `ip/hotspot/user/print` | follow | `hotspot_user` | `name` | No | 5m | REGISTERED |
| `ip/hotspot/active/print` | follow | `hotspot_active` | — | Yes | 5m | REGISTERED |
| `ip/hotspot/active/print stats` | follow+stats | `hotspot_active_stats` | — | Yes | 30s | NEW |
| `ip/hotspot/host/print` | follow | `hotspot_host` | — | No | 5m | REGISTERED |
| `ip/hotspot/cookie/print` | follow | `hotspot_cookie` | — | No | 5m | REGISTERED |
| `ip/hotspot/ip-binding/print` | follow | `ip_binding` | — | No | 5m | REGISTERED |
| `ip/hotspot/walled-garden/print` | follow | `walled_garden` | — | No | 5m | REGISTERED |
| `ip/hotspot/walled-garden/ip/print` | follow | `walled_garden_ip` | — | No | 5m | REGISTERED |
| `ip/hotspot/print` | follow | `hotspot_server` | `name` | No | 5m | REGISTERED |

### POLL (data jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `ip/hotspot/user/profile/print` | `hotspot_profile` | 5m | 10m | REGISTERED (was STREAM, reclassify) |
| `ip/hotspot/profile/print` | `hotspot_server_profile` | 5m | 10m | REGISTERED |
| `ip/hotspot/service-port/print` | `hotspot_service_port` | 30m | 60m | REGISTERED |

### QUERY

| Path | Status |
|------|--------|
| `ip/hotspot/user/{find,get}` | REGISTERED |
| `ip/hotspot/user/profile/{find,get}` | REGISTERED |
| `ip/hotspot/active/{find,get}` | REGISTERED |
| `ip/hotspot/{cookie,host,ip-binding,walled-garden,walled-garden/ip}/{find,get}` | NEW |
| `ip/hotspot/{find,get}` | NEW |
| `ip/hotspot/profile/{find,get}` | NEW |
| `ip/hotspot/service-port/{find,get}` | NEW |

### MUTATION

| Path | Status | Notes |
|------|--------|-------|
| `ip/hotspot/user/{add,set,remove,enable,disable,reset-counters}` | REGISTERED | |
| `ip/hotspot/user/profile/{add,set,remove,reset}` | REGISTERED | |
| `ip/hotspot/active/{remove,login}` | REGISTERED | |
| `ip/hotspot/active/{add,set,reset}` | NEW | |
| `ip/hotspot/host/{remove}` | REGISTERED | |
| `ip/hotspot/host/{make-binding,reset}` | NEW | |
| `ip/hotspot/cookie/remove` | REGISTERED | |
| `ip/hotspot/ip-binding/{add,set,remove,enable,disable}` | REGISTERED | |
| `ip/hotspot/ip-binding/move` | NEW | |
| `ip/hotspot/walled-garden/{add,set,remove}` | REGISTERED | |
| `ip/hotspot/walled-garden/{enable,disable,move,reset-counters,reset-counters-all}` | NEW | |
| `ip/hotspot/walled-garden/ip/{add,remove}` | REGISTERED | |
| `ip/hotspot/walled-garden/ip/{set,enable,disable,move}` | NEW | |
| `ip/hotspot/{add,set,remove,enable,disable,reset-html}` | REGISTERED | |
| `ip/hotspot/profile/{add,set,remove}` | REGISTERED | |
| `ip/hotspot/service-port/{set,enable,disable}` | NEW | |
| `ip/hotspot/setup` | NEW | Auto-configure hotspot |

---

## 2. Firewall (Category: `firewall`)

### STREAM (data dinamis — connections, NAT, address-list berubah sering)

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `ip/firewall/connection/print` | follow | `firewall_connection` | — | No | 5m | REGISTERED |
| `ip/firewall/nat/print` | follow | `firewall_nat` | — | No | 5m | REGISTERED |
| `ip/firewall/address-list/print` | follow | `firewall_address_list` | — | No | 5m | REGISTERED |

### POLL (rules/config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `ip/firewall/filter/print` | `firewall_filter` | 10m | 20m | REGISTERED |
| `ip/firewall/mangle/print` | `firewall_mangle` | 10m | 20m | REGISTERED |
| `ip/firewall/raw/print` | `firewall_raw` | 10m | 20m | REGISTERED |
| `ip/firewall/calea/print` | `firewall_calea` | 5m | 10m | NEW |
| `ip/firewall/layer7-protocol/print` | `firewall_layer7` | 30m | 60m | NEW |
| `ip/firewall/service-port/print` | `firewall_service_port` | 30m | 60m | NEW |
| `ip/firewall/connection/tracking/print` | `connection_tracking` | 10m | 20m | NEW |

### QUERY

| Path | Status |
|------|--------|
| `ip/firewall/{filter,address-list,mangle,connection}/{find,get}` | REGISTERED / NEW (get) |
| `ip/firewall/{nat,raw,calea,layer7-protocol,service-port}/{find,get}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `ip/firewall/{filter,mangle,raw}/{add,set,remove,enable,disable,move}` | REGISTERED |
| `ip/firewall/{filter,mangle,raw}/reset-counters` | REGISTERED |
| `ip/firewall/address-list/{add,set,remove,enable,disable}` | REGISTERED |
| `ip/firewall/connection/remove` | REGISTERED |
| `ip/firewall/nat/{add,set,remove,enable,disable,move}` | REGISTERED |
| `ip/firewall/{filter,mangle,raw,nat}/reset-counters-all` | NEW |
| `ip/firewall/calea/{add,set,remove,enable,disable,move,reset-counters}` | NEW |
| `ip/firewall/layer7-protocol/{add,set,remove}` | NEW |
| `ip/firewall/service-port/{set,enable,disable}` | NEW |
| `ip/firewall/connection/tracking/set` | NEW |

---

## 3. DHCP (Category: `dhcp`)

### STREAM (leases berubah sering — connect/disconnect)

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `ip/dhcp-server/lease/print` | follow | `dhcp_lease` | `address` | No | 5m | REGISTERED |
| `ip/dhcp-server/alert/print` | follow | `dhcp_alert` | — | No | 5m | NEW |
| `ip/dhcp-server/matcher/print` | follow | `dhcp_matcher` | — | No | 5m | NEW |

### POLL (server config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `ip/dhcp-server/print` | `dhcp_server` | 5m | 10m | REGISTERED |
| `ip/dhcp-server/network/print` | `dhcp_network` | 10m | 20m | REGISTERED |
| `ip/dhcp-server/option/print` | `dhcp_option` | 30m | 60m | REGISTERED |
| `ip/dhcp-client/print` | `dhcp_client` | 2m | 4m | REGISTERED |
| `ip/dhcp-client/option/print` | `dhcp_client_option` | 5m | 10m | NEW |
| `ip/dhcp-server/config/print` | `dhcp_config` | 30m | 60m | NEW |
| `ip/dhcp-server/option/sets/print` | `dhcp_option_set` | 30m | 60m | NEW |

### STREAM — Monitor

| Path | Mechanism | Measurement | TS | TTL | Status |
|------|-----------|-------------|----|-----|--------|
| `ip/dhcp-relay/monitor` | monitor | `dhcp_relay_status` | No | 60s | NEW |

### QUERY

| Path | Status |
|------|--------|
| `ip/dhcp-server/lease/find` | REGISTERED |
| `ip/dhcp-server/find` | REGISTERED |
| `ip/dhcp-server/{lease,get,network,option,alert,matcher}/{find,get}` | NEW |
| `ip/dhcp-client/{find,get}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `ip/dhcp-server/{add,set,remove,enable,disable}` | REGISTERED |
| `ip/dhcp-server/lease/{add,remove}` | REGISTERED |
| `ip/dhcp-server/network/{add,set,remove}` | REGISTERED |
| `ip/dhcp-server/option/{add,set,remove}` | REGISTERED |
| `ip/dhcp-client/{add,set,remove,enable,disable,renew,release}` | REGISTERED |
| `ip/dhcp-server/lease/{set,enable,disable,check-status,make-static,send-reconfigure}` | NEW |
| `ip/dhcp-server/{alert,matcher}/{add,set,remove,enable,disable}` | NEW |
| `ip/dhcp-server/config/set` | NEW |
| `ip/dhcp-server/option/sets/{add,set,remove}` | NEW |
| `ip/dhcp-server/setup` | NEW |

---

## 4. Queue (Category: `queue`)

### STREAM (data bandwidth — berubah cepat)

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `queue/simple/print` | follow | `queue_simple` | `name` | No | 5m | REGISTERED |
| `queue/simple/print stats` | follow+stats | `queue_simple_stats` | `name` | **Yes** | 30s | NEW |
| `queue/tree/print stats` | follow+stats | `queue_tree_stats` | — | **Yes** | 30s | NEW |
| `queue/monitor` | monitor | `queue_monitor` | — | **Yes** | 10s | NEW |

### POLL (config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `queue/tree/print` | `queue_tree` | 10m | 20m | REGISTERED |
| `queue/interface/print` | `queue_interface` | 1m | 2m | REGISTERED |
| `queue/type/print` | `queue_type` | 30m | 60m | REGISTERED |

### QUERY

| Path | Status |
|------|--------|
| `queue/{simple,tree,interface,type}/{find,get}` | REGISTERED / NEW (get) |

### MUTATION

| Path | Status |
|------|--------|
| `queue/simple/{add,set,remove}` | REGISTERED |
| `queue/tree/{add,set,remove,enable,disable,reset-counters}` | REGISTERED |
| `queue/type/{add,set,remove}` | REGISTERED |
| `queue/simple/{reset-counters,reset-counters-all,move,enable,disable}` | NEW |
| `queue/tree/reset-counters-all` | NEW |

---

## 5. Interface (Category: `interface`)

### STREAM — Print follow (data dinamis)

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `interface/print` | follow | `interface` | `name` | No | 5m | REGISTERED |
| `interface/bridge/host/print` | follow | `bridge_host` | — | No | 5m | REGISTERED |
| `interface/wireless/registration-table/print` | follow | `wireless_registration` | — | No | 5m | REGISTERED |
| `interface/wifi/registration-table/print` | follow | `wifi_registration` | — | No | 5m | REGISTERED |
| `interface/list/member/print` | follow | `interface_list_member` | — | No | 5m | NEW |
| `interface/wifi/capsman/remote-cap/print` | follow | `wifi_remote_cap` | — | No | 5m | NEW |

### STREAM — Monitor (kontinu, interval-driven)

| Path | What It Monitors | Measurement | TS | TTL | Status |
|------|-----------------|-------------|----|-----|--------|
| `interface/monitor-traffic` | Per-interface rx/tx bytes, packets | `interface_traffic` | **Yes** | 30s | REGISTERED |
| `interface/ethernet/monitor` | Per-port rx/tx, errors, speed, status | `ethernet_port_stats` | **Yes** | 10s | NEW |
| `interface/pppoe-server/monitor` | PPPoE server sessions | `pppoe_server_status` | **Yes** | 30s | NEW |
| `interface/pppoe-client/monitor` | PPPoE client link state | `pppoe_client_status` | No | 30s | NEW |
| `interface/lte/monitor` | LTE signal RSRP/RSINQ, cell, band | `lte_signal` | **Yes** | 30s | NEW |
| `interface/bonding/monitor` | LACP state, active slaves | `bonding_status` | No | 30s | NEW |
| `interface/bonding/monitor-slaves` | Individual slave status | `bonding_slave_status` | No | 30s | NEW |
| `interface/bridge/monitor` | Bridge STP state | `bridge_status` | No | 30s | NEW |
| `interface/bridge/port/monitor` | Per-port STP role, path cost | `bridge_port_status` | No | 30s | NEW |
| `interface/vrrp/monitor` | VRRP state changes | `vrrp_status` | No | 30s | NEW |
| `interface/wifi/monitor` | WiFi radio stats, noise, clients | `wifi_radio_stats` | **Yes** | 30s | NEW |
| `interface/wireless/monitor` | Legacy wireless radio stats | `wireless_radio_stats` | **Yes** | 30s | NEW |
| `interface/l2tp-client/monitor` | L2TP tunnel state | `l2tp_client_status` | No | 30s | NEW |
| `interface/l2tp-server/monitor` | L2TP server state | `l2tp_server_status` | No | 30s | NEW |
| `interface/ovpn-client/monitor` | OpenVPN client tunnel | `ovpn_client_status` | No | 30s | NEW |
| `interface/ovpn-server/monitor` | OpenVPN server tunnel | `ovpn_server_status` | No | 30s | NEW |
| `interface/pptp-client/monitor` | PPTP tunnel state | `pptp_client_status` | No | 30s | NEW |
| `interface/pptp-server/monitor` | PPTP server state | `pptp_server_status` | No | 30s | NEW |
| `interface/sstp-client/monitor` | SSTP tunnel state | `sstp_client_status` | No | 30s | NEW |
| `interface/sstp-server/monitor` | SSTP server state | `sstp_server_status` | No | 30s | NEW |
| `interface/vpls/monitor` | VPLS tunnel state | `vpls_status` | No | 30s | NEW |
| `interface/ppp-client/monitor` | PPP client modem state | `ppp_client_status` | No | 30s | NEW |
| `interface/ppp-server/monitor` | PPP server state | `ppp_server_status` | No | 30s | NEW |

### POLL (interface config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `interface/vlan/print` | `vlan` | 5m | 10m | REGISTERED |
| `interface/bridge/print` | `bridge` | 5m | 10m | REGISTERED |
| `interface/bridge/port/print` | `bridge_port` | 5m | 10m | REGISTERED |
| `interface/ethernet/print` | `ethernet` | 5m | 10m | REGISTERED |
| `interface/wireless/print` | `wireless` | 5m | 10m | REGISTERED |
| `interface/wifi/print` | `wifi` | 5m | 10m | REGISTERED |
| `interface/wireguard/print` | `wireguard` | 5m | 10m | REGISTERED |
| `interface/wireguard/peers/print` | `wireguard_peers` | 2m | 4m | REGISTERED |
| `interface/pppoe-client/print` | `pppoe_client` | 2m | 4m | REGISTERED |
| `interface/l2tp-client/print` | `l2tp_client` | 2m | 4m | REGISTERED |
| `interface/bonding/print` | `bonding` | 5m | 10m | NEW |
| `interface/vrrp/print` | `vrrp` | 2m | 4m | NEW |
| `interface/list/print` | `interface_list` | 5m | 10m | NEW |
| `interface/pppoe-server/print` | `pppoe_server` | 5m | 10m | NEW |
| `interface/pppoe-server/server/print` | `pppoe_server_config` | 10m | 20m | NEW |
| `interface/lte/print` | `lte` | 2m | 4m | NEW |
| `interface/lte/apn/print` | `lte_apn` | 10m | 20m | NEW |
| `interface/ovpn-client/print` | `ovpn_client` | 2m | 4m | NEW |
| `interface/ovpn-server/print` | `ovpn_server` | 5m | 10m | NEW |
| `interface/macsec/print` | `macsec` | 5m | 10m | NEW |
| `interface/macvlan/print` | `macvlan` | 5m | 10m | NEW |
| `interface/veth/print` | `veth` | 5m | 10m | NEW |
| `interface/vxlan/print` | `vxlan` | 5m | 10m | NEW |
| `interface/vpls/print` | `vpls` | 5m | 10m | NEW |
| `interface/pptp-client/print` | `pptp_client` | 2m | 4m | NEW |
| `interface/sstp-client/print` | `sstp_client` | 2m | 4m | NEW |
| `interface/wifi/radio/print` | `wifi_radio` | 5m | 10m | NEW |
| `interface/wifi/{aaa,access-list,channel,configuration,datapath,interworking,security,provisioning,steering}/print` | various | 5-30m | 10-60m | NEW |
| `interface/wireless/{access-list,security-profiles,connect-list,channels}/print` | various | 5-30m | 10-60m | NEW |

### QUERY

| Path | Status |
|------|--------|
| `interface/{find,get}` | REGISTERED |
| `interface/{vlan,bridge,bridge/port,ethernet,wireless,wifi,wireguard,wireguard/peers,pppoe-client,l2tp-client}/find` | REGISTERED |
| `interface/{bonding,vrrp,list,list/member,pppoe-server,wifi/radio}/{find,get}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `interface/{vlan,bridge,bridge/port,ethernet,wireless,wifi,wireguard,wireguard/peers,pppoe-client,l2tp-client}/*` | REGISTERED |
| `interface/{bonding,vrrp,list,list/member,pppoe-server,pppoe-server/server}/*` | NEW |
| `interface/wifi/{aaa,access-list,channel,configuration,datapath,security,provisioning,steering}/*` | NEW |
| `interface/wireless/{access-list,security-profiles,connect-list}/*` | NEW |

---

## 6. System (Category: `system`)

### STREAM — Continuous metrics

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `system/resource/print` | interval=1s | `system_resource` | — | **Yes** | 10s | REGISTERED |
| `system/resource/monitor` | monitor | `system_resource_monitor` | — | **Yes** | 10s | NEW |
| `system/scheduler/print` | follow | `system_scheduler` | `name` | No | 5m | REGISTERED |
| `system/script/print` | follow | `system_script` | `name` | No | 5m | REGISTERED |
| `system/script/environment/print` | follow | `script_env` | `name` | No | 5m | NEW |
| `system/script/job/print` | follow | `script_job` | — | No | 5m | NEW |

### STREAM — Monitor

| Path | What It Monitors | Measurement | TS | TTL | Status |
|------|-----------------|-------------|----|-----|--------|
| `system/resource/monitor` | CPU/RAM continuous | `system_resource_monitor` | **Yes** | 10s | NEW |
| `system/ups/monitor` | UPS battery/voltage | `ups_status` | **Yes** | 30s | NEW |
| `system/gps/monitor` | GPS coordinates | `gps_location` | No | 10s | NEW |

### POLL (config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `system/identity/print` | `system_identity` | 5m | 10m | REGISTERED |
| `system/clock/print` | `system_clock` | 60s | 120s | REGISTERED |
| `system/health/print` | `system_health` | 30s | 60s | REGISTERED |
| `system/routerboard/print` | `system_routerboard` | 10m | 15m | REGISTERED |
| `system/resource/cpu/print` | `system_resource_cpu` | 30s | 60s | REGISTERED |
| `system/logging/print` | `system_logging` | 30m | 60m | REGISTERED |
| `system/logging/action/print` | `system_logging_action` | 30m | 60m | REGISTERED |
| `system/package/print` | `system_package` | 60m | 120m | REGISTERED |
| `system/ntp/client/print` | `ntp_client` | 60m | 120m | REGISTERED |
| `system/ntp/client/servers/print` | `ntp_servers` | 60m | 120m | REGISTERED |
| `system/resource/hardware/print` | `system_hardware` | 30m | 60m | NEW |
| `system/resource/irq/print` | `system_irq` | 10m | 20m | NEW |
| `system/resource/irq/rps/print` | `system_irq_rps` | 10m | 20m | NEW |
| `system/ntp/key/print` | `ntp_key` | 60m | 120m | NEW |
| `system/ntp/server/print` | `ntp_server` | 60m | 120m | NEW |
| `system/package/update/print` | `system_package_update` | 60m | 120m | NEW |

### QUERY

| Path | Status |
|------|--------|
| `log/print` | REGISTERED |
| `system/{logging,package}/find` | REGISTERED |
| `system/{scheduler,script,health,resource,resource/cpu}/{find,get}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `system/scheduler/{add,set,remove,enable,disable}` | REGISTERED |
| `system/script/{add,set,remove,run}` | REGISTERED |
| `system/{shutdown,reboot}` | REGISTERED |
| `system/logging/{add,set,remove,enable,disable}` | REGISTERED |
| `system/ntp/client/set` | REGISTERED |
| `system/ntp/client/servers/{add,set,remove}` | REGISTERED |
| `system/script/environment/{set,remove}`, `system/script/job/remove` | NEW |
| `system/ntp/{key,set,server/set}` | NEW |
| `system/package/{enable,disable,downgrade,uninstall}` | NEW |
| `system/package/update/{check-for-updates,download,install,cancel}` | NEW |

---

## 7. PPP / PPPoE (Category: `ppp`)

### STREAM (PPPoE sessions — berubah real-time)

| Path | Mechanism | Volatility | Measurement | Index | TS | TTL | Status |
|------|-----------|------------|-------------|-------|----|-----|--------|
| `ppp/active/print` | follow | TINGGI | `ppp_active` | `name` | No | 5m | REGISTERED |
| `ppp/secret/print` | follow | SEDANG | `ppp_secret` | `name` | No | 5m | REGISTERED |

### POLL (profiles/config — jarang berubah)

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `ppp/profile/print` | `ppp_profile` | 10m | 20m | REGISTERED (was STREAM, reclassify) |
| `ppp/aaa/print` | `ppp_aaa` | 30m | 60m | NEW |
| `ppp/l2tp-secret/print` | `ppp_l2tp_secret` | 30m | 60m | NEW |

### QUERY

| Path | Status |
|------|--------|
| `ppp/{secret,active,profile}/find` | REGISTERED |
| `ppp/{secret,active,profile}/get` | NEW |
| `ppp/l2tp-secret/{find,get}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `ppp/secret/{add,set,remove,enable,disable}` | REGISTERED |
| `ppp/active/remove` | REGISTERED |
| `ppp/profile/{add,set,remove,enable,disable}` | REGISTERED |
| `ppp/aaa/set` | NEW |
| `ppp/l2tp-secret/{add,set,remove}` | NEW |

---

## 8. DNS (Category: `dns`)

### STREAM

| Path | Mechanism | Measurement | Index | TS | TTL | Status |
|------|-----------|-------------|-------|----|-----|--------|
| `ip/dns/static/print` | follow | `dns_static` | `name` | No | 5m | REGISTERED |
| `ip/dns/forwarders/print` | follow | `dns_forwarders` | — | No | 5m | NEW |

### POLL

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `ip/dns/print` | `dns_settings` | 30m | 60m | REGISTERED |
| `ip/dns/adlist/print` | `dns_adlist` | 30m | 60m | NEW |
| `ip/dns/cache/print` | `dns_cache` | 5m | 10m | NEW |
| `ip/dns/cache/all/print` | `dns_cache_all` | 5m | 10m | NEW |

### QUERY

| Path | Status |
|------|--------|
| `ip/dns/static/find` | REGISTERED |
| `ip/dns/{static,cache,cache/all}/{get,find}` | NEW |

### MUTATION

| Path | Status |
|------|--------|
| `ip/dns/set`, `ip/dns/static/{add,set,remove,enable,disable}`, `ip/dns/cache/flush` | REGISTERED |
| `ip/dns/{static/move,adlist,forwarders}/*` | NEW |

---

## 9. IP Address / Route / ARP (Category: `network`)

### STREAM

| Path | Measurement | Index | TS | TTL | Status |
|------|-------------|-------|----|-----|--------|
| `ip/address/print` | `ip_address` | `address` | No | 5m | REGISTERED |
| `ip/route/print` | `ip_route` | — | No | 5m | REGISTERED |
| `ip/arp/print` | `ip_arp` | `address` | No | 5m | REGISTERED |

### QUERY / MUTATION

(Sama seperti sebelumnya — tidak ada perubahan)

---

## 10. RADIUS (Category: `radius`)

### POLL

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `radius/print` | `radius` | 30m | 60m | REGISTERED |
| `radius/incoming/print` | `radius_incoming` | 5m | 10m | NEW |

### STREAM — Monitor

| Path | What It Monitors | Measurement | TS | TTL | Status |
|------|-----------------|-------------|----|-----|--------|
| `radius/monitor` | RADIUS connectivity, RTT | `radius_monitor` | **Yes** | 30s | NEW |
| `radius/incoming/monitor` | RADIUS incoming listener | `radius_incoming_status` | No | 60s | NEW |

### QUERY / MUTATION

(Sama seperti sebelumnya + `radius/reset-counters`, `radius/incoming/{set,reset-counters}`)

---

## 11. User (Category: `user`)

### STREAM

| Path | Measurement | Index | TS | TTL | Status |
|------|-------------|-------|----|-----|--------|
| `user/active/print` | `user_active` | — | No | 5m | REGISTERED |

### POLL

| Path | Measurement | Interval | TTL | Status |
|------|-------------|----------|-----|--------|
| `user/print` | `user` | 30m | 60m | REGISTERED |
| `user/group/print` | `user_group` | 30m | 60m | REGISTERED |
| `user/{aaa,settings,ssh-keys,ssh-keys/private}/print` | various | 30m | 60m | NEW |

### QUERY / MUTATION

(Sama seperti sebelumnya)

---

## 12. SNMP (Category: `snmp`) — ENTIRELY NEW

### POLL

| Path | Measurement | Interval | TTL |
|------|-------------|----------|-----|
| `snmp/print` | `snmp_settings` | 30m | 60m |
| `snmp/community/print` | `snmp_community` | 30m | 60m |

### QUERY: `snmp/community/{find,get}`, `snmp/get`
### MUTATION: `snmp/set`, `snmp/community/{add,set,remove,enable,disable}`, `snmp/send-trap`

---

## 13. Log (Category: `log`)

### QUERY: `log/{print,find,get}` (REGISTERED + NEW)
### MUTATION: `log/{info,warning,error}` (NEW — inject log entries)

---

## 14. User Manager (Category: `user_manager`)

Not in JSON schema 7.20.8. Registered manually.

### STREAM: `user-manager/session/print` (REGISTERED)
### POLL: `user-manager/{user,profile,router}/print` (REGISTERED)
### QUERY: `user-manager/{user,session}/find` (REGISTERED)
### MUTATION: `user-manager/{user,profile}/{add,set,remove}` (REGISTERED)

### STREAM — Monitor (NEW)

| Path | What It Monitors | Measurement | Status |
|------|-----------------|-------------|--------|
| `user-manager/monitor` | User manager overall stats | `um_status` | NEW |
| `user-manager/router/monitor` | Router link state | `um_router_status` | NEW |
| `user-manager/user/monitor` | User session state | `um_user_status` | NEW |

---

## 15. Other Monitor Commands (non-interface)

| Path | What It Monitors | Category | TS | Status |
|------|-----------------|----------|----|--------|
| `ip/proxy/monitor` | HTTP proxy stats | proxy | **Yes** | NEW |
| `ip/traffic-flow/monitor` | NetFlow/IPFIX stats | network | **Yes** | NEW |
| `disk/monitor-traffic` | Disk I/O stats | system | No | NEW |
| `system/ntp/monitor-peers` | NTP peer sync | system | No | NEW |
| `file/sync/monitor` | File sync state | system | No | NEW |

---

## 16. Stats Commands (arg pada print, bukan command terpisah)

Berikut `print stats` yang bernilai untuk ISP. Digunakan dengan: `/path/print =stats =follow`

| Path + stats | What Stats Show | Stream? | InfluxDB? | Priority |
|-------------|----------------|---------|-----------|----------|
| `queue/simple/print stats` | Per-queue bytes/packets in/out, dropped | Yes (follow+stats) | **Yes** | HIGH |
| `queue/tree/print stats` | Per-tree-node counters | Yes | **Yes** | MEDIUM |
| `interface/print stats` | Per-interface rx/tx bytes, packets, errors | Yes | **Yes** | HIGH |
| `interface/ethernet/print stats` | Per-ethernet-port counters | Yes | **Yes** | HIGH |
| `ip/hotspot/active/print stats` | Active session byte counters | Yes | **Yes** | HIGH |
| `interface/wifi/registration-table/print stats` | Per-WiFi-client traffic | Yes | **Yes** | MEDIUM |
| `interface/wireless/registration-table/print stats` | Per-wireless-client traffic | Yes | **Yes** | MEDIUM |
| `ip/firewall/filter/print stats` | Per-rule packet/byte counters | No (poll) | Yes | MEDIUM |
| `ip/firewall/nat/print stats` | Per-NAT-rule counters | No (poll) | Yes | LOW |

---

## Priority Recommendations

### HIGH — Core ISP monitoring, register soon

| New Path | Type | Why |
|----------|------|-----|
| `interface/ethernet/monitor` | STREAM (monitor) | Physical port monitoring — rx/tx, errors, speed |
| `queue/simple/print stats` | STREAM (follow+stats) | Per-user bandwidth consumption — billing/analytics |
| `queue/monitor` | STREAM (monitor) | Aggregate queue traffic |
| `interface/pppoe-server/monitor` | STREAM (monitor) | PPPoE server session monitoring |
| `ppp/aaa/print` + `ppp/aaa/set` | POLL + MUTATION | PPP/PPPoE authentication config |
| `interface/pppoe-server/*` | POLL + MUTATION | PPPoE server management |
| `interface/bonding/*` + `interface/bonding/monitor` | POLL + STREAM | Link aggregation + LACP health |
| `interface/vrrp/*` + `interface/vrrp/monitor` | POLL + STREAM | Gateway redundancy |
| `radius/monitor` + `radius/incoming/monitor` | STREAM (monitor) | RADIUS server RTT + incoming CoA |
| `ip/dns/adlist/*` | POLL + MUTATION | DNS ad-blocking (value-add) |
| `snmp/*` | POLL + MUTATION | SNMP monitoring |
| `system/resource/monitor` | STREAM (monitor) | Continuous CPU/RAM metrics |

### MEDIUM — Operational quality

| New Path | Why |
|----------|-----|
| `interface/lte/monitor` | LTE signal quality for WAN monitoring |
| `interface/wifi/monitor` | WiFi radio health |
| `interface/bridge/port/monitor` | STP state changes |
| `ip/firewall/connection/tracking/*` | Conntrack tuning |
| `interface/lte/*` | LTE failover WAN config |
| `ip/dhcp-server/lease/make-static` | Lease to static conversion |
| `system/ups/monitor` | UPS monitoring |

### LOW — Admin tooling

| New Path | Why |
|----------|-----|
| Various `get` queries, `reset-counters-all` | Admin convenience |
| `user/ssh-keys/*` | SSH key management |
| `system/package/update/*` | Firmware management |

---

## Statistics

| | REGISTERED | NEW | Total |
|--|-----------|-----|-------|
| STREAM (follow) | ~20 | ~6 | ~26 |
| STREAM (monitor) | 1 | ~25 | ~26 |
| STREAM (stats) | 0 | ~6 | ~6 |
| POLL | ~30 | ~40 | ~70 |
| QUERY | ~35 | ~60 | ~95 |
| MUTATION | ~90 | ~120 | ~210 |
| **Total** | **~176** | **~257** | **~433** |
