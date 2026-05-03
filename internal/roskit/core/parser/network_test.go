package parser_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamSentence_Interface(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "interface"}
	pairs := map[string]string{
		".id": "*IF1", "name": "ether1", "type": "ether",
		"mtu": "1500", "running": "true", "disabled": "false",
		"comment": "uplink", "rx-byte": "1048576", "tx-byte": "2097152",
		"rx-packet": "10000", "tx-packet": "20000", "link-downs": "0",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*IF1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "ether1", result.CacheData["name"])
	assert.Equal(t, "ether", result.CacheData["type"])
	assert.Equal(t, "1500", result.CacheData["mtu"])
	assert.Equal(t, "true", result.CacheData["running"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "uplink", result.CacheData["comment"])
	assert.Equal(t, "1048576", result.CacheData["rx_byte"])
	assert.Equal(t, "2097152", result.CacheData["tx_byte"])
	assert.Equal(t, "10000", result.CacheData["rx_packet"])
	assert.Equal(t, "20000", result.CacheData["tx_packet"])
	assert.Equal(t, "0", result.CacheData["link_downs"])
}

func TestParseStreamSentence_ARP(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "arp"}
	pairs := map[string]string{
		".id": "*AR1", "address": "192.168.88.100",
		"mac-address": "AA:BB:CC:DD:EE:FF", "interface": "ether2",
		"status": "reachable", "complete": "true",
		"disabled": "false", "dynamic": "true", "comment": "",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*AR1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "192.168.88.100", result.CacheData["address"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["mac_address"])
	assert.Equal(t, "ether2", result.CacheData["interface"])
	assert.Equal(t, "reachable", result.CacheData["status"])
	assert.Equal(t, "true", result.CacheData["complete"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "true", result.CacheData["dynamic"])
}

func TestParseStreamSentence_DHCPLease(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "dhcp_lease"}
	pairs := map[string]string{
		".id": "*DL1", "address": "192.168.88.150",
		"mac-address": "11:22:33:44:55:66", "client-id": "ff:11.22.33.44.55.66",
		"server": "dhcp1", "lease-time": "1d",
		"comment": "laptop", "disabled": "false", "block-access": "false",
		"rate-limit": "10M/10M", "dynamic": "true", "host-name": "MyPC",
		"status": "bound",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*DL1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "192.168.88.150", result.CacheData["address"])
	assert.Equal(t, "11:22:33:44:55:66", result.CacheData["mac_address"])
	assert.Equal(t, "ff:11.22.33.44.55.66", result.CacheData["client_id"])
	assert.Equal(t, "dhcp1", result.CacheData["server"])
	assert.Equal(t, "1d", result.CacheData["lease_time"])
	assert.Equal(t, "laptop", result.CacheData["comment"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "false", result.CacheData["block_access"])
	assert.Equal(t, "10M/10M", result.CacheData["rate_limit"])
	assert.Equal(t, "true", result.CacheData["dynamic"])
	assert.Equal(t, "MyPC", result.CacheData["host_name"])
	assert.Equal(t, "bound", result.CacheData["status"])
}

func TestParseStreamSentence_IPPool(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ip_pool"}
	pairs := map[string]string{
		".id": "*IP1", "name": "hs-pool",
		"ranges": "192.168.88.100-192.168.88.200", "next-pool": "none",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*IP1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "hs-pool", result.CacheData["name"])
	assert.Equal(t, "192.168.88.100-192.168.88.200", result.CacheData["ranges"])
	assert.Equal(t, "none", result.CacheData["next_pool"])
}

func TestParseStreamSentence_FirewallNAT(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "firewall_nat"}
	pairs := map[string]string{
		".id": "*FN1", "chain": "dstnat", "action": "dst-nat",
		"protocol": "tcp", "dst-address": "", "dst-port": "8080",
		"src-address": "", "src-port": "",
		"to-addresses": "192.168.88.10", "to-ports": "80",
		"comment": "web redirect", "disabled": "false",
		"in-interface": "ether1", "out-interface": "",
		"log": "false", "log-prefix": "",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*FN1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "dstnat", result.CacheData["chain"])
	assert.Equal(t, "dst-nat", result.CacheData["action"])
	assert.Equal(t, "tcp", result.CacheData["protocol"])
	assert.Equal(t, "8080", result.CacheData["dst_port"])
	assert.Equal(t, "192.168.88.10", result.CacheData["to_addresses"])
	assert.Equal(t, "80", result.CacheData["to_ports"])
	assert.Equal(t, "web redirect", result.CacheData["comment"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "ether1", result.CacheData["in_interface"])
	assert.Equal(t, "false", result.CacheData["log"])
}

func TestParseStreamSentence_QueueSimple(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "queue_simple"}
	pairs := map[string]string{
		".id": "*QS1", "name": "queue1", "target": "192.168.88.100/32",
		"parent": "none", "packet-mark": "",
		"rate-limit": "10M/10M", "max-limit": "20M/20M",
		"burst-limit": "30M/30M", "burst-threshold": "15M/15M",
		"burst-time": "10s/10s", "priority": "8",
		"queue": "pcq-upload-default/pcq-download-default",
		"comment": "user queue", "disabled": "false", "dynamic": "false",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*QS1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "queue1", result.CacheData["name"])
	assert.Equal(t, "192.168.88.100/32", result.CacheData["target"])
	assert.Equal(t, "none", result.CacheData["parent"])
	assert.Equal(t, "10M/10M", result.CacheData["rate_limit"])
	assert.Equal(t, "20M/20M", result.CacheData["max_limit"])
	assert.Equal(t, "30M/30M", result.CacheData["burst_limit"])
	assert.Equal(t, "15M/15M", result.CacheData["burst_threshold"])
	assert.Equal(t, "10s/10s", result.CacheData["burst_time"])
	assert.Equal(t, "8", result.CacheData["priority"])
	assert.Equal(t, "pcq-upload-default/pcq-download-default", result.CacheData["queue"])
	assert.Equal(t, "user queue", result.CacheData["comment"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "false", result.CacheData["dynamic"])
}
