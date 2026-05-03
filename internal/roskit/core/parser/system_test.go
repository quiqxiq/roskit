package parser_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamSentence_SystemScheduler(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_scheduler"}
	pairs := map[string]string{
		".id": "*SC1", "name": "daily-backup",
		"start-date": "jan/01/2024", "start-time": "03:00:00",
		"interval": "24h", "next-run": "03:00:00",
		"on-event": "/system backup save", "disabled": "false",
		"comment": "auto backup", "owner": "admin", "policy": "read,write",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*SC1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "daily-backup", result.CacheData["name"])
	assert.Equal(t, "jan/01/2024", result.CacheData["start_date"])
	assert.Equal(t, "03:00:00", result.CacheData["start_time"])
	assert.Equal(t, "24h", result.CacheData["interval"])
	assert.Equal(t, "03:00:00", result.CacheData["next_run"])
	assert.Equal(t, "/system backup save", result.CacheData["on_event"])
	assert.Equal(t, "false", result.CacheData["disabled"])
	assert.Equal(t, "auto backup", result.CacheData["comment"])
	assert.Equal(t, "admin", result.CacheData["owner"])
	assert.Equal(t, "read,write", result.CacheData["policy"])
}

func TestParseStreamSentence_SystemScript(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_script"}
	pairs := map[string]string{
		".id": "*SS1", "name": "sales-script",
		"source": ":put \"hello\"", "owner": "Jan2024",
		"comment": "mikhmon", "policy": "read,write",
		"last-started": "jan/15/2024 10:30:00", "run-count": "42",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*SS1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "sales-script", result.CacheData["name"])
	assert.Equal(t, ":put \"hello\"", result.CacheData["source"])
	assert.Equal(t, "Jan2024", result.CacheData["owner"])
	assert.Equal(t, "mikhmon", result.CacheData["comment"])
	assert.Equal(t, "read,write", result.CacheData["policy"])
	assert.Equal(t, "42", result.CacheData["run_count"])
}

func TestParsePollReply_SystemIdentity(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_identity"}
	rows := []map[string]string{{"name": "MikroTik-GW"}}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
	assert.Equal(t, "MikroTik-GW", results[0].CacheData["name"])
}

func TestParsePollReply_SystemIdentity_Empty(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_identity"}
	results := parser.ParsePollReply("router1", meta, []map[string]string{})
	assert.Nil(t, results)
}

func TestParsePollReply_SystemClock(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_clock"}
	rows := []map[string]string{{
		"time": "14:30:00", "date": "jan/15/2024",
		"time-zone-name": "Asia/Jakarta", "dst-active": "false",
	}}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
	assert.Equal(t, "14:30:00", results[0].CacheData["time"])
	assert.Equal(t, "jan/15/2024", results[0].CacheData["date"])
	assert.Equal(t, "Asia/Jakarta", results[0].CacheData["time_zone_name"])
	assert.Equal(t, "false", results[0].CacheData["dst_active"])
}

func TestParsePollReply_SystemClock_Empty(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_clock"}
	results := parser.ParsePollReply("router1", meta, []map[string]string{})
	assert.Nil(t, results)
}

func TestParsePollReply_SystemHealth(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_health"}
	rows := []map[string]string{{
		"voltage": "24.5", "temperature": "42",
		"cpu-temperature": "55", "power-consumption": "12.3",
		"board-temperature": "38",
	}}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
	assert.Equal(t, "24.5", results[0].CacheData["voltage"])
	assert.Equal(t, "42", results[0].CacheData["temperature"])
	assert.Equal(t, "55", results[0].CacheData["cpu_temperature"])
	assert.Equal(t, "12.3", results[0].CacheData["power_consumption"])
	assert.Equal(t, "38", results[0].CacheData["board_temperature"])
}

func TestParsePollReply_SystemHealth_Empty(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_health"}
	results := parser.ParsePollReply("router1", meta, []map[string]string{})
	assert.Nil(t, results)
}

func TestParsePollReply_SystemRouterboard(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_routerboard"}
	rows := []map[string]string{{
		"model": "RB750Gr3", "serial-number": "ABC12345",
		"firmware-type": "factory", "firmware": "6.49.10",
		"revision": "r3", "upgrade-firmware": "7.15",
		"routerboard": "true",
	}}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
	assert.Equal(t, "RB750Gr3", results[0].CacheData["model"])
	assert.Equal(t, "ABC12345", results[0].CacheData["serial_number"])
	assert.Equal(t, "factory", results[0].CacheData["firmware_type"])
	assert.Equal(t, "6.49.10", results[0].CacheData["firmware"])
	assert.Equal(t, "r3", results[0].CacheData["revision"])
	assert.Equal(t, "7.15", results[0].CacheData["upgrade_firmware"])
	assert.Equal(t, "true", results[0].CacheData["routerboard"])
}

func TestParsePollReply_SystemRouterboard_Empty(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_routerboard"}
	results := parser.ParsePollReply("router1", meta, []map[string]string{})
	assert.Nil(t, results)
}


