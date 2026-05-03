package model_test

import (
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInterfaceStatsFromReply(t *testing.T) {
	data := map[string]string{
		"name":                 "ether1",
		"rx-bits-per-second":   "1000000",
		"tx-bits-per-second":   "2000000",
		"rx-packets-per-second": "1500",
		"tx-packets-per-second": "2500",
		"rx-errors":            "10",
		"tx-errors":            "5",
		"rx-drops":             "3",
		"tx-drops":             "2",
		"link-downs":           "1",
	}
	s := model.NewInterfaceStatsFromReply(data)
	require.NotNil(t, s)
	assert.Equal(t, "ether1", s.Name)
	assert.Equal(t, uint64(1000000), s.RXBitsPerSecond)
	assert.Equal(t, uint64(2000000), s.TXBitsPerSecond)
	assert.Equal(t, uint64(1500), s.RXPacketsPerSecond)
	assert.Equal(t, uint64(2500), s.TXPacketsPerSecond)
	assert.Equal(t, uint64(10), s.RXErrors)
	assert.Equal(t, uint64(5), s.TXErrors)
	assert.Equal(t, uint64(3), s.RXDrops)
	assert.Equal(t, uint64(2), s.TXDrops)
	assert.Equal(t, uint64(1), s.LinkDowns)
	assert.False(t, s.Timestamp.IsZero())
}

func TestInterfaceStats_ToCacheData(t *testing.T) {
	data := map[string]string{
		"name":               "ether1",
		"rx-bits-per-second": "500000",
		"tx-bits-per-second": "800000",
		"link-downs":         "0",
	}
	s := model.NewInterfaceStatsFromReply(data)
	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	cache := s.ToCacheData(ts)
	assert.Equal(t, "ether1", cache["name"])
	assert.Equal(t, "500000", cache["rx_bits_per_second"])
	assert.Equal(t, "800000", cache["tx_bits_per_second"])
	assert.Equal(t, "0", cache["link_downs"])
	assert.Equal(t, "2024-06-01T12:00:00Z", cache["timestamp"])
}

func TestNewInterfaceFromReply(t *testing.T) {
	data := map[string]string{
		".id":        "*1",
		"name":       "ether1",
		"type":       "ethernet",
		"mtu":        "1500",
		"running":    "true",
		"disabled":   "false",
		"comment":    "uplink",
		"rx-byte":    "1048576",
		"tx-byte":    "2097152",
		"rx-packet":  "5000",
		"tx-packet":  "6000",
		"link-downs": "2",
	}
	iface := model.NewInterfaceFromReply(data)
	require.NotNil(t, iface)
	assert.Equal(t, "*1", iface.ID)
	assert.Equal(t, "ether1", iface.Name)
	assert.Equal(t, "ethernet", iface.Type)
	assert.Equal(t, "1500", iface.MTU)
	assert.True(t, iface.Running)
	assert.False(t, iface.Disabled)
	assert.Equal(t, "uplink", iface.Comment)
	assert.Equal(t, uint64(1048576), iface.RXByte)
	assert.Equal(t, uint64(2097152), iface.TXByte)
	assert.Equal(t, uint64(5000), iface.RXPacket)
	assert.Equal(t, uint64(6000), iface.TXPacket)
	assert.Equal(t, uint64(2), iface.LinkDowns)
}

func TestInterface_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":       "*2",
		"name":      "ether2",
		"type":      "ethernet",
		"mtu":       "1500",
		"running":   "yes",
		"disabled":  "no",
		"rx-byte":   "100",
		"tx-byte":   "200",
		"link-downs": "0",
	}
	iface := model.NewInterfaceFromReply(data)
	ts := time.Date(2024, 1, 15, 8, 30, 0, 0, time.UTC)
	cache := iface.ToCacheData(ts)
	assert.Equal(t, "*2", cache["id"])
	assert.Equal(t, "ether2", cache["name"])
	assert.Equal(t, "ethernet", cache["type"])
	assert.Equal(t, "1500", cache["mtu"])
	assert.Equal(t, "true", cache["running"])
	assert.Equal(t, "false", cache["disabled"])
	assert.Equal(t, "100", cache["rx_byte"])
	assert.Equal(t, "200", cache["tx_byte"])
	assert.Equal(t, "2024-01-15T08:30:00Z", cache["timestamp"])
}

func TestNewDHCPLeaseFromReply(t *testing.T) {
	data := map[string]string{
		".id":          "*1A",
		"address":      "192.168.88.100",
		"mac-address":  "AA:BB:CC:DD:EE:FF",
		"client-id":    "ff:aa:bb",
		"server":       "dhcp1",
		"lease-time":   "1d",
		"comment":      "printer",
		"disabled":     "false",
		"block-access": "true",
		"rate-limit":   "10M/10M",
		"dynamic":      "yes",
		"host-name":    "printer-hp",
		"status":       "bound",
	}
	lease := model.NewDHCPLeaseFromReply(data)
	require.NotNil(t, lease)
	assert.Equal(t, "*1A", lease.ID)
	assert.Equal(t, "192.168.88.100", lease.Address)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", lease.MACAddress)
	assert.Equal(t, "ff:aa:bb", lease.ClientID)
	assert.Equal(t, "dhcp1", lease.Server)
	assert.Equal(t, "1d", lease.LeaseTime)
	assert.Equal(t, "printer", lease.Comment)
	assert.False(t, lease.Disabled)
	assert.True(t, lease.BlockAccess)
	assert.Equal(t, "10M/10M", lease.RateLimit)
	assert.True(t, lease.Dynamic)
	assert.Equal(t, "printer-hp", lease.Hostname)
	assert.Equal(t, "bound", lease.Status)
}

func TestDHCPLease_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":         "*1B",
		"address":     "10.0.0.50",
		"mac-address": "11:22:33:44:55:66",
		"server":      "dhcp2",
		"disabled":    "true",
		"dynamic":     "false",
	}
	lease := model.NewDHCPLeaseFromReply(data)
	ts := time.Date(2024, 3, 20, 14, 0, 0, 0, time.UTC)
	cache := lease.ToCacheData(ts)
	assert.Equal(t, "*1B", cache["id"])
	assert.Equal(t, "10.0.0.50", cache["address"])
	assert.Equal(t, "11:22:33:44:55:66", cache["mac_address"])
	assert.Equal(t, "dhcp2", cache["server"])
	assert.Equal(t, "true", cache["disabled"])
	assert.Equal(t, "false", cache["dynamic"])
	assert.Equal(t, "2024-03-20T14:00:00Z", cache["timestamp"])
}

func TestDHCPLease_ToTags(t *testing.T) {
	data := map[string]string{
		".id":         "*1",
		"address":     "192.168.1.10",
		"mac-address": "AA:BB:CC:11:22:33",
		"server":      "dhcp1",
	}
	lease := model.NewDHCPLeaseFromReply(data)
	tags := lease.ToTags()
	assert.Equal(t, "192.168.1.10", tags["address"])
	assert.Equal(t, "AA:BB:CC:11:22:33", tags["mac_address"])
	assert.Equal(t, "dhcp1", tags["server"])
}

func TestDHCPLease_ToFields(t *testing.T) {
	data := map[string]string{
		".id":          "*1",
		"client-id":    "ff:00:01",
		"lease-time":   "3h",
		"disabled":     "yes",
		"block-access": "true",
		"rate-limit":   "5M/5M",
		"comment":      "laptop",
	}
	lease := model.NewDHCPLeaseFromReply(data)
	fields := lease.ToFields()
	assert.Equal(t, "ff:00:01", fields["client_id"])
	assert.Equal(t, "3h", fields["lease_time"])
	assert.Equal(t, true, fields["disabled"])
	assert.Equal(t, true, fields["block_access"])
	assert.Equal(t, "5M/5M", fields["rate_limit"])
	assert.Equal(t, "laptop", fields["comment"])
}

func TestNewPPPSecretFromReply(t *testing.T) {
	data := map[string]string{
		".id":           "*S1",
		"name":          "ppp-user1",
		"password":      "secret123",
		"profile":       "default",
		"service":       "pppoe",
		"caller-id":     "AA:BB:CC:DD:EE:FF",
		"local-address": "10.0.0.1",
		"remote-address": "10.0.0.100",
		"disabled":      "false",
		"comment":       "main user",
	}
	secret := model.NewPPPSecretFromReply(data)
	require.NotNil(t, secret)
	assert.Equal(t, "*S1", secret.ID)
	assert.Equal(t, "ppp-user1", secret.Name)
	assert.Equal(t, "secret123", secret.Password)
	assert.Equal(t, "default", secret.Profile)
	assert.Equal(t, "pppoe", secret.Service)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", secret.CallerID)
	assert.Equal(t, "10.0.0.1", secret.LocalAddress)
	assert.Equal(t, "10.0.0.100", secret.RemoteAddress)
	assert.False(t, secret.Disabled)
	assert.Equal(t, "main user", secret.Comment)
}

func TestPPPSecret_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":      "*S2",
		"name":     "ppp-user2",
		"disabled": "yes",
	}
	secret := model.NewPPPSecretFromReply(data)
	ts := time.Date(2024, 5, 10, 9, 0, 0, 0, time.UTC)
	cache := secret.ToCacheData(ts)
	assert.Equal(t, "*S2", cache["id"])
	assert.Equal(t, "ppp-user2", cache["name"])
	assert.Equal(t, "true", cache["disabled"])
	assert.Equal(t, "2024-05-10T09:00:00Z", cache["timestamp"])
}

func TestPPPSecret_ToTags(t *testing.T) {
	data := map[string]string{
		".id":     "*S1",
		"name":    "user-a",
		"profile": "premium",
		"service": "pppoe",
	}
	secret := model.NewPPPSecretFromReply(data)
	tags := secret.ToTags()
	assert.Equal(t, "user-a", tags["name"])
	assert.Equal(t, "premium", tags["profile"])
	assert.Equal(t, "pppoe", tags["service"])
}

func TestPPPSecret_ToFields(t *testing.T) {
	data := map[string]string{
		".id":            "*S1",
		"caller-id":      "11:22:33:44:55:66",
		"local-address":  "172.16.0.1",
		"remote-address": "172.16.0.2",
		"disabled":       "true",
		"comment":        "vip user",
	}
	secret := model.NewPPPSecretFromReply(data)
	fields := secret.ToFields()
	assert.Equal(t, true, fields["disabled"])
	assert.Equal(t, "11:22:33:44:55:66", fields["caller_id"])
	assert.Equal(t, "172.16.0.1", fields["local_address"])
	assert.Equal(t, "172.16.0.2", fields["remote_address"])
	assert.Equal(t, "vip user", fields["comment"])
}

func TestNewPPPActiveFromReply(t *testing.T) {
	data := map[string]string{
		".id":             "*A1",
		"name":            "ppp-user1",
		"service":         "pppoe",
		"caller-id":       "AA:BB:CC:DD:EE:FF",
		"address":         "10.0.0.100",
		"uptime":          "1h30m",
		"encoding":        "MPPE128",
		"session-id":      "0x12345678",
		"limit-bytes-in":  "1073741824",
		"limit-bytes-out": "2147483648",
	}
	active := model.NewPPPActiveFromReply(data)
	require.NotNil(t, active)
	assert.Equal(t, "*A1", active.ID)
	assert.Equal(t, "ppp-user1", active.Name)
	assert.Equal(t, "pppoe", active.Service)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", active.CallerID)
	assert.Equal(t, "10.0.0.100", active.Address)
	assert.Equal(t, "1h30m", active.Uptime)
	assert.Equal(t, "MPPE128", active.Encoding)
	assert.Equal(t, "0x12345678", active.SessionID)
	assert.Equal(t, uint64(1073741824), active.LimitBytesIn)
	assert.Equal(t, uint64(2147483648), active.LimitBytesOut)
}

func TestPPPActive_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":             "*A2",
		"name":            "ppp-user2",
		"service":         "l2tp",
		"address":         "10.1.1.50",
		"limit-bytes-in":  "0",
		"limit-bytes-out": "0",
	}
	active := model.NewPPPActiveFromReply(data)
	ts := time.Date(2024, 7, 4, 16, 30, 0, 0, time.UTC)
	cache := active.ToCacheData(ts)
	assert.Equal(t, "*A2", cache["id"])
	assert.Equal(t, "ppp-user2", cache["name"])
	assert.Equal(t, "l2tp", cache["service"])
	assert.Equal(t, "10.1.1.50", cache["address"])
	assert.Equal(t, "0", cache["limit_bytes_in"])
	assert.Equal(t, "0", cache["limit_bytes_out"])
	assert.Equal(t, "2024-07-04T16:30:00Z", cache["timestamp"])
}

func TestPPPActive_ToTags(t *testing.T) {
	data := map[string]string{
		".id":     "*A1",
		"name":    "active-user",
		"service": "pptp",
	}
	active := model.NewPPPActiveFromReply(data)
	tags := active.ToTags()
	assert.Equal(t, "active-user", tags["name"])
	assert.Equal(t, "pptp", tags["service"])
}

func TestPPPActive_ToFields(t *testing.T) {
	data := map[string]string{
		".id":             "*A1",
		"caller-id":       "AA:BB:CC:DD:EE:FF",
		"address":         "10.0.0.100",
		"uptime":          "2h",
		"encoding":        "none",
		"session-id":      "0xABCD",
		"limit-bytes-in":  "5000",
		"limit-bytes-out": "6000",
	}
	active := model.NewPPPActiveFromReply(data)
	fields := active.ToFields()
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", fields["caller_id"])
	assert.Equal(t, "10.0.0.100", fields["address"])
	assert.Equal(t, "2h", fields["uptime"])
	assert.Equal(t, "none", fields["encoding"])
	assert.Equal(t, "0xABCD", fields["session_id"])
	assert.Equal(t, uint64(5000), fields["limit_bytes_in"])
	assert.Equal(t, uint64(6000), fields["limit_bytes_out"])
}

func TestNewARPEntryFromReply(t *testing.T) {
	data := map[string]string{
		".id":        "*1",
		"address":    "192.168.88.50",
		"mac-address": "AA:BB:CC:11:22:33",
		"interface":  "ether1",
		"status":     "reachable",
		"complete":   "true",
		"disabled":   "false",
		"dynamic":    "yes",
		"comment":    "server",
	}
	entry := model.NewARPEntryFromReply(data)
	require.NotNil(t, entry)
	assert.Equal(t, "*1", entry.ID)
	assert.Equal(t, "192.168.88.50", entry.Address)
	assert.Equal(t, "AA:BB:CC:11:22:33", entry.MACAddress)
	assert.Equal(t, "ether1", entry.Interface)
	assert.Equal(t, "reachable", entry.Status)
	assert.True(t, entry.Complete)
	assert.False(t, entry.Disabled)
	assert.True(t, entry.Dynamic)
	assert.Equal(t, "server", entry.Comment)
}

func TestARPEntry_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":        "*2",
		"address":    "10.0.0.1",
		"mac-address": "11:22:33:44:55:66",
		"interface":  "bridge1",
		"complete":   "yes",
		"disabled":   "yes",
		"dynamic":    "false",
	}
	entry := model.NewARPEntryFromReply(data)
	ts := time.Date(2024, 2, 28, 20, 0, 0, 0, time.UTC)
	cache := entry.ToCacheData(ts)
	assert.Equal(t, "*2", cache["id"])
	assert.Equal(t, "10.0.0.1", cache["address"])
	assert.Equal(t, "11:22:33:44:55:66", cache["mac_address"])
	assert.Equal(t, "bridge1", cache["interface"])
	assert.Equal(t, "true", cache["complete"])
	assert.Equal(t, "true", cache["disabled"])
	assert.Equal(t, "false", cache["dynamic"])
	assert.Equal(t, "2024-02-28T20:00:00Z", cache["timestamp"])
}

func TestNewIPPoolFromReply(t *testing.T) {
	data := map[string]string{
		".id":       "*1",
		"name":      "hs-pool",
		"ranges":    "192.168.88.100-192.168.88.200",
		"next-pool": "none",
	}
	pool := model.NewIPPoolFromReply(data)
	require.NotNil(t, pool)
	assert.Equal(t, "*1", pool.ID)
	assert.Equal(t, "hs-pool", pool.Name)
	assert.Equal(t, "192.168.88.100-192.168.88.200", pool.Ranges)
	assert.Equal(t, "none", pool.NextPool)
}

func TestIPPool_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":    "*2",
		"name":   "ppp-pool",
		"ranges": "10.0.0.1-10.0.0.254",
	}
	pool := model.NewIPPoolFromReply(data)
	ts := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	cache := pool.ToCacheData(ts)
	assert.Equal(t, "*2", cache["id"])
	assert.Equal(t, "ppp-pool", cache["name"])
	assert.Equal(t, "10.0.0.1-10.0.0.254", cache["ranges"])
	assert.Equal(t, "", cache["next_pool"])
	assert.Equal(t, "2024-04-01T00:00:00Z", cache["timestamp"])
}

func TestNewFirewallNATFromReply(t *testing.T) {
	data := map[string]string{
		".id":           "*N1",
		"chain":         "dstnat",
		"action":        "dst-nat",
		"protocol":      "tcp",
		"dst-address":   "1.2.3.4",
		"dst-port":      "80",
		"src-address":   "0.0.0.0/0",
		"src-port":      "",
		"to-addresses":  "192.168.88.100",
		"to-ports":      "8080",
		"comment":       "web redirect",
		"disabled":      "false",
		"in-interface":  "ether1",
		"out-interface": "ether2",
		"log":           "yes",
		"log-prefix":    "nat:",
	}
	nat := model.NewFirewallNATFromReply(data)
	require.NotNil(t, nat)
	assert.Equal(t, "*N1", nat.ID)
	assert.Equal(t, "dstnat", nat.Chain)
	assert.Equal(t, "dst-nat", nat.Action)
	assert.Equal(t, "tcp", nat.Protocol)
	assert.Equal(t, "1.2.3.4", nat.DstAddress)
	assert.Equal(t, "80", nat.DstPort)
	assert.Equal(t, "0.0.0.0/0", nat.SrcAddress)
	assert.Equal(t, "", nat.SrcPort)
	assert.Equal(t, "192.168.88.100", nat.ToAddresses)
	assert.Equal(t, "8080", nat.ToPorts)
	assert.Equal(t, "web redirect", nat.Comment)
	assert.False(t, nat.Disabled)
	assert.Equal(t, "ether1", nat.InInterface)
	assert.Equal(t, "ether2", nat.OutInterface)
	assert.True(t, nat.Log)
	assert.Equal(t, "nat:", nat.LogPrefix)
}

func TestFirewallNAT_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":      "*N2",
		"chain":    "srcnat",
		"action":   "masquerade",
		"disabled": "yes",
		"log":      "false",
	}
	nat := model.NewFirewallNATFromReply(data)
	ts := time.Date(2024, 8, 15, 12, 0, 0, 0, time.UTC)
	cache := nat.ToCacheData(ts)
	assert.Equal(t, "*N2", cache["id"])
	assert.Equal(t, "srcnat", cache["chain"])
	assert.Equal(t, "masquerade", cache["action"])
	assert.Equal(t, "true", cache["disabled"])
	assert.Equal(t, "false", cache["log"])
	assert.Equal(t, "2024-08-15T12:00:00Z", cache["timestamp"])
}

func TestNewQueueSimpleFromReply(t *testing.T) {
	data := map[string]string{
		".id":             "*Q1",
		"name":            "queue1",
		"target":          "192.168.88.100/32",
		"parent":          "none",
		"packet-mark":     "voip",
		"rate-limit":      "10M/10M",
		"max-limit":       "50M/50M",
		"burst-limit":     "100M/100M",
		"burst-threshold": "30M/30M",
		"burst-time":      "10s/10s",
		"priority":        "8",
		"queue":           "pcq-upload-default/pcq-download-default",
		"comment":         "voip priority",
		"disabled":        "false",
		"dynamic":         "yes",
	}
	q := model.NewQueueSimpleFromReply(data)
	require.NotNil(t, q)
	assert.Equal(t, "*Q1", q.ID)
	assert.Equal(t, "queue1", q.Name)
	assert.Equal(t, "192.168.88.100/32", q.Target)
	assert.Equal(t, "none", q.Parent)
	assert.Equal(t, "voip", q.PacketMark)
	assert.Equal(t, "10M/10M", q.RateLimit)
	assert.Equal(t, "50M/50M", q.MaxLimit)
	assert.Equal(t, "100M/100M", q.BurstLimit)
	assert.Equal(t, "30M/30M", q.BurstThreshold)
	assert.Equal(t, "10s/10s", q.BurstTime)
	assert.Equal(t, "8", q.Priority)
	assert.Equal(t, "pcq-upload-default/pcq-download-default", q.Queue)
	assert.Equal(t, "voip priority", q.Comment)
	assert.False(t, q.Disabled)
	assert.True(t, q.Dynamic)
}

func TestQueueSimple_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id":       "*Q2",
		"name":      "queue2",
		"target":    "10.0.0.1/32",
		"max-limit": "20M/20M",
		"disabled":  "yes",
		"dynamic":   "false",
	}
	q := model.NewQueueSimpleFromReply(data)
	ts := time.Date(2024, 9, 1, 6, 0, 0, 0, time.UTC)
	cache := q.ToCacheData(ts)
	assert.Equal(t, "*Q2", cache["id"])
	assert.Equal(t, "queue2", cache["name"])
	assert.Equal(t, "10.0.0.1/32", cache["target"])
	assert.Equal(t, "20M/20M", cache["max_limit"])
	assert.Equal(t, "true", cache["disabled"])
	assert.Equal(t, "false", cache["dynamic"])
	assert.Equal(t, "2024-09-01T06:00:00Z", cache["timestamp"])
}

func TestToTagsToFields_Table(t *testing.T) {
	tests := []struct {
		name    string
		build   func() (tags map[string]string, fields map[string]interface{})
		tagKeys []string
	}{
		{
			name: "DHCPLease",
			build: func() (map[string]string, map[string]interface{}) {
				d := model.NewDHCPLeaseFromReply(map[string]string{
					".id": "*1", "address": "10.0.0.5", "mac-address": "AA:BB:CC:DD:EE:FF",
					"server": "dhcp1", "client-id": "cid1", "disabled": "true", "comment": "test",
				})
				return d.ToTags(), d.ToFields()
			},
			tagKeys: []string{"address", "mac_address", "server"},
		},
		{
			name: "PPPSecret",
			build: func() (map[string]string, map[string]interface{}) {
				s := model.NewPPPSecretFromReply(map[string]string{
					".id": "*1", "name": "user1", "profile": "prof1", "service": "pppoe",
					"disabled": "yes", "caller-id": "AA:BB", "comment": "secret",
				})
				return s.ToTags(), s.ToFields()
			},
			tagKeys: []string{"name", "profile", "service"},
		},
		{
			name: "PPPActive",
			build: func() (map[string]string, map[string]interface{}) {
				a := model.NewPPPActiveFromReply(map[string]string{
					".id": "*1", "name": "active1", "service": "l2tp",
					"address": "10.0.0.1", "uptime": "5m", "caller-id": "CC:DD",
				})
				return a.ToTags(), a.ToFields()
			},
			tagKeys: []string{"name", "service"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags, fields := tt.build()
			for _, key := range tt.tagKeys {
				assert.Contains(t, tags, key, "tags should contain %s", key)
			}
			assert.NotEmpty(t, fields, "fields should not be empty")
		})
	}
}
