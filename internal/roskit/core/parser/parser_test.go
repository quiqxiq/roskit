package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUint64(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"0", 0},
		{"1048576", 1048576},
		{"18446744073709551615", 18446744073709551615}, // max uint64
		{"", 0},
		{"invalid", 0},
		{"-1", 0},
		{"1.5", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parser.ParseUint64(tt.input))
		})
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"100", 100},
		{"-100", -100},
		{"9223372036854775807", 9223372036854775807}, // max int64
		{"", 0},
		{"invalid", 0},
		{"1.5", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parser.ParseInt64(tt.input))
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"yes", true},
		{"TRUE", true},
		{"YES", true},
		{"True", true},
		{"false", false},
		{"no", false},
		{"FALSE", false},
		{"", false},
		{"0", false},
		{"1", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parser.ParseBool(tt.input))
		})
	}
}

func TestParseMikroTikTime(t *testing.T) {
	t.Run("jan/02/2006 format", func(t *testing.T) {
		result := parser.ParseMikroTikTime("jan/02/2006 15:04:05")
		assert.False(t, result.IsZero())
		assert.Equal(t, 2006, result.Year())
	})

	t.Run("Jan/02/2006 format capitalised", func(t *testing.T) {
		result := parser.ParseMikroTikTime("Jan/02/2006 15:04:05")
		assert.False(t, result.IsZero())
	})

	t.Run("2006-01-02 15:04:05 format", func(t *testing.T) {
		result := parser.ParseMikroTikTime("2006-01-02 15:04:05")
		assert.False(t, result.IsZero())
		assert.Equal(t, 2006, result.Year())
	})

	t.Run("RFC3339 format", func(t *testing.T) {
		result := parser.ParseMikroTikTime("2024-03-15T10:30:00Z")
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.March, result.Month())
	})

	t.Run("empty string returns now", func(t *testing.T) {
		before := time.Now().Add(-time.Second)
		result := parser.ParseMikroTikTime("")
		assert.True(t, result.After(before))
	})

	t.Run("whitespace-only string", func(t *testing.T) {
		before := time.Now().Add(-time.Second)
		result := parser.ParseMikroTikTime("   ")
		assert.True(t, result.After(before))
	})
}

func TestFormatBool(t *testing.T) {
	assert.Equal(t, "true", parser.FormatBool(true))
	assert.Equal(t, "false", parser.FormatBool(false))
}

func TestFormatUint64(t *testing.T) {
	assert.Equal(t, "0", parser.FormatUint64(0))
	assert.Equal(t, "1048576", parser.FormatUint64(1048576))
}

func TestFormatInt64(t *testing.T) {
	assert.Equal(t, "0", parser.FormatInt64(0))
	assert.Equal(t, "-100", parser.FormatInt64(-100))
	assert.Equal(t, "100", parser.FormatInt64(100))
}

func TestIsDead(t *testing.T) {
	assert.True(t, parser.IsDead(map[string]string{".dead": "true"}))
	assert.True(t, parser.IsDead(map[string]string{".dead": "yes"}))
	assert.False(t, parser.IsDead(map[string]string{".dead": "false"}))
	assert.False(t, parser.IsDead(map[string]string{}))
	assert.False(t, parser.IsDead(map[string]string{"name": "user"}))
}

func TestParseStreamSentence_EmptyPairs(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_user"}
	result := parser.ParseStreamSentence("router1", meta, map[string]string{})
	assert.Nil(t, result)
}

func TestParseStreamSentence_DeadEntry(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_user"}
	pairs := map[string]string{".id": "*1", ".dead": "true"}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.True(t, result.IsDead)
	assert.Equal(t, "*1", result.EntityID)
}

func TestParseStreamSentence_DeadWithoutID(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_user"}
	pairs := map[string]string{".dead": "true"}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	assert.Nil(t, result, "dead entry without .id should return nil")
}

func TestParseStreamSentence_GenericFallback(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "unknown_measurement"}
	pairs := map[string]string{".id": "*99", "name": "something"}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*99", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "something", result.CacheData["name"])
}

func TestParseStreamSentence_GenericFallback_NoID(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "unknown_measurement"}
	pairs := map[string]string{"name": "something"}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	assert.Nil(t, result, "generic fallback without .id should return nil")
}

func TestParseStreamSentence_HotspotUser(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "hotspot_user"}
	pairs := map[string]string{
		".id":               "*1",
		"name":              "user001",
		"password":          "pass001",
		"profile":           "2hours",
		"server":            "hotspot1",
		"mac-address":       "AA:BB:CC:DD:EE:FF",
		"limit-uptime":      "2h",
		"limit-bytes-total": "0",
		"disabled":          "false",
		"uptime":            "1h",
		"bytes-in":          "0",
		"bytes-out":         "0",
	}
	result := parser.ParseStreamSentence("router1", meta, pairs)
	require.NotNil(t, result)
	assert.Equal(t, "*1", result.EntityID)
	assert.False(t, result.IsDead)
	assert.Equal(t, "user001", result.CacheData["name"])
}

func TestParsePollReply_EmptyRows(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_resource"}
	result := parser.ParsePollReply("router1", meta, []map[string]string{})
	assert.Nil(t, result)
}

func TestParsePollReply_GenericFallback(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "unknown_poll"}
	rows := []map[string]string{
		{".id": "*1", "name": "alpha"},
		{".id": "*2", "name": "beta"},
	}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 2)
	assert.Equal(t, "*1", results[0].EntityID)
	assert.Equal(t, "*2", results[1].EntityID)
}

func TestParsePollReply_GenericFallback_NoID(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "singleton_poll"}
	rows := []map[string]string{
		{"uptime": "1d", "version": "7.14"},
	}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
}

func TestParsePollReply_SystemResource(t *testing.T) {
	meta := &command.CommandMeta{Measurement: "system_resource"}
	rows := []map[string]string{
		{
			"uptime":            "1d2h3m",
			"version":           "7.14.3",
			"architecture-name": "arm",
			"board-name":        "RB750Gr3",
			"cpu":               "ARM",
			"cpu-load":          "12",
			"free-memory":       "104857600",
			"total-memory":      "268435456",
			"free-hdd-space":    "52428800",
			"total-hdd-space":   "134217728",
		},
	}
	results := parser.ParsePollReply("router1", meta, rows)
	require.Len(t, results, 1)
	assert.Equal(t, "singleton", results[0].EntityID)
	assert.True(t, strings.Contains(results[0].CacheData["uptime"], "1d"))
}
