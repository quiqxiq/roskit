// Package testhelpers provides shared fixtures and helpers for roskit tests.
package testhelpers

// RouterOS wire-format reply maps for all major entity types.
// Use these as input to New*FromReply constructors in tests.

var HotspotUserReply = map[string]string{
	".id":               "*1",
	"name":              "user001",
	"password":          "pass001",
	"profile":           "2hours",
	"server":            "hotspot1",
	"mac-address":       "AA:BB:CC:DD:EE:FF",
	"limit-uptime":      "2h",
	"limit-bytes-total": "0",
	"comment":           "test user",
	"disabled":          "false",
	"uptime":            "1h30m",
	"bytes-in":          "1048576",
	"bytes-out":         "2097152",
}

var HotspotActiveReply = map[string]string{
	".id":               "*A1",
	"user":              "user001",
	"address":           "192.168.88.100",
	"mac-address":       "AA:BB:CC:DD:EE:FF",
	"server":            "hotspot1",
	"uptime":            "30m",
	"session-time-left": "1h30m",
	"keepalive-timeout": "10s",
	"login-by":          "cookie",
}

var HotspotProfileReply = map[string]string{
	".id":                "*P1",
	"name":               "2hours",
	"address-pool":       "hs-pool",
	"rate-limit":         "1M/1M",
	"shared-users":       "1",
	"status-autorefresh": "1m",
	"on-login":           "script",
	"on-logout":          "",
	"parent-queue":       "",
}

var HotspotWalledGardenReply = map[string]string{
	".id":         "*W1",
	"dst-address": "8.8.8.8",
	"dst-port":    "80",
	"src-address": "",
	"protocol":    "tcp",
	"action":      "allow",
	"comment":     "google dns",
	"disabled":    "false",
	"server":      "hotspot1",
}

var SystemResourceReply = map[string]string{
	"uptime":            "1d2h3m",
	"version":           "7.14.3 (stable)",
	"architecture-name": "arm",
	"board-name":        "RB750Gr3",
	"cpu":               "ARM",
	"cpu-load":          "12",
	"free-memory":       "104857600",
	"total-memory":      "268435456",
	"free-hdd-space":    "52428800",
	"total-hdd-space":   "134217728",
}

var HotspotServerReply = map[string]string{
	".id": "*S1", "name": "hotspot1", "interface": "ether1", "disabled": "false",
}

var HotspotHostReply = map[string]string{
	".id": "*H1", "mac-address": "AA:BB:CC:DD:EE:FF", "address": "192.168.88.100",
	"server": "hotspot1", "to-address": "10.5.50.1", "authorized": "true",
}

var HotspotCookieReply = map[string]string{
	".id": "*C1", "user": "user001", "mac-address": "AA:BB:CC:DD:EE:FF", "address": "192.168.88.100",
}

var IPBindingReply = map[string]string{
	".id": "*B1", "mac-address": "AA:BB:CC:DD:EE:FF", "address": "192.168.88.100",
	"type": "bypassed", "disabled": "false", "comment": "test binding",
}

var SystemSchedulerReply = map[string]string{
	".id": "*1", "name": "test-scheduler", "start-date": "jan/01/2024",
	"start-time": "00:00:00", "interval": "1d", "next-run": "jan/02/2024 00:00:00",
	"on-event": ":put test", "disabled": "false", "comment": "test",
	"owner": "admin", "policy": "read,write",
}

var SystemScriptReply = map[string]string{
	".id": "*2", "name": "test-script", "source": ":put hello",
	"owner": "admin", "comment": "mikhmon", "policy": "read,write",
	"last-started": "jan/01/2024 10:00:00", "run-count": "5",
}

var SystemIdentityReply = map[string]string{
	"name": "MikroTik-GW",
}

var SystemClockReply = map[string]string{
	"time": "10:30:00", "date": "jan/15/2024", "time-zone-name": "Asia/Jakarta", "dst-active": "false",
}

var SystemHealthReply = map[string]string{
	"voltage": "24.5", "temperature": "42", "cpu-temperature": "48",
	"power-consumption": "5", "board-temperature": "40",
}

var SystemRouterboardReply = map[string]string{
	"model": "RB750Gr3", "serial-number": "ABC123", "firmware-type": "factory",
	"firmware": "7.14.3", "revision": "r2", "upgrade-firmware": "7.15",
	"routerboard": "true",
}

var InterfaceStatsReply = map[string]string{
	"name": "ether1", "rx-bits-per-second": "1000000", "tx-bits-per-second": "2000000",
	"rx-packets-per-second": "500", "tx-packets-per-second": "600",
	"rx-errors": "1", "tx-errors": "0", "rx-drops": "2", "tx-drops": "0", "link-downs": "3",
}

var InterfaceReply = map[string]string{
	".id": "*1", "name": "ether1", "type": "ether", "mtu": "1500",
	"running": "true", "disabled": "false", "comment": "WAN",
	"rx-byte": "1048576", "tx-byte": "2097152", "rx-packet": "1000", "tx-packet": "2000", "link-downs": "0",
}

var DHCPLeaseReply = map[string]string{
	".id": "*L1", "address": "192.168.88.100", "mac-address": "AA:BB:CC:DD:EE:FF",
	"client-id": "ff:aa:bb", "server": "dhcp1", "lease-time": "1d",
	"comment": "test", "disabled": "false", "block-access": "false",
	"rate-limit": "1M/1M", "dynamic": "false", "host-name": "test-pc", "status": "bound",
}

var PPPSecretReply = map[string]string{
	".id": "*P1", "name": "ppp-user1", "password": "secret123", "profile": "default",
	"service": "pppoe", "caller-id": "AA:BB:CC:DD:EE:FF", "local-address": "10.0.0.1",
	"remote-address": "10.0.0.2", "disabled": "false", "comment": "test ppp",
}

var PPPActiveReply = map[string]string{
	".id": "*PA1", "name": "ppp-user1", "service": "pppoe", "caller-id": "AA:BB:CC:DD:EE:FF",
	"address": "10.0.0.2", "uptime": "1h30m", "encoding": "MPPE128",
	"session-id": "ABC123", "limit-bytes-in": "1048576", "limit-bytes-out": "2097152",
}

var ARPEntryReply = map[string]string{
	".id": "*A1", "address": "192.168.88.100", "mac-address": "AA:BB:CC:DD:EE:FF",
	"interface": "ether1", "status": "reachable", "complete": "true",
	"disabled": "false", "dynamic": "false", "comment": "test arp",
}

var IPPoolReply = map[string]string{
	".id": "*IP1", "name": "hs-pool", "ranges": "192.168.88.100-192.168.88.200", "next-pool": "none",
}

var FirewallNATReply = map[string]string{
	".id": "*N1", "chain": "dstnat", "action": "dst-nat", "protocol": "tcp",
	"dst-address": "0.0.0.0", "dst-port": "80", "src-address": "", "src-port": "",
	"to-addresses": "192.168.88.1", "to-ports": "8080", "comment": "redirect",
	"disabled": "false", "in-interface": "ether1", "out-interface": "",
	"log": "false", "log-prefix": "",
}

var QueueSimpleReply = map[string]string{
	".id": "*Q1", "name": "queue1", "target": "192.168.88.100/32", "parent": "none",
	"packet-mark": "", "rate-limit": "1M/1M", "max-limit": "2M/2M",
	"burst-limit": "", "burst-threshold": "", "burst-time": "",
	"priority": "8", "queue": "pcq-upload-default/pcq-download-default",
	"comment": "test queue", "disabled": "false", "dynamic": "false",
}
