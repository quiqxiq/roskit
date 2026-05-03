package parser_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamSentence_HotspotActive(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_active"}
	pairs := map[string]string{
		".id": "*A1", "user": "user001", "address": "192.168.88.100",
		"mac-address": "AA:BB:CC:DD:EE:FF", "server": "hotspot1",
		"uptime": "30m", "session-time-left": "1h30m",
		"keepalive-timeout": "10s", "login-by": "cookie",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*A1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "user001", result.CacheData["user"])
	assert.Equal(t, "192.168.88.100", result.CacheData["address"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["mac_address"])
	assert.Equal(t, "hotspot1", result.CacheData["server"])
	assert.Equal(t, "30m", result.CacheData["uptime"])
	assert.Equal(t, "1h30m", result.CacheData["session_time_left"])
	assert.Equal(t, "10s", result.CacheData["keepalive_timeout"])
	assert.Equal(t, "cookie", result.CacheData["login_by"])
}

func TestParseStreamSentence_HotspotProfile(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_profile"}
	pairs := map[string]string{
		".id": "*P1", "name": "default", "address-pool": "pool1",
		"rate-limit": "10M/10M", "shared-users": "2",
		"status-autorefresh": "1m", "on-login": ":put (\"hello\")",
		"on-logout": "", "parent-queue": "none",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*P1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "default", result.CacheData["name"])
	assert.Equal(t, "pool1", result.CacheData["address_pool"])
	assert.Equal(t, "10M/10M", result.CacheData["rate_limit"])
	assert.Equal(t, "2", result.CacheData["shared_users"])
	assert.Equal(t, "1m", result.CacheData["status_autorefresh"])
	assert.Equal(t, ":put (\"hello\")", result.CacheData["on_login"])
	assert.Equal(t, "none", result.CacheData["parent_queue"])
}

func TestParseStreamSentence_HotspotServer(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_server"}
	pairs := map[string]string{
		".id": "*S1", "name": "hs1", "interface": "ether2",
		"disabled": "false",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*S1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "hs1", result.CacheData["name"])
	assert.Equal(t, "ether2", result.CacheData["interface"])
	assert.Equal(t, "false", result.CacheData["disabled"])
}

func TestParseStreamSentence_HotspotHost(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_host"}
	pairs := map[string]string{
		".id": "*H1", "mac-address": "11:22:33:44:55:66",
		"address": "10.0.0.5", "server": "hs1",
		"to-address": "10.5.5.5", "authorized": "true",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*H1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "11:22:33:44:55:66", result.CacheData["mac"])
	assert.Equal(t, "10.0.0.5", result.CacheData["address"])
	assert.Equal(t, "hs1", result.CacheData["server"])
	assert.Equal(t, "10.5.5.5", result.CacheData["to_address"])
	assert.Equal(t, "true", result.CacheData["authorized"])
}

func TestParseStreamSentence_HotspotCookie(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_cookie"}
	pairs := map[string]string{
		".id": "*C1", "user": "admin", "mac-address": "AA:BB:CC:DD:EE:FF",
		"address": "192.168.1.50",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*C1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "admin", result.CacheData["user"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["mac"])
	assert.Equal(t, "192.168.1.50", result.CacheData["address"])
}

func TestParseStreamSentence_IPBinding(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "ip_binding"}
	pairs := map[string]string{
		".id": "*B1", "mac-address": "AA:BB:CC:DD:EE:FF",
		"address": "192.168.88.200", "type": "bypassed",
		"disabled": "false", "comment": "allowed device",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*B1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.CacheData["mac"])
	assert.Equal(t, "192.168.88.200", result.CacheData["address"])
	assert.Equal(t, "bypassed", result.CacheData["type"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "allowed device", result.CacheData["comment"])
}

func TestParseStreamSentence_WalledGarden(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "walled_garden"}
	pairs := map[string]string{
		".id": "*W1", "dst-address": "example.com", "dst-port": "443",
		"src-address": "", "protocol": "tcp", "action": "allow",
		"comment": "https access", "disabled": "false", "server": "all",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*W1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "example.com", result.CacheData["dst_address"])
	assert.Equal(t, "443", result.CacheData["dst_port"])
	assert.Equal(t, "tcp", result.CacheData["protocol"])
	assert.Equal(t, "allow", result.CacheData["action"])
	assert.Equal(t, "https access", result.CacheData["comment"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "all", result.CacheData["server"])
}

func TestParseStreamSentence_SystemResourceStream(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_resource"}
	pairs := map[string]string{
		"uptime": "3d5h", "version": "7.15",
		"architecture-name": "arm64", "board-name": "CCR2004",
		"cpu": "AL32400", "cpu-load": "8",
		"free-memory": "209715200", "total-memory": "1073741824",
		"free-hdd-space": "67108864", "total-hdd-space": "268435456",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "singleton", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "3d5h", result.CacheData["uptime"])
	assert.Equal(t, "7.15", result.CacheData["version"])
	assert.Equal(t, "arm64", result.CacheData["architecture_name"])
	assert.Equal(t, "CCR2004", result.CacheData["board_name"])
	assert.Equal(t, "8", result.CacheData["cpu_load"])
	assert.Equal(t, "209715200", result.CacheData["free_memory"])
	assert.Equal(t, "1073741824", result.CacheData["total_memory"])
}
